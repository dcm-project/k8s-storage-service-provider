package kubernetes

import (
	"context"
	"fmt"

	v1alpha1 "github.com/dcm-project/k8s-storage-service-provider/api/v1alpha1"
	"github.com/dcm-project/k8s-storage-service-provider/internal/store"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Create creates a new volume backed by a PersistentVolumeClaim.
func (s *K8sVolumeStore) Create(ctx context.Context, spec v1alpha1.StorageSpec, id string) (*v1alpha1.Volume, error) {
	labels := dcmLabels(id)
	if spec.Metadata.Labels != nil {
		labels = mergeLabels(labels, *spec.Metadata.Labels)
	}

	existingByID, err := s.client.CoreV1().PersistentVolumeClaims(s.cfg.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: instanceSelector(id),
	})
	if err != nil {
		return nil, err
	}
	if len(existingByID.Items) > 0 {
		return nil, &store.ConflictError{Message: fmt.Sprintf("volume with instance ID %q already exists", id)}
	}

	_, err = s.client.CoreV1().PersistentVolumeClaims(s.cfg.Namespace).Get(ctx, spec.Metadata.Name, metav1.GetOptions{})
	if err == nil {
		return nil, &store.ConflictError{Message: fmt.Sprintf("PVC %q already exists", spec.Metadata.Name)}
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	storageClass := resolveStorageClass(spec, s.cfg.DefaultStorageClass)
	if storageClass != "" {
		if err := s.validateStorageClass(ctx, storageClass); err != nil {
			return nil, err
		}
	}

	pvc, err := buildPVC(spec, s.cfg, labels)
	if err != nil {
		return nil, err
	}

	created, err := s.client.CoreV1().PersistentVolumeClaims(s.cfg.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, &store.ConflictError{Message: fmt.Sprintf("PVC %q already exists", spec.Metadata.Name)}
		}
		return nil, err
	}

	return s.buildVolume(created, id), nil
}

func (s *K8sVolumeStore) validateStorageClass(ctx context.Context, name string) error {
	_, err := s.client.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return &store.FailedPreconditionError{Message: fmt.Sprintf("StorageClass %q does not exist", name)}
		}
		return err
	}
	return nil
}
