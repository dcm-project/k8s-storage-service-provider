// Package kubernetes implements Kubernetes-backed operations for the storage SP.
package kubernetes

import (
	"context"
	"log/slog"

	"github.com/dcm-project/k8s-storage-service-provider/internal/store"
	"k8s.io/client-go/kubernetes"
)

// K8sVolumeStore implements store.HealthChecker using the Kubernetes API.
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

var _ store.HealthChecker = (*K8sVolumeStore)(nil)

// CheckHealth verifies the backing Kubernetes cluster is reachable.
func (s *K8sVolumeStore) CheckHealth(_ context.Context) error {
	_, err := s.client.Discovery().ServerVersion()
	if err != nil {
		s.logger.Warn("kubernetes health check failed", "error", err)
		return err
	}
	return nil
}
