package registration_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dcm-project/k8s-storage-service-provider/internal/config"
	"github.com/dcm-project/k8s-storage-service-provider/internal/registration"
)

var _ = Describe("Registration Payload", func() {
	It("contains storage service type, volumes endpoint, and CRUD operations", func() {
		cfg := &config.Config{
			Provider: config.ProviderConfig{
				Name:        "k8s-storage-sp",
				DisplayName: "K8s Storage SP",
				Endpoint:    "https://sp.example.com",
			},
		}

		payload := registration.BuildPayload(cfg)

		Expect(payload.Name).To(Equal("k8s-storage-sp"))
		Expect(payload.ServiceType).To(Equal("storage"))
		Expect(payload.DisplayName).To(HaveValue(Equal("K8s Storage SP")))
		Expect(payload.Endpoint).To(Equal("https://sp.example.com/api/v1alpha1/volumes"))
		Expect(payload.Operations).To(HaveValue(ConsistOf("CREATE", "READ", "UPDATE", "DELETE")))
		Expect(payload.SchemaVersion).To(Equal("v1alpha1"))
	})
})
