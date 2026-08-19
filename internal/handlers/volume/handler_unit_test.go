package volume_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	v1alpha1 "github.com/dcm-project/k8s-storage-service-provider/api/v1alpha1"
	oapigen "github.com/dcm-project/k8s-storage-service-provider/internal/api/server"
	"github.com/dcm-project/k8s-storage-service-provider/internal/dcm"
	"github.com/dcm-project/k8s-storage-service-provider/internal/handlers/volume"
	"github.com/dcm-project/k8s-storage-service-provider/internal/store"
	"github.com/dcm-project/k8s-storage-service-provider/internal/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVolumeHandlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Volume Handlers Suite")
}

type mockVolumeRepository struct {
	CreateFunc      func(ctx context.Context, spec v1alpha1.StorageSpec, id string) (*v1alpha1.Volume, error)
	GetFunc         func(ctx context.Context, volumeID string) (*v1alpha1.Volume, error)
	ListFunc        func(ctx context.Context, maxPageSize int32, pageToken string) (*v1alpha1.VolumeList, error)
	DeleteFunc      func(ctx context.Context, volumeID string) error
	CheckHealthFunc func(ctx context.Context) error
}

func (m *mockVolumeRepository) Create(ctx context.Context, spec v1alpha1.StorageSpec, id string) (*v1alpha1.Volume, error) {
	if m.CreateFunc == nil {
		return nil, nil
	}
	return m.CreateFunc(ctx, spec, id)
}

func (m *mockVolumeRepository) Get(ctx context.Context, volumeID string) (*v1alpha1.Volume, error) {
	if m.GetFunc == nil {
		return nil, nil
	}
	return m.GetFunc(ctx, volumeID)
}

func (m *mockVolumeRepository) List(ctx context.Context, maxPageSize int32, pageToken string) (*v1alpha1.VolumeList, error) {
	if m.ListFunc == nil {
		return &v1alpha1.VolumeList{Volumes: &[]v1alpha1.Volume{}}, nil
	}
	return m.ListFunc(ctx, maxPageSize, pageToken)
}

func (m *mockVolumeRepository) Delete(ctx context.Context, volumeID string) error {
	if m.DeleteFunc == nil {
		return nil
	}
	return m.DeleteFunc(ctx, volumeID)
}

func (m *mockVolumeRepository) CheckHealth(ctx context.Context) error {
	if m.CheckHealthFunc == nil {
		return nil
	}
	return m.CheckHealthFunc(ctx)
}

func validCreateSpec() v1alpha1.StorageSpec {
	return v1alpha1.StorageSpec{
		ServiceType: v1alpha1.Storage,
		Capacity:    "10Gi",
		Metadata:    v1alpha1.VolumeMetadata{Name: "app-data"},
	}
}

func newVolumeResult(spec v1alpha1.StorageSpec, id string) *v1alpha1.Volume {
	status := v1alpha1.PROVISIONING
	path := "volumes/" + id
	now := time.Now().UTC()
	ns := "default"
	spec.Metadata.Namespace = &ns
	return &v1alpha1.Volume{
		Id:         &id,
		Path:       &path,
		Spec:       spec,
		Status:     &status,
		CreateTime: &now,
		UpdateTime: &now,
	}
}

