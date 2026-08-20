package gateway

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"secure-ops-pipeline/services/device-gateway/adapter"
)

// newSyncPublisher returns a publish function safe to call from Run's
// goroutine, plus an accessor to read published events from the test
// goroutine without a data race.
func newSyncPublisher(fail bool) (publish func(adapter.Event) error, get func() []adapter.Event) {
	var mu sync.Mutex
	var published []adapter.Event

	publish = func(ev adapter.Event) error {
		mu.Lock()
		defer mu.Unlock()
		if fail {
			return errors.New("simulated publish failure")
		}
		published = append(published, ev)
		return nil
	}
	get = func() []adapter.Event {
		mu.Lock()
		defer mu.Unlock()
		out := make([]adapter.Event, len(published))
		copy(out, published)
		return out
	}
	return publish, get
}

// waitFor polls cond until it's true or the timeout elapses, failing the
// test on timeout. Needed because Run/runAdapter operate across
// goroutines with no other synchronization point to hook into.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within timeout")
}

func TestGateway_Run_PublishesEventsFromSingleAdapter(t *testing.T) {
	publish, published := newSyncPublisher(false)
	g := New(publish)

	m := &mockAdapter{
		id:         "mock-001",
		deviceType: adapter.DeviceMotionSensor,
		events: []adapter.Event{
			{DeviceID: "mock-001"},
			{DeviceID: "mock-001"},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go g.Run(ctx, []adapter.Adapter{m})

	waitFor(t, time.Second, func() bool { return len(published()) == 2 })
}

func TestGateway_Run_PublishesFromMultipleAdaptersIndependently(t *testing.T) {
	// Proves the goroutine-per-adapter design: a slow/blocked adapter must
	// not prevent another adapter's events from being published.
	publish, published := newSyncPublisher(false)
	g := New(publish)

	fast := &mockAdapter{
		id:         "fast",
		deviceType: adapter.DeviceMotionSensor,
		events:     []adapter.Event{{DeviceID: "fast"}, {DeviceID: "fast"}},
	}
	// "blocked" never emits or closes -- it just sits there via Listen's
	// ctx.Done() path, simulating a device that never sends anything.
	blocked := &mockAdapter{
		id:         "blocked",
		deviceType: adapter.DeviceMotionSensor,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go g.Run(ctx, []adapter.Adapter{fast, blocked})

	waitFor(t, time.Second, func() bool { return len(published()) == 2 })

	for _, ev := range published() {
		if ev.DeviceID != "fast" {
			t.Errorf("unexpected event from device %q, want only from 'fast'", ev.DeviceID)
		}
	}
}

func TestGateway_Run_PublishFailureDoesNotStopTheLoop(t *testing.T) {
	// If publish() errors (e.g. a transient ZeroMQ send failure), Run must
	// log and continue, not crash or stop processing further events.
	publish, _ := newSyncPublisher(true) // always fails
	g := New(publish)

	m := &mockAdapter{
		id:         "mock-002",
		deviceType: adapter.DeviceMotionSensor,
		events:     []adapter.Event{{DeviceID: "mock-002"}, {DeviceID: "mock-002"}},
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		g.Run(ctx, []adapter.Adapter{m})
		close(done)
	}()

	// Give it time to process both events (and fail to publish both)
	// without panicking, then confirm Run is still responsive to cancellation.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Run returned cleanly after cancellation -- it didn't get stuck
		// or panic despite every publish() call failing.
	case <-time.After(time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

func TestGateway_Run_StopsOnContextCancel(t *testing.T) {
	publish, _ := newSyncPublisher(false)
	g := New(publish)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		g.Run(ctx, nil) // no adapters at all -- Run should still respect cancellation
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return promptly after context cancellation")
	}
}

func TestRunAdapter_ConnectFailureSkipsListenButStillCloses(t *testing.T) {
	g := New(func(adapter.Event) error { return nil })

	m := &mockAdapter{
		id:         "mock-003",
		deviceType: adapter.DeviceMotionSensor,
		connectErr: errors.New("simulated connect failure"),
	}

	out := make(chan adapter.Event, 10)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		g.runAdapter(ctx, m, out)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runAdapter did not return after a Connect failure")
	}

	if !m.wasConnectCalled() {
		t.Error("Connect was never called")
	}
	if !m.wasCloseCalled() {
		t.Error("Close was not called despite defer -- cleanup must run even on Connect failure")
	}
	select {
	case ev := <-out:
		t.Errorf("unexpected event forwarded after Connect failure: %+v", ev)
	default:
		// expected: nothing forwarded
	}
}

func TestRunAdapter_NonFatalErrorsDoNotStopForwarding(t *testing.T) {
	// A dropped reading (emitErr) should be logged and skipped, not treat
	// the whole adapter as dead -- events after it must still forward.
	g := New(func(adapter.Event) error { return nil })

	m := &mockAdapter{
		id:         "mock-004",
		deviceType: adapter.DeviceMotionSensor,
		emitErr:    errors.New("simulated dropped reading"),
		events:     []adapter.Event{{DeviceID: "mock-004"}},
	}

	out := make(chan adapter.Event, 10)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go g.runAdapter(ctx, m, out)

	select {
	case ev := <-out:
		if ev.DeviceID != "mock-004" {
			t.Errorf("DeviceID = %q, want %q", ev.DeviceID, "mock-004")
		}
	case <-time.After(time.Second):
		t.Fatal("event was never forwarded after a non-fatal adapter error")
	}
}

func TestRunAdapter_ContextCancelStopsPromptly(t *testing.T) {
	g := New(func(adapter.Event) error { return nil })

	// No events, no error, no close -- this adapter would block forever
	// on its own; only ctx cancellation should end runAdapter.
	m := &mockAdapter{id: "mock-005", deviceType: adapter.DeviceMotionSensor}

	out := make(chan adapter.Event, 10)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		g.runAdapter(ctx, m, out)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond) // let it settle into the select loop
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runAdapter did not stop promptly after context cancellation")
	}
}

// Sanity check that mockAdapter itself satisfies the interface it's
// standing in for -- if this ever stops compiling, the mock has drifted
// from the real contract and every test above is testing the wrong thing.
var _ adapter.Adapter = (*mockAdapter)(nil)
