// Package composite wires health and volume handlers into StrictServerInterface.
package composite

import (
	"context"

	oapigen "github.com/dcm-project/k8s-storage-service-provider/internal/api/server"
	"github.com/dcm-project/k8s-storage-service-provider/internal/handlers/health"
	"github.com/dcm-project/k8s-storage-service-provider/internal/handlers/volume"
)

// Handler implements StrictServerInterface by forwarding to resource handlers.
type Handler struct {
	health *health.Handler
	volume *volume.Handler
}

// NewHandler creates a composite StrictServerInterface multiplexer.
func NewHandler(healthHandler *health.Handler, volumeHandler *volume.Handler) *Handler {
	return &Handler{health: healthHandler, volume: volumeHandler}
}

var _ oapigen.StrictServerInterface = (*Handler)(nil)

func (h *Handler) GetHealth(ctx context.Context, req oapigen.GetHealthRequestObject) (oapigen.GetHealthResponseObject, error) {
	return h.health.GetHealth(ctx, req)
}

func (h *Handler) ListVolumes(ctx context.Context, req oapigen.ListVolumesRequestObject) (oapigen.ListVolumesResponseObject, error) {
	return h.volume.ListVolumes(ctx, req)
}

func (h *Handler) CreateVolume(ctx context.Context, req oapigen.CreateVolumeRequestObject) (oapigen.CreateVolumeResponseObject, error) {
	return h.volume.CreateVolume(ctx, req)
}

func (h *Handler) GetVolume(ctx context.Context, req oapigen.GetVolumeRequestObject) (oapigen.GetVolumeResponseObject, error) {
	return h.volume.GetVolume(ctx, req)
}

func (h *Handler) DeleteVolume(ctx context.Context, req oapigen.DeleteVolumeRequestObject) (oapigen.DeleteVolumeResponseObject, error) {
	return h.volume.DeleteVolume(ctx, req)
}