var _ = Describe("Volume API Handlers", func() {
	var (
		repo *mockVolumeRepository
		h    *volume.Handler
	)

	BeforeEach(func() {
		repo = &mockVolumeRepository{}
		h = volume.NewHandler(repo, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	})

	Describe("CreateVolume", func() {
		It("returns 201 on success (TC-U060)", func() {
			repo.CreateFunc = func(_ context.Context, spec v1alpha1.StorageSpec, id string) (*v1alpha1.Volume, error) {
				return newVolumeResult(spec, id), nil
			}

			resp, err := h.CreateVolume(context.Background(), oapigen.CreateVolumeRequestObject{
				Body: &v1alpha1.Volume{Spec: validCreateSpec()},
			})
			Expect(err).NotTo(HaveOccurred())
			created, ok := resp.(oapigen.CreateVolume201JSONResponse)
			Expect(ok).To(BeTrue())
			Expect(created.Status).NotTo(BeNil())
			Expect(*created.Status).To(Equal(v1alpha1.PROVISIONING))
			Expect(created.Id).NotTo(BeNil())
		})

		It("uses client-specified id", func() {
			var capturedID string
			repo.CreateFunc = func(_ context.Context, spec v1alpha1.StorageSpec, id string) (*v1alpha1.Volume, error) {
				capturedID = id
				return newVolumeResult(spec, id), nil
			}

			resp, err := h.CreateVolume(context.Background(), oapigen.CreateVolumeRequestObject{
				Params: v1alpha1.CreateVolumeParams{Id: util.Ptr("app-data-volume")},
				Body:   &v1alpha1.Volume{Spec: validCreateSpec()},
			})
			Expect(err).NotTo(HaveOccurred())
			_, ok := resp.(oapigen.CreateVolume201JSONResponse)
			Expect(ok).To(BeTrue())
			Expect(capturedID).To(Equal("app-data-volume"))
		})

		It("generates an AEP-122 ID when no id query param (REQ-API-030)", func() {
			var capturedID string
			repo.CreateFunc = func(_ context.Context, spec v1alpha1.StorageSpec, id string) (*v1alpha1.Volume, error) {
				capturedID = id
				return newVolumeResult(spec, id), nil
			}

			resp, err := h.CreateVolume(context.Background(), oapigen.CreateVolumeRequestObject{
				Body: &v1alpha1.Volume{Spec: validCreateSpec()},
			})
			Expect(err).NotTo(HaveOccurred())
			created, ok := resp.(oapigen.CreateVolume201JSONResponse)
			Expect(ok).To(BeTrue())
			Expect(capturedID).NotTo(BeEmpty())
			Expect(capturedID).To(HaveLen(26))
			Expect(capturedID).To(MatchRegexp(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`))
			Expect(capturedID).NotTo(MatchRegexp(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`))
			Expect(created.Id).NotTo(BeNil())
			Expect(*created.Id).To(Equal(capturedID))
		})

		It("returns 409 on conflict (TC-U061)", func() {
			repo.CreateFunc = func(_ context.Context, _ v1alpha1.StorageSpec, _ string) (*v1alpha1.Volume, error) {
				return nil, &store.ConflictError{Message: "PVC already exists"}
			}

			resp, err := h.CreateVolume(context.Background(), oapigen.CreateVolumeRequestObject{
				Body: &v1alpha1.Volume{Spec: validCreateSpec()},
			})
			Expect(err).NotTo(HaveOccurred())
			errResp, ok := resp.(oapigen.CreateVolume409ApplicationProblemPlusJSONResponse)
			Expect(ok).To(BeTrue())
			Expect(errResp.Type).To(Equal(v1alpha1.ALREADYEXISTS))
			Expect(errResp.Status).NotTo(BeNil())
			Expect(*errResp.Status).To(Equal(int32(409)))
		})

		It("returns 422 when StorageClass is missing", func() {
			repo.CreateFunc = func(_ context.Context, _ v1alpha1.StorageSpec, _ string) (*v1alpha1.Volume, error) {
				return nil, &store.FailedPreconditionError{Message: `StorageClass "missing" does not exist`}
			}

			resp, err := h.CreateVolume(context.Background(), oapigen.CreateVolumeRequestObject{
				Body: &v1alpha1.Volume{Spec: validCreateSpec()},
			})
			Expect(err).NotTo(HaveOccurred())
			errResp, ok := resp.(oapigen.CreateVolume422ApplicationProblemPlusJSONResponse)
			Expect(ok).To(BeTrue())
			Expect(errResp.Type).To(Equal(v1alpha1.FAILEDPRECONDITION))
		})

		It("returns 400 when capacity is missing (TC-U062)", func() {
			spec := validCreateSpec()
			spec.Capacity = ""

			resp, err := h.CreateVolume(context.Background(), oapigen.CreateVolumeRequestObject{
				Body: &v1alpha1.Volume{Spec: spec},
			})
			Expect(err).NotTo(HaveOccurred())
			errResp, ok := resp.(oapigen.CreateVolume400ApplicationProblemPlusJSONResponse)
			Expect(ok).To(BeTrue())
			Expect(errResp.Type).To(Equal(v1alpha1.INVALIDARGUMENT))
		})

		It("rejects reserved health volume ID", func() {
			resp, err := h.CreateVolume(context.Background(), oapigen.CreateVolumeRequestObject{
				Params: v1alpha1.CreateVolumeParams{Id: util.Ptr("health")},
				Body:   &v1alpha1.Volume{Spec: validCreateSpec()},
			})
			Expect(err).NotTo(HaveOccurred())
			errResp, ok := resp.(oapigen.CreateVolume400ApplicationProblemPlusJSONResponse)
			Expect(ok).To(BeTrue())
			Expect(*errResp.Detail).To(ContainSubstring("reserved"))
		})

		It("rejects reserved DCM labels", func() {
			spec := validCreateSpec()
			spec.Metadata.Labels = &map[string]string{dcm.LabelManagedBy: "user"}

			resp, err := h.CreateVolume(context.Background(), oapigen.CreateVolumeRequestObject{
				Body: &v1alpha1.Volume{Spec: spec},
			})
			Expect(err).NotTo(HaveOccurred())
			errResp, ok := resp.(oapigen.CreateVolume400ApplicationProblemPlusJSONResponse)
			Expect(ok).To(BeTrue())
			Expect(errResp.Type).To(Equal(v1alpha1.INVALIDARGUMENT))
		})

		It("rejects wrong service_type", func() {
			spec := validCreateSpec()
			spec.ServiceType = "container"

			resp, err := h.CreateVolume(context.Background(), oapigen.CreateVolumeRequestObject{
				Body: &v1alpha1.Volume{Spec: spec},
			})
			Expect(err).NotTo(HaveOccurred())
			_, ok := resp.(oapigen.CreateVolume400ApplicationProblemPlusJSONResponse)
			Expect(ok).To(BeTrue())
		})
	})

	Describe("GetVolume", func() {
		It("returns 200 when found (TC-U065)", func() {
			repo.GetFunc = func(_ context.Context, id string) (*v1alpha1.Volume, error) {
				return newVolumeResult(validCreateSpec(), id), nil
			}

			resp, err := h.GetVolume(context.Background(), oapigen.GetVolumeRequestObject{VolumeId: "vol-1"})
			Expect(err).NotTo(HaveOccurred())
			got, ok := resp.(oapigen.GetVolume200JSONResponse)
			Expect(ok).To(BeTrue())
			Expect(*got.Id).To(Equal("vol-1"))
		})

		It("returns 404 when missing (TC-U066)", func() {
			repo.GetFunc = func(_ context.Context, id string) (*v1alpha1.Volume, error) {
				return nil, &store.NotFoundError{ID: id}
			}

			resp, err := h.GetVolume(context.Background(), oapigen.GetVolumeRequestObject{VolumeId: "missing"})
			Expect(err).NotTo(HaveOccurred())
			errResp, ok := resp.(oapigen.GetVolume404ApplicationProblemPlusJSONResponse)
			Expect(ok).To(BeTrue())
			Expect(errResp.Type).To(Equal(v1alpha1.NOTFOUND))
			Expect(errResp.Status).NotTo(BeNil())
			Expect(*errResp.Status).To(Equal(int32(404)))
		})

		It("returns 422 on ambiguous instance ID (AC-K8S-170)", func() {
			repo.GetFunc = func(_ context.Context, id string) (*v1alpha1.Volume, error) {
				return nil, &store.ConflictError{Message: "multiple PVCs found for volume " + id}
			}

			resp, err := h.GetVolume(context.Background(), oapigen.GetVolumeRequestObject{VolumeId: "dup"})
			Expect(err).NotTo(HaveOccurred())
			errResp, ok := resp.(oapigen.GetVolume422ApplicationProblemPlusJSONResponse)
			Expect(ok).To(BeTrue())
			Expect(errResp.Type).To(Equal(v1alpha1.FAILEDPRECONDITION))
			Expect(errResp.Status).NotTo(BeNil())
			Expect(*errResp.Status).To(Equal(int32(422)))
		})
	})

	Describe("ListVolumes", func() {
		It("returns 200 with volumes (TC-U063)", func() {
			vols := []v1alpha1.Volume{*newVolumeResult(validCreateSpec(), "a")}
			repo.ListFunc = func(_ context.Context, _ int32, _ string) (*v1alpha1.VolumeList, error) {
				return &v1alpha1.VolumeList{Volumes: &vols}, nil
			}

			resp, err := h.ListVolumes(context.Background(), oapigen.ListVolumesRequestObject{})
			Expect(err).NotTo(HaveOccurred())
			list, ok := resp.(oapigen.ListVolumes200JSONResponse)
			Expect(ok).To(BeTrue())
			Expect(*list.Volumes).To(HaveLen(1))
		})

		It("returns 400 for invalid page token", func() {
			repo.ListFunc = func(_ context.Context, _ int32, _ string) (*v1alpha1.VolumeList, error) {
				return nil, &store.InvalidArgumentError{Message: "invalid page_token"}
			}

			resp, err := h.ListVolumes(context.Background(), oapigen.ListVolumesRequestObject{
				Params: v1alpha1.ListVolumesParams{PageToken: util.Ptr("bad")},
			})
			Expect(err).NotTo(HaveOccurred())
			errResp, ok := resp.(oapigen.ListVolumes400ApplicationProblemPlusJSONResponse)
			Expect(ok).To(BeTrue())
			Expect(errResp.Type).To(Equal(v1alpha1.INVALIDARGUMENT))
		})
	})

	Describe("DeleteVolume", func() {
		It("returns 204 on success (TC-U069)", func() {
			repo.DeleteFunc = func(_ context.Context, _ string) error { return nil }

			resp, err := h.DeleteVolume(context.Background(), oapigen.DeleteVolumeRequestObject{VolumeId: "vol-1"})
			Expect(err).NotTo(HaveOccurred())
			_, ok := resp.(oapigen.DeleteVolume204Response)
			Expect(ok).To(BeTrue())
		})

		It("returns 404 when missing (TC-U070)", func() {
			repo.DeleteFunc = func(_ context.Context, id string) error {
				return &store.NotFoundError{ID: id}
			}

			resp, err := h.DeleteVolume(context.Background(), oapigen.DeleteVolumeRequestObject{VolumeId: "missing"})
			Expect(err).NotTo(HaveOccurred())
			errResp, ok := resp.(oapigen.DeleteVolume404ApplicationProblemPlusJSONResponse)
			Expect(ok).To(BeTrue())
			Expect(errResp.Type).To(Equal(v1alpha1.NOTFOUND))
			Expect(errResp.Status).NotTo(BeNil())
			Expect(*errResp.Status).To(Equal(int32(404)))
		})

		It("returns 422 on ambiguous instance ID (AC-K8S-170)", func() {
			repo.DeleteFunc = func(_ context.Context, id string) error {
				return &store.ConflictError{Message: "multiple PVCs found for volume " + id}
			}

			resp, err := h.DeleteVolume(context.Background(), oapigen.DeleteVolumeRequestObject{VolumeId: "dup"})
			Expect(err).NotTo(HaveOccurred())
			errResp, ok := resp.(oapigen.DeleteVolume422ApplicationProblemPlusJSONResponse)
			Expect(ok).To(BeTrue())
			Expect(errResp.Type).To(Equal(v1alpha1.FAILEDPRECONDITION))
			Expect(errResp.Status).NotTo(BeNil())
			Expect(*errResp.Status).To(Equal(int32(422)))
		})
	})
})
