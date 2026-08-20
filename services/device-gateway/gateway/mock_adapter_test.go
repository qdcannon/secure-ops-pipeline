package gateway

import (
	"context"
	"sync"

	"secure-ops-pipeline/services/device-gateway/adapter"
)

// mockAdapter is a controllable stand-in for a real device adapter, used
// so gateway tests exercise Run/runAdapter's orchestration logic without
// depending on any real device (or even the adapter package's concrete
// types). It satisfies adapter.Adapter the same way MotionSensor does.
type mockAdapter struct {
	id         string
	deviceType adapter.DeviceType
	connectErr error
	events     []adapter.Event // emitted in order, then evCh closes
	emitErr    error           // if set, sent once on errCh before events

	mu            sync.Mutex
	connectCalled bool
	closeCalled   bool
}

func (m *mockAdapter) DeviceType() adapter.DeviceType { return m.deviceType }
func (m *mockAdapter) DeviceID() string               { return m.id }

func (m *mockAdapter) Connect(ctx context.Context) error {
	m.mu.Lock()
	m.connectCalled = true
	m.mu.Unlock()
	return m.connectErr
}

func (m *mockAdapter) Listen(ctx context.Context) (<-chan adapter.Event, <-chan error) {
	evCh := make(chan adapter.Event)
	errCh := make(chan error)

	go func() {
		defer close(evCh)
		defer close(errCh)

		if m.emitErr != nil {
			select {
			case errCh <- m.emitErr:
			case <-ctx.Done():
				return
			}
		}

		for _, e := range m.events {
			select {
			case evCh <- e:
			case <-ctx.Done():
				return
			}
		}
	}()

	return evCh, errCh
}

func (m *mockAdapter) Sign(payload []byte, metadata map[string]string) (adapter.Event, error) {
	return adapter.Event{}, nil
}

func (m *mockAdapter) Close() error {
	m.mu.Lock()
	m.closeCalled = true
	m.mu.Unlock()
	return nil
}

func (m *mockAdapter) wasConnectCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connectCalled
}

func (m *mockAdapter) wasCloseCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closeCalled
}
