// Package composite wires health and unimplemented volume handlers for incremental delivery.
package composite

import (
	"context"

	v1alpha1 "github.com/dcm-project/k8s-storage-service-provider/api/v1alpha1"
	oapigen "github.com/dcm-project/k8s-storage-service-provider/internal/api/server"
	"github.com/dcm-project/k8s-storage-service-provider/internal/handlers/health"
	"github.com/dcm-project/k8s-storage-service-provider/internal/httperror"
	"github.com/dcm-project/k8s-storage-service-provider/internal/util"
)

// Handler implements StrictServerInterface by delegating health to the health handler
// and returning not-implemented responses for volume operations.
type Handler struct {
	health *health.Handler
}

// NewHandler creates a composite handler for health and stub volume routes.
func NewHandler(healthHandler *health.Handler) *Handler {
	return &Handler{health: healthHandler}
}

var _ oapigen.StrictServerInterface = (*Handler)(nil)

func (h *Handler) GetHealth(ctx context.Context, req oapigen.GetHealthRequestObject) (oapigen.GetHealthResponseObject, error) {
	return h.health.GetHealth(ctx, req)
}

func notImplemented() v1alpha1.Error {
	return v1alpha1.Error{
		Type:   v1alpha1.INTERNAL,
		Title:  httperror.InternalTitle,
		Detail: util.Ptr("volume API not implemented"),
	}
}

func (h *Handler) ListVolumes(_ context.Context, _ oapigen.ListVolumesRequestObject) (oapigen.ListVolumesResponseObject, error) {
	err := notImplemented()
	return oapigen.ListVolumes500ApplicationProblemPlusJSONResponse(err), nil
}

func (h *Handler) CreateVolume(_ context.Context, _ oapigen.CreateVolumeRequestObject) (oapigen.CreateVolumeResponseObject, error) {
	err := notImplemented()
	return oapigen.CreateVolume500ApplicationProblemPlusJSONResponse(err), nil
}

func (h *Handler) GetVolume(_ context.Context, _ oapigen.GetVolumeRequestObject) (oapigen.GetVolumeResponseObject, error) {
	err := notImplemented()
	return oapigen.GetVolume500ApplicationProblemPlusJSONResponse(err), nil
}

func (h *Handler) UpdateVolume(_ context.Context, _ oapigen.UpdateVolumeRequestObject) (oapigen.UpdateVolumeResponseObject, error) {
	err := notImplemented()
	return oapigen.UpdateVolume500ApplicationProblemPlusJSONResponse(err), nil
}

func (h *Handler) DeleteVolume(_ context.Context, _ oapigen.DeleteVolumeRequestObject) (oapigen.DeleteVolumeResponseObject, error) {
	err := notImplemented()
	return oapigen.DeleteVolume500ApplicationProblemPlusJSONResponse(err), nil
}
