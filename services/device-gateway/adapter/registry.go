// adapter/registry.go
package adapter

import "fmt"

// Factory builds an Adapter instance from a device ID and an arbitrary,
// adapter-specific config map (as parsed from devices.json). Each device
// type's file registers its own Factory in an init() function, so main.go
// never needs to know concrete adapter types exist.
type Factory func(deviceID string, config map[string]string) (Adapter, error)

var registry = map[DeviceType]Factory{}

// Register makes a device type's factory available for dynamic construction.
// Called from each adapter's init(), before main() runs.
func Register(t DeviceType, f Factory) {
	registry[t] = f
}

// Build looks up the registered factory for a device type and constructs it.
func Build(t DeviceType, deviceID string, config map[string]string) (Adapter, error) {
	f, ok := registry[t]
	if !ok {
		return nil, fmt.Errorf("no adapter registered for device type %q", t)
	}
	return f(deviceID, config)
}
