// Package volume implements the volume API request handlers.
package volume

import (
	"context"
	"log/slog"

	oapigen "github.com/dcm-project/k8s-storage-service-provider/internal/api/server"
	"github.com/dcm-project/k8s-storage-service-provider/internal/store"
)

// Handler implements volume CRUD operations for the OpenAPI StrictServerInterface.
type Handler struct {
	store  store.VolumeRepository
	logger *slog.Logger
}

// NewHandler creates a Handler backed by the given repository.
func NewHandler(repo store.VolumeRepository, logger *slog.Logger) *Handler {
	return &Handler{
		store:  repo,
		logger: logger,
	}
}

const volumesBasePath = "/api/v1alpha1/volumes"

func (h *Handler) CreateVolume(ctx context.Context, req oapigen.CreateVolumeRequestObject) (oapigen.CreateVolumeResponseObject, error) {
	requestPath := volumesBasePath

	if req.Body == nil {
		return newCreateError400("request body is required", requestPath), nil
	}

	spec := req.Body.Spec

	var id string
	if req.Params.Id != nil {
		id = *req.Params.Id
	} else {
		generated, err := generateVolumeID()
		if err != nil {
			h.logger.Error("failed to generate volume id", "error", err)
			return h.mapCreateError(err, requestPath), nil
		}
		id = generated
	}

	if err := validateVolumeID(id); err != nil {
		return newCreateError400(err.Error(), requestPath), nil
	}
	if err := validateCreateSpec(spec); err != nil {
		return newCreateError400(err.Error(), requestPath), nil
	}
	if err := validateUserLabels(spec.Metadata.Labels); err != nil {
		return newCreateError400(err.Error(), requestPath), nil
	}

	result, err := h.store.Create(ctx, spec, id)
	if err != nil {
		return h.mapCreateError(err, requestPath), nil
	}
	return oapigen.CreateVolume201JSONResponse(*result), nil
}

func (h *Handler) GetVolume(ctx context.Context, req oapigen.GetVolumeRequestObject) (oapigen.GetVolumeResponseObject, error) {
	requestPath := volumesBasePath + "/" + req.VolumeId
	result, err := h.store.Get(ctx, req.VolumeId)
	if err != nil {
		return h.mapGetError(err, requestPath), nil
	}
	return oapigen.GetVolume200JSONResponse(*result), nil
}

func (h *Handler) DeleteVolume(ctx context.Context, req oapigen.DeleteVolumeRequestObject) (oapigen.DeleteVolumeResponseObject, error) {
	requestPath := volumesBasePath + "/" + req.VolumeId
	if err := h.store.Delete(ctx, req.VolumeId); err != nil {
		return h.mapDeleteError(err, requestPath), nil
	}
	return oapigen.DeleteVolume204Response{}, nil
}

func (h *Handler) ListVolumes(ctx context.Context, req oapigen.ListVolumesRequestObject) (oapigen.ListVolumesResponseObject, error) {
	var maxPageSize int32
	if req.Params.MaxPageSize != nil {
		maxPageSize = *req.Params.MaxPageSize
	}

	var pageToken string
	if req.Params.PageToken != nil {
		pageToken = *req.Params.PageToken
	}

	result, err := h.store.List(ctx, maxPageSize, pageToken)
	if err != nil {
		return h.mapListError(err, volumesBasePath), nil
	}
	return oapigen.ListVolumes200JSONResponse(*result), nil
}
