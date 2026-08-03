package config_test

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dcm-project/k8s-storage-service-provider/internal/config"
)

var _ = Describe("Configuration", func() {
	clearEnv := func() {
		_ = os.Unsetenv("SP_SERVER_ADDRESS")
		_ = os.Unsetenv("SP_SERVER_SHUTDOWN_TIMEOUT")
		_ = os.Unsetenv("SP_SERVER_READ_TIMEOUT")
		_ = os.Unsetenv("SP_SERVER_WRITE_TIMEOUT")
		_ = os.Unsetenv("SP_SERVER_IDLE_TIMEOUT")
		_ = os.Unsetenv("SP_NAME")
		_ = os.Unsetenv("SP_DISPLAY_NAME")
		_ = os.Unsetenv("SP_ENDPOINT")
		_ = os.Unsetenv("SP_REGION")
		_ = os.Unsetenv("SP_ZONE")
		_ = os.Unsetenv("DCM_REGISTRATION_URL")
		_ = os.Unsetenv("SP_NATS_URL")
		_ = os.Unsetenv("SP_SERVER_REQUEST_TIMEOUT")
		_ = os.Unsetenv("SP_K8S_DEFAULT_STORAGE_CLASS")
		_ = os.Unsetenv("SP_K8S_DEFAULT_ACCESS_MODE")
		_ = os.Unsetenv("SP_MONITOR_PUBLISH_MAX_ATTEMPTS")
		_ = os.Unsetenv("SP_MONITOR_DEBOUNCE_MS")
		_ = os.Unsetenv("SP_MONITOR_RESYNC_PERIOD")
	}

	BeforeEach(func() {
		clearEnv()
	})

	AfterEach(func() {
		clearEnv()
	})

	setRequiredEnv := func() {
		_ = os.Setenv("SP_NAME", "test-sp")
		_ = os.Setenv("SP_ENDPOINT", "https://test.example.com")
		_ = os.Setenv("DCM_REGISTRATION_URL", "https://dcm.example.com")
		_ = os.Setenv("SP_NATS_URL", "nats://test:4222")
	}

	It("loads configuration from environment variables", func() {
		setRequiredEnv()
		_ = os.Setenv("SP_SERVER_ADDRESS", ":9090")
		_ = os.Setenv("SP_K8S_DEFAULT_STORAGE_CLASS", "gp3-csi")
		_ = os.Setenv("SP_K8S_DEFAULT_ACCESS_MODE", "ReadWriteMany")

		cfg, err := config.Load()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Server.Address).To(Equal(":9090"))
		Expect(cfg.Kubernetes.DefaultStorageClass).To(Equal("gp3-csi"))
		Expect(cfg.Kubernetes.DefaultAccessMode).To(Equal("ReadWriteMany"))
	})

	It("applies default values when no config is specified", func() {
		setRequiredEnv()

		cfg, err := config.Load()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Server.Address).To(Equal(":8080"))
		Expect(cfg.Kubernetes.DefaultAccessMode).To(Equal("ReadWriteOnce"))
		Expect(cfg.Server.ShutdownTimeout).To(Equal(15 * time.Second))
		Expect(cfg.Monitoring.PublishMaxAttempts).To(Equal(5))
		Expect(cfg.Monitoring.DebounceMs).To(Equal(500))
	})

	It("loads SP_MONITOR_PUBLISH_MAX_ATTEMPTS override", func() {
		setRequiredEnv()
		_ = os.Setenv("SP_MONITOR_PUBLISH_MAX_ATTEMPTS", "3")

		cfg, err := config.Load()
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.Monitoring.PublishMaxAttempts).To(Equal(3))
	})

	It("returns error when required fields are missing", func() {
		cfg, err := config.Load()
		Expect(err).To(HaveOccurred())
		Expect(cfg).To(BeNil())
		errMsg := err.Error()
		Expect(errMsg).To(ContainSubstring("SP_NAME"))
		Expect(errMsg).To(ContainSubstring("SP_ENDPOINT"))
		Expect(errMsg).To(ContainSubstring("DCM_REGISTRATION_URL"))
		Expect(errMsg).To(ContainSubstring("SP_NATS_URL"))
	})

	It("returns error when SP_NATS_URL is missing", func() {
		setRequiredEnv()
		_ = os.Unsetenv("SP_NATS_URL")

		cfg, err := config.Load()
		Expect(err).To(HaveOccurred())
		Expect(cfg).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("SP_NATS_URL"))
	})

	It("returns error when default access mode is invalid", func() {
		setRequiredEnv()
		_ = os.Setenv("SP_K8S_DEFAULT_ACCESS_MODE", "InvalidMode")

		cfg, err := config.Load()
		Expect(err).To(HaveOccurred())
		Expect(cfg).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("SP_K8S_DEFAULT_ACCESS_MODE"))
	})
})
