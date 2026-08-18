package monitoring

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	v1alpha1 "github.com/dcm-project/k8s-storage-service-provider/api/v1alpha1"
)

func TestSubmitIfChanged_suppressesProvisioningAfterFailed(t *testing.T) {
	t.Parallel()

	instanceID := "fail-latch"
	m := &StatusMonitor{
		logger:        slog.Default(),
		lastPublished: make(map[string]StatusEvent),
		lastSubmitted: make(map[string]StatusEvent),
	}
	m.lastSubmitted[instanceID] = StatusEvent{
		InstanceID: instanceID,
		Status:     v1alpha1.FAILED,
		Message:    "no matching volume",
	}

	var submitted []StatusEvent
	debouncer := NewDebouncer(time.Hour, func(event StatusEvent) {
		submitted = append(submitted, event)
	})

	m.submitIfChanged(debouncer, StatusEvent{
		InstanceID: instanceID,
		Status:     v1alpha1.PROVISIONING,
		Message:    "PVC is pending",
	})

	m.mu.Lock()
	last := m.lastSubmitted[instanceID]
	m.mu.Unlock()

	if last.Status != v1alpha1.FAILED {
		t.Fatalf("lastSubmitted status = %s, want FAILED", last.Status)
	}
	if len(submitted) != 0 {
		t.Fatalf("debouncer published %d events, want 0", len(submitted))
	}
}

func TestSubmitIfChanged_concurrentFailedNotOverwrittenByProvisioning(t *testing.T) {
	t.Parallel()

	const iterations = 500
	instanceID := "race-fail-prov"

	for i := 0; i < iterations; i++ {
		m := &StatusMonitor{
			logger:        slog.Default(),
			lastPublished: make(map[string]StatusEvent),
			lastSubmitted: make(map[string]StatusEvent),
		}
		debouncer := NewDebouncer(time.Hour, func(StatusEvent) {})

		prov := StatusEvent{
			InstanceID: instanceID,
			Status:     v1alpha1.PROVISIONING,
			Message:    "PVC is pending",
		}
		fail := StatusEvent{
			InstanceID: instanceID,
			Status:     v1alpha1.FAILED,
			Message:    "no matching volume",
		}

		start := make(chan struct{})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			m.submitIfChanged(debouncer, prov)
		}()
		go func() {
			defer wg.Done()
			<-start
			m.submitIfChanged(debouncer, fail)
		}()
		close(start)
		wg.Wait()

		m.mu.Lock()
		last := m.lastSubmitted[instanceID]
		m.mu.Unlock()

		if last.Status == v1alpha1.PROVISIONING {
			t.Fatalf("iteration %d: lastSubmitted downgraded to PROVISIONING after concurrent FAILED", i)
		}
	}
}

func TestPublishWithRetry_clearsMapsAfterDeleted(t *testing.T) {
	t.Parallel()

	instanceID := "gone-123"
	m := &StatusMonitor{
		logger:        slog.Default(),
		cfg:           MonitorConfig{PublishMaxAttempts: 1},
		publisher:     recordingPublisher{},
		lastPublished: map[string]StatusEvent{instanceID: {InstanceID: instanceID, Status: v1alpha1.DELETING}},
		lastSubmitted: map[string]StatusEvent{instanceID: {InstanceID: instanceID, Status: v1alpha1.DELETED}},
	}

	m.publishWithRetry(context.Background(), StatusEvent{
		InstanceID: instanceID,
		Status:     v1alpha1.DELETED,
		Message:    "resource no longer exists",
	})

	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.lastPublished[instanceID]; ok {
		t.Fatal("lastPublished still has entry after DELETED publish")
	}
	if _, ok := m.lastSubmitted[instanceID]; ok {
		t.Fatal("lastSubmitted still has entry after DELETED publish")
	}
}

type recordingPublisher struct{}

func (recordingPublisher) Publish(context.Context, StatusEvent) error { return nil }

func (recordingPublisher) Close() error { return nil }
