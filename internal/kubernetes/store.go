// Package kubernetes implements Kubernetes-backed operations for the storage SP.
package kubernetes

import (
	"context"
	"fmt"
	"log/slog"

	v1alpha1 "github.com/dcm-project/k8s-storage-service-provider/api/v1alpha1"
	"github.com/dcm-project/k8s-storage-service-provider/internal/store"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// K8sVolumeStore implements store.VolumeRepository backed by PersistentVolumeClaims.
type K8sVolumeStore struct {
	client kubernetes.Interface
	cfg    K8sConfig
	logger *slog.Logger
}

// NewK8sVolumeStore creates a new K8sVolumeStore with the given client, config, and logger.
func NewK8sVolumeStore(client kubernetes.Interface, cfg K8sConfig, logger *slog.Logger) *K8sVolumeStore {
	return &K8sVolumeStore{
		client: client,
		cfg:    cfg,
		logger: logger,
	}
}

var _ store.VolumeRepository = (*K8sVolumeStore)(nil)

// CheckHealth verifies the backing Kubernetes cluster is reachable.
func (s *K8sVolumeStore) CheckHealth(_ context.Context) error {
	_, err := s.client.Discovery().ServerVersion()
	if err != nil {
		s.logger.Warn("kubernetes health check failed", "error", err)
		return err
	}
	return nil
}

func (s *K8sVolumeStore) findPVC(ctx context.Context, volumeID string) (*corev1.PersistentVolumeClaim, error) {
	pvcs, err := s.client.CoreV1().PersistentVolumeClaims(s.cfg.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: instanceSelector(volumeID),
	})
	if err != nil {
		return nil, err
	}
	if len(pvcs.Items) == 0 {
		return nil, &store.NotFoundError{ID: volumeID}
	}
	if len(pvcs.Items) > 1 {
		return nil, &store.ConflictError{Message: fmt.Sprintf("multiple PVCs found for volume %q", volumeID)}
	}
	return &pvcs.Items[0], nil
}

func (s *K8sVolumeStore) buildVolume(pvc *corev1.PersistentVolumeClaim, instanceID string) *v1alpha1.Volume {
	v := volumeFromPVC(pvc, instanceID)
	return &v
}
