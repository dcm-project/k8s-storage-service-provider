package monitoring

import (
	"sync"
	"time"
)

// instanceState holds per-instance debounce state.
type instanceState struct {
	timer   *time.Timer
	pending StatusEvent
	ver     uint64
}

// Debouncer coalesces rapid status change events within a time window,
// publishing only the last event once the window elapses.
type Debouncer struct {
	interval  time.Duration
	publishFn func(StatusEvent)
	mu        sync.Mutex
	wg        sync.WaitGroup
	instances map[string]*instanceState
	stopped   bool
}

// NewDebouncer creates a Debouncer with the given interval and publish callback.
func NewDebouncer(interval time.Duration, publishFn func(StatusEvent)) *Debouncer {
	return &Debouncer{
		interval:  interval,
		publishFn: publishFn,
		instances: make(map[string]*instanceState),
	}
}

// Submit queues a status event for the given instance. If another event
// arrives within the debounce window, the previous event is replaced and the
// window is reset. Only the latest pending event is published when the timer fires.
func (d *Debouncer) Submit(instanceID string, event StatusEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stopped {
		return
	}

	state, exists := d.instances[instanceID]
	if !exists {
		state = &instanceState{}
		d.instances[instanceID] = state
	}

	// Increment the version. This invalidates ANY flying callback
	state.pending = event
	state.ver++

	// Capture the version for this specific callback closure
	ver := state.ver

	// Stop old timer if exists.
	if state.timer != nil {
		state.timer.Stop()
	}

	// Schedule the callback to run after the debounce interval.
	state.timer = time.AfterFunc(d.interval, func() {
		var latest StatusEvent
		publish := false

		func() {
			d.mu.Lock()
			defer d.mu.Unlock()
			if d.stopped {
				return
			}
			// Ignore stale callbacks from a timer replaced by a newer Submit.
			cur, ok := d.instances[instanceID]
			if !ok || cur.ver != ver {
				return
			}
			latest = cur.pending
			delete(d.instances, instanceID)
			d.wg.Add(1)
			publish = true
		}()

		if !publish {
			return
		}
		defer d.wg.Done()
		d.publishFn(latest)
	})
}

// Stop halts the debouncer: new submissions are ignored, pending timers are
// cancelled, and each instance's latest pending event is flushed via publishFn.
// Stop waits for those flushes and any in-flight publishFn callbacks before returning.
func (d *Debouncer) Stop() {
	d.mu.Lock()
	d.stopped = true
	toFlush := make([]StatusEvent, 0, len(d.instances))
	for id, state := range d.instances {
		if state.timer != nil {
			state.timer.Stop()
		}
		toFlush = append(toFlush, state.pending)
		delete(d.instances, id)
	}
	d.wg.Add(len(toFlush))
	d.mu.Unlock()

	for _, event := range toFlush {
		d.publishFn(event)
		d.wg.Done()
	}
	d.wg.Wait()
}
