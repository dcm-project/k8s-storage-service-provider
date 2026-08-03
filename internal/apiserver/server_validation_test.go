package apiserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	v1alpha1 "github.com/dcm-project/k8s-storage-service-provider/api/v1alpha1"
	oapigen "github.com/dcm-project/k8s-storage-service-provider/internal/api/server"
	"github.com/dcm-project/k8s-storage-service-provider/internal/apiserver"
	"github.com/dcm-project/k8s-storage-service-provider/internal/config"
	"github.com/dcm-project/k8s-storage-service-provider/internal/handlers/composite"
	"github.com/dcm-project/k8s-storage-service-provider/internal/handlers/health"
	"github.com/dcm-project/k8s-storage-service-provider/internal/handlers/volume"
	"github.com/dcm-project/k8s-storage-service-provider/internal/store"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	validCreateBody = `{"spec":{"service_type":"storage","metadata":{"name":"test-vol"},"capacity":"10Gi"}}`
	volumesAPIPath  = "/api/v1alpha1/volumes"
)

func createBodyWithName(name string) string {
	return fmt.Sprintf(`{"spec":{"service_type":"storage","metadata":{"name":%q},"capacity":"10Gi"}}`, name)
}

// stubVolumeRepository is a minimal store for happy-path middleware tests.
type stubVolumeRepository struct{}

func (s *stubVolumeRepository) Create(_ context.Context, spec v1alpha1.StorageSpec, id string) (*v1alpha1.Volume, error) {
	now := time.Now().UTC()
	status := v1alpha1.PROVISIONING
	path := "volumes/" + id
	ns := "default"
	spec.Metadata.Namespace = &ns
	return &v1alpha1.Volume{
		Id:         &id,
		Path:       &path,
		Status:     &status,
		CreateTime: &now,
		UpdateTime: &now,
		Spec:       spec,
	}, nil
}

func (s *stubVolumeRepository) Get(_ context.Context, _ string) (*v1alpha1.Volume, error) {
	panic("unexpected call to Get")
}

func (s *stubVolumeRepository) List(_ context.Context, _ int32, _ string) (*v1alpha1.VolumeList, error) {
	panic("unexpected call to List")
}

func (s *stubVolumeRepository) Delete(_ context.Context, _ string) error {
	panic("unexpected call to Delete")
}

func (s *stubVolumeRepository) CheckHealth(_ context.Context) error {
	return nil
}

var _ store.VolumeRepository = (*stubVolumeRepository)(nil)

func startValidationServer(repo store.VolumeRepository) string {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Address:         ":0",
			ShutdownTimeout: 5 * time.Second,
		},
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	healthHandler := health.NewHandler(repo, logger, time.Now(), "0.0.1-test")
	volumeHandler := volume.NewHandler(repo, logger)
	apiHandler := composite.NewHandler(healthHandler, volumeHandler)
	strictAdapter := oapigen.NewStrictHandlerWithOptions(apiHandler, nil, oapigen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  apiserver.NewRequestErrorHandler(logger),
		ResponseErrorHandlerFunc: apiserver.NewResponseErrorHandler(logger),
	})
	srv := apiserver.New(cfg, logger, strictAdapter)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	Expect(err).NotTo(HaveOccurred())
	addr := ln.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())
	DeferCleanup(cancel)

	go func() {
		_ = srv.Run(ctx, ln)
	}()

	Eventually(func() error {
		resp, reqErr := http.Get(fmt.Sprintf("http://%s%s/health", addr, volumesAPIPath))
		if reqErr != nil {
			return reqErr
		}
		_ = resp.Body.Close()
		return nil
	}).WithTimeout(5 * time.Second).WithPolling(50 * time.Millisecond).Should(Succeed())

	return fmt.Sprintf("http://%s", addr)
}

func expectProblem400(resp *http.Response, description string) map[string]any {
	Expect(resp.StatusCode).To(Equal(http.StatusBadRequest),
		"expected 400 for: %s", description)
	Expect(resp.Header.Get("Content-Type")).To(Equal("application/problem+json"),
		"expected RFC 9457 content type for: %s", description)

	body, err := io.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())

	var problemJSON map[string]any
	Expect(json.Unmarshal(body, &problemJSON)).To(Succeed(),
		"body should be valid JSON for: %s", description)
	Expect(problemJSON).To(HaveKey("type"),
		"RFC 9457 body must have 'type' for: %s", description)
	Expect(problemJSON["type"]).To(Equal("INVALID_ARGUMENT"))
	Expect(problemJSON).To(HaveKey("title"),
		"RFC 9457 body must have 'title' for: %s", description)
	Expect(problemJSON).To(HaveKey("status"),
		"RFC 9457 body must have 'status' for: %s", description)
	Expect(problemJSON).To(HaveKey("instance"),
		"RFC 9457 body should have 'instance' (REQ-XC-ERR-030) for: %s", description)
	return problemJSON
}

