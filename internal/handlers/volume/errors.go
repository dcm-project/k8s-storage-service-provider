package volume

import (
	"errors"
	"net/http"

	v1alpha1 "github.com/dcm-project/k8s-storage-service-provider/api/v1alpha1"
	oapigen "github.com/dcm-project/k8s-storage-service-provider/internal/api/server"
	"github.com/dcm-project/k8s-storage-service-provider/internal/httperror"
	"github.com/dcm-project/k8s-storage-service-provider/internal/store"
	"github.com/dcm-project/k8s-storage-service-provider/internal/util"
)

func newCreateError400(detail, requestPath string) oapigen.CreateVolume400ApplicationProblemPlusJSONResponse {
	return oapigen.CreateVolume400ApplicationProblemPlusJSONResponse{
		Type:     v1alpha1.INVALIDARGUMENT,
		Title:    "Invalid argument",
		Status:   util.Ptr(int32(http.StatusBadRequest)),
		Detail:   &detail,
		Instance: &requestPath,
	}
}

func (h *Handler) mapCreateError(err error, requestPath string) oapigen.CreateVolumeResponseObject {
	if _, ok := errors.AsType[*store.ConflictError](err); ok {
		detail := err.Error()
		return oapigen.CreateVolume409ApplicationProblemPlusJSONResponse{
			Type:     v1alpha1.ALREADYEXISTS,
			Title:    "Already exists",
			Status:   util.Ptr(int32(http.StatusConflict)),
			Detail:   &detail,
			Instance: &requestPath,
		}
	}

	if _, ok := errors.AsType[*store.FailedPreconditionError](err); ok {
		detail := err.Error()
		return oapigen.CreateVolume422ApplicationProblemPlusJSONResponse{
			Type:     v1alpha1.FAILEDPRECONDITION,
			Title:    "Failed precondition",
			Status:   util.Ptr(int32(http.StatusUnprocessableEntity)),
			Detail:   &detail,
			Instance: &requestPath,
		}
	}

	if _, ok := errors.AsType[*store.InvalidArgumentError](err); ok {
		return newCreateError400(err.Error(), requestPath)
	}

	h.logger.Error("unexpected error in CreateVolume", "error", err)
	detail := httperror.InternalDetail
	return oapigen.CreateVolume500ApplicationProblemPlusJSONResponse{
		Type:     v1alpha1.INTERNAL,
		Title:    httperror.InternalTitle,
		Status:   util.Ptr(int32(http.StatusInternalServerError)),
		Detail:   &detail,
		Instance: &requestPath,
	}
}

func (h *Handler) mapGetError(err error, requestPath string) oapigen.GetVolumeResponseObject {
	if _, ok := errors.AsType[*store.NotFoundError](err); ok {
		detail := err.Error()
		return oapigen.GetVolume404ApplicationProblemPlusJSONResponse{
			Type:     v1alpha1.NOTFOUND,
			Title:    "Not found",
			Status:   util.Ptr(int32(http.StatusNotFound)),
			Detail:   &detail,
			Instance: &requestPath,
		}
	}

	var conflict *store.ConflictError
	if errors.As(err, &conflict) {
		detail := err.Error()
		return oapigen.GetVolume422ApplicationProblemPlusJSONResponse{
			Type:     v1alpha1.FAILEDPRECONDITION,
			Title:    "Failed precondition",
			Status:   util.Ptr(int32(http.StatusUnprocessableEntity)),
			Detail:   &detail,
			Instance: &requestPath,
		}
	}

	h.logger.Error("unexpected error in GetVolume", "error", err)
	detail := httperror.InternalDetail
	return oapigen.GetVolume500ApplicationProblemPlusJSONResponse{
		Type:     v1alpha1.INTERNAL,
		Title:    httperror.InternalTitle,
		Status:   util.Ptr(int32(http.StatusInternalServerError)),
		Detail:   &detail,
		Instance: &requestPath,
	}
}

func (h *Handler) mapDeleteError(err error, requestPath string) oapigen.DeleteVolumeResponseObject {
	if _, ok := errors.AsType[*store.NotFoundError](err); ok {
		detail := err.Error()
		return oapigen.DeleteVolume404ApplicationProblemPlusJSONResponse{
			Type:     v1alpha1.NOTFOUND,
			Title:    "Not found",
			Status:   util.Ptr(int32(http.StatusNotFound)),
			Detail:   &detail,
			Instance: &requestPath,
		}
	}

	var conflict *store.ConflictError
	if errors.As(err, &conflict) {
		detail := err.Error()
		return oapigen.DeleteVolume422ApplicationProblemPlusJSONResponse{
			Type:     v1alpha1.FAILEDPRECONDITION,
			Title:    "Failed precondition",
			Status:   util.Ptr(int32(http.StatusUnprocessableEntity)),
			Detail:   &detail,
			Instance: &requestPath,
		}
	}

	h.logger.Error("unexpected error in DeleteVolume", "error", err)
	detail := httperror.InternalDetail
	return oapigen.DeleteVolume500ApplicationProblemPlusJSONResponse{
		Type:     v1alpha1.INTERNAL,
		Title:    httperror.InternalTitle,
		Status:   util.Ptr(int32(http.StatusInternalServerError)),
		Detail:   &detail,
		Instance: &requestPath,
	}
}

func (h *Handler) mapListError(err error, requestPath string) oapigen.ListVolumesResponseObject {
	if _, ok := errors.AsType[*store.InvalidArgumentError](err); ok {
		detail := err.Error()
		return oapigen.ListVolumes400ApplicationProblemPlusJSONResponse{
			Type:     v1alpha1.INVALIDARGUMENT,
			Title:    "Invalid argument",
			Status:   util.Ptr(int32(http.StatusBadRequest)),
			Detail:   &detail,
			Instance: &requestPath,
		}
	}

	h.logger.Error("unexpected error in ListVolumes", "error", err)
	detail := httperror.InternalDetail
	return oapigen.ListVolumes500ApplicationProblemPlusJSONResponse{
		Type:     v1alpha1.INTERNAL,
		Title:    httperror.InternalTitle,
		Status:   util.Ptr(int32(http.StatusInternalServerError)),
		Detail:   &detail,
		Instance: &requestPath,
	}
}
