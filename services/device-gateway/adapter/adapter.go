// Package adapter defines the contract every device type (camera, motion
// sensor, and future types like rfid_access, alarm, radio) must implement
// to plug into the Device Gateway.
package adapter

import (
	"context"
	"time"
)

// DeviceType identifies the category of device an adapter handles.
type DeviceType string

const (
	DeviceCamera       DeviceType = "camera"
	DeviceMotionSensor DeviceType = "motion_sensor"
	DeviceRFIDAccess   DeviceType = "rfid_access" // future
	DeviceAlarm        DeviceType = "alarm"       // future
	DeviceRadio        DeviceType = "radio"       // future
)

// Event is the normalized, signed payload every adapter produces,
// regardless of the underlying device type. This is what the Gateway
// publishes over ZeroMQ to the Processor.
type Event struct {
	DeviceID   string            `json:"device_id"`
	DeviceType DeviceType        `json:"device_type"`
	Timestamp  time.Time         `json:"timestamp"`
	Payload    []byte            `json:"payload"`
	Checksum   string            `json:"checksum"`  // SHA-256 hex of Payload
	Signature  []byte            `json:"signature"` // Ed25519 signature over Checksum
	PublicKey  []byte            `json:"public_key"` // sender's public key, for verification
	Metadata   map[string]string `json:"metadata"`
}

// Adapter is the contract every device type must implement. Each adapter
// instance represents one connected device and owns its own connection
// lifecycle; the Gateway runs one goroutine per adapter and fans their
// output into a shared event channel.
type Adapter interface {
	DeviceType() DeviceType
	DeviceID() string
	Connect(ctx context.Context) error
	Listen(ctx context.Context) (<-chan Event, <-chan error)
	Sign(payload []byte, metadata map[string]string) (Event, error)
	Close() error
}
