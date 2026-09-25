package adapter

import (
	"context"
	"fmt"
)

// init registers stub factories for every device type that has an
// interface contract but no real implementation yet. This is what proves
// the "add a new device type without touching the gateway" claim in
// docs/architecture.md -- these four types exist in devices.json and the
// registry today, with zero changes to gateway.go, main.go, or the
// Adapter interface itself.
func init() {
	for _, t := range []DeviceType{DeviceRFIDAccess, DeviceAlarm, DeviceRadio, DevicePresence} {
		t := t // capture loop variable per-iteration
		Register(t, func(deviceID string, config map[string]string) (Adapter, error) {
			return &stubAdapter{deviceID: deviceID, deviceType: t}, nil
		})
	}
}

// stubAdapter satisfies the Adapter interface for a device type with no
// real implementation yet. It constructs successfully (so it can appear
// in devices.json without error) but fails at Connect with a clear,
// consistent message -- the Gateway already handles a Connect failure
// gracefully (logs, closes, moves on to other devices; see
// gateway_test.go's TestRunAdapter_ConnectFailureSkipsListenButStillCloses),
// so no special-casing is needed anywhere else.
type stubAdapter struct {
	deviceID   string
	deviceType DeviceType
}

func (s *stubAdapter) DeviceType() DeviceType { return s.deviceType }
func (s *stubAdapter) DeviceID() string       { return s.deviceID }

func (s *stubAdapter) Connect(ctx context.Context) error {
	return fmt.Errorf("adapter for device type %q is not yet implemented", s.deviceType)
}

func (s *stubAdapter) Listen(ctx context.Context) (<-chan Event, <-chan error) {
	events := make(chan Event)
	errs := make(chan error)
	close(events)
	close(errs)
	return events, errs
}

func (s *stubAdapter) Sign(payload []byte, metadata map[string]string) (Event, error) {
	return Event{}, fmt.Errorf("adapter for device type %q is not yet implemented", s.deviceType)
}

func (s *stubAdapter) Close() error { return nil }
