package kubernetes

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Delete removes a volume by deleting its PersistentVolumeClaim.
func (s *K8sVolumeStore) Delete(ctx context.Context, volumeID string) error {
	pvc, err := s.findPVC(ctx, volumeID)
	if err != nil {
		return err
	}
	err = s.client.CoreV1().PersistentVolumeClaims(s.cfg.Namespace).Delete(ctx, pvc.Name, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
