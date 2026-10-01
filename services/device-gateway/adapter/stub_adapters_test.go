package adapter

import (
	"context"
	"testing"
)

var _ Adapter = (*stubAdapter)(nil)

func TestStubAdapters_AllFutureTypesRegistered(t *testing.T) {
	// If a new future device type gets added to adapter.go's DeviceType
	// consts but someone forgets to register a stub for it, this test
	// catches the gap immediately rather than it surfacing as a confusing
	// "no adapter registered" error the first time devices.json uses it.
	futureTypes := []DeviceType{DeviceRFIDAccess, DeviceAlarm, DeviceRadio, DevicePresence}

	for _, dt := range futureTypes {
		a, err := Build(dt, "stub-test", nil)
		if err != nil {
			t.Errorf("Build(%q) failed, expected a registered stub: %v", dt, err)
			continue
		}
		if a.DeviceType() != dt {
			t.Errorf("stub DeviceType() = %q, want %q", a.DeviceType(), dt)
		}
	}
}

func TestStubAdapter_ConnectFailsClearly(t *testing.T) {
	a, err := Build(DeviceAlarm, "stub-alarm-001", nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	err = a.Connect(context.Background())
	if err == nil {
		t.Fatal("Connect succeeded on a stub adapter, expected a not-implemented error")
	}
}

func TestStubAdapter_ListenReturnsClosedChannels(t *testing.T) {
	a, err := Build(DeviceRadio, "stub-radio-001", nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	evCh, errCh := a.Listen(context.Background())

	if _, ok := <-evCh; ok {
		t.Error("expected evCh to be closed immediately for a stub adapter")
	}
	if _, ok := <-errCh; ok {
		t.Error("expected errCh to be closed immediately for a stub adapter")
	}
}