var _ = Describe("Volume API - Request Validation", func() {
	// Middleware-only cases use a stub so health readiness works; Create is never
	// reached when OpenAPI rejects the request.
	DescribeTable("validates request body via OpenAPI middleware (REQ-HTTP-090, TC-U091)",
		func(bodyJSON string, description string) {
			baseURL := startValidationServer(&stubVolumeRepository{})

			resp, err := http.Post(
				baseURL+volumesAPIPath,
				"application/json",
				strings.NewReader(bodyJSON),
			)
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			problem := expectProblem400(resp, description)
			Expect(problem["instance"]).To(Equal(volumesAPIPath))
		},

		Entry("empty object",
			`{}`,
			"empty object missing required spec field"),
		Entry("missing capacity",
			`{"spec":{"service_type":"storage","metadata":{"name":"test-vol"}}}`,
			"missing required capacity field"),
		Entry("missing metadata",
			`{"spec":{"service_type":"storage","capacity":"10Gi"}}`,
			"missing required metadata field"),
		Entry("missing service_type",
			`{"spec":{"metadata":{"name":"test-vol"},"capacity":"10Gi"}}`,
			"missing required service_type field"),
		Entry("missing metadata.name",
			`{"spec":{"service_type":"storage","metadata":{},"capacity":"10Gi"}}`,
			"missing required metadata.name"),
		Entry("invalid service_type enum",
			`{"spec":{"service_type":"vm","metadata":{"name":"test-vol"},"capacity":"10Gi"}}`,
			"invalid service_type enum value"),
		Entry("capacity wrong type",
			`{"spec":{"service_type":"storage","metadata":{"name":"test-vol"},"capacity":10}}`,
			"capacity wrong type"),
		Entry("capacity invalid pattern",
			`{"spec":{"service_type":"storage","metadata":{"name":"test-vol"},"capacity":"ten-gig"}}`,
			"capacity does not match OpenAPI pattern"),
		Entry("malformed JSON",
			`{not valid json}`,
			"malformed JSON body"),
		Entry("empty body",
			``,
			"empty request body"),
		Entry("metadata.name with invalid characters",
			`{"spec":{"service_type":"storage","metadata":{"name":"Invalid_Name!"},"capacity":"10Gi"}}`,
			"metadata.name with invalid characters"),
		Entry("raw StorageSpec body without spec wrapper",
			`{"service_type":"storage","metadata":{"name":"test-vol"},"capacity":"10Gi"}`,
			"raw body missing required spec wrapper"),
	)

	DescribeTable("rejects invalid client IDs via OpenAPI middleware",
		func(invalidID string, description string) {
			baseURL := startValidationServer(&stubVolumeRepository{})

			resp, err := http.Post(
				baseURL+volumesAPIPath+"?id="+invalidID,
				"application/json",
				strings.NewReader(validCreateBody),
			)
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			problem := expectProblem400(resp, description)
			Expect(problem["instance"]).To(HavePrefix(volumesAPIPath + "?id="))
		},
		Entry("leading dash", "-leading-dash", "ID starting with dash"),
		Entry("trailing dash", "trailing-", "ID ending with dash"),
		Entry("has underscore", "has_underscore", "ID containing underscore"),
		Entry("UPPERCASE", "UPPERCASE", "ID with uppercase letters"),
		Entry("too long (64 chars)", strings.Repeat("a", 64), "ID exceeding 63 character limit"),
	)

	DescribeTable("accepts valid boundary IDs via OpenAPI middleware",
		func(validID string, description string) {
			baseURL := startValidationServer(&stubVolumeRepository{})

			resp, err := http.Post(
				baseURL+volumesAPIPath+"?id="+validID,
				"application/json",
				strings.NewReader(createBodyWithName(validID)),
			)
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			Expect(resp.StatusCode).NotTo(Equal(http.StatusBadRequest),
				"valid ID should pass OpenAPI validation: %s", description)
		},
		Entry("single char", "a", "minimum length"),
		Entry("two chars", "ab", "two characters"),
		Entry("max length (63 chars)", strings.Repeat("a", 63), "maximum length"),
		Entry("with hyphens", "a-b", "dash in middle"),
		Entry("letters and digits", "a0", "letter followed by digit"),
		Entry("starts with digit", "1abc", "starts with digit"),
	)

	It("passes a valid request through OpenAPI middleware (REQ-HTTP-091 happy path)", func() {
		baseURL := startValidationServer(&stubVolumeRepository{})

		resp, err := http.Post(
			baseURL+volumesAPIPath,
			"application/json",
			strings.NewReader(validCreateBody),
		)
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = resp.Body.Close() }()

		Expect(resp.StatusCode).To(Equal(http.StatusCreated))
		Expect(resp.Header.Get("Content-Type")).NotTo(ContainSubstring("application/problem+json"))

		respBody, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())

		var result map[string]any
		Expect(json.Unmarshal(respBody, &result)).To(Succeed())
		Expect(result).To(HaveKey("spec"))
		spec, ok := result["spec"].(map[string]any)
		Expect(ok).To(BeTrue(), "spec should be an object")
		Expect(spec["service_type"]).To(Equal("storage"))
		Expect(spec["capacity"]).To(Equal("10Gi"))
		Expect(spec).To(HaveKey("metadata"))
		meta, ok := spec["metadata"].(map[string]any)
		Expect(ok).To(BeTrue(), "metadata should be an object")
		Expect(meta["name"]).To(Equal("test-vol"))
	})
})
