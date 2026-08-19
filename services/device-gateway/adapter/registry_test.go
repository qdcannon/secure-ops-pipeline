package adapter

import "testing"

func TestRegistry_BuildKnownType(t *testing.T) {
	// motion_sensor registers itself via init() in motion_sensor.go, so by
	// the time tests run, it should already be present in the registry.
	a, err := Build(DeviceMotionSensor, "registry-test-001", map[string]string{
		"interval_seconds": "1",
	})
	if err != nil {
		t.Fatalf("Build failed for a registered type: %v", err)
	}
	if a.DeviceID() != "registry-test-001" {
		t.Errorf("DeviceID() = %q, want %q", a.DeviceID(), "registry-test-001")
	}
	if a.DeviceType() != DeviceMotionSensor {
		t.Errorf("DeviceType() = %q, want %q", a.DeviceType(), DeviceMotionSensor)
	}
}

func TestRegistry_BuildUnknownTypeFails(t *testing.T) {
	_, err := Build(DeviceType("not_a_real_device_type"), "x", nil)
	if err == nil {
		t.Error("Build succeeded for an unregistered device type, want an error")
	}
}

func TestRegistry_InvalidIntervalConfigFails(t *testing.T) {
	// Exercises the config-parsing error path in motion_sensor.go's
	// factory function -- a malformed devices.json value should fail
	// loudly at construction time, not silently fall back to a default.
	_, err := Build(DeviceMotionSensor, "registry-test-002", map[string]string{
		"interval_seconds": "not-a-number",
	})
	if err == nil {
		t.Error("Build succeeded with an invalid interval_seconds value, want an error")
	}
}
