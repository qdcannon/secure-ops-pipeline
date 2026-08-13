package adapter

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"time"
)

// init registers this adapter type so main.go can construct it dynamically
// by name (from devices.json) via adapter.Build, without importing this
// concrete type directly.
func init() {
	Register(DeviceMotionSensor, func(deviceID string, config map[string]string) (Adapter, error) {
		interval := 2 * time.Second
		if raw, ok := config["interval_seconds"]; ok {
			secs, err := strconv.Atoi(raw)
			if err != nil {
				return nil, fmt.Errorf("invalid interval_seconds %q for %s: %w", raw, deviceID, err)
			}
			interval = time.Duration(secs) * time.Second
		}
		return NewMotionSensor(deviceID, interval)
	})
}

// MotionSensor is a simulated motion sensor adapter. In place of real
// hardware, it periodically emits a "motion"/"clear" reading, with an
// occasional randomized gap in reporting to exercise the Processor's
// heartbeat/anomaly detection downstream.
type MotionSensor struct {
	deviceID   string
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	interval   time.Duration
}

// NewMotionSensor generates a fresh Ed25519 keypair for this device
// instance, simulating per-device attestation identity.
func NewMotionSensor(deviceID string, interval time.Duration) (*MotionSensor, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("key generation failed: %w", err)
	}
	return &MotionSensor{
		deviceID:   deviceID,
		privateKey: priv,
		publicKey:  pub,
		interval:   interval,
	}, nil
}

func (m *MotionSensor) DeviceType() DeviceType { return DeviceMotionSensor } //implement adapter interface function DeviceType
func (m *MotionSensor) DeviceID() string       { return m.deviceID }         //implement adapter interface fucntion Device Id

func (m *MotionSensor) Connect(ctx context.Context) error { //implement adapter interface function Connect
	// Real hardware: open serial/GPIO/network connection here.
	return nil
}

func (m *MotionSensor) Sign(payload []byte, metadata map[string]string) (Event, error) { //implement adapter interface function Sign
	sum := sha256.Sum256(payload)                       //generate checksum
	checksum := hex.EncodeToString(sum[:])              //encode checksum to string
	sig := ed25519.Sign(m.privateKey, []byte(checksum)) //create signature for payload

	return Event{ //return event data shape
		DeviceID:   m.deviceID,
		DeviceType: DeviceMotionSensor,
		Timestamp:  time.Now().UTC(),
		Payload:    payload,
		Checksum:   checksum,
		Signature:  sig,
		PublicKey:  m.publicKey,
		Metadata:   metadata,
	}, nil
}

// Listen simulates periodic sensor readings. Roughly 1-in-6 cycles, it
// skips a beat entirely (simulating a sensor going silent / being
// tampered with) so the Processor's gap-detection has something real to
// catch in the demo.
func (m *MotionSensor) Listen(ctx context.Context) (<-chan Event, <-chan error) {
	events := make(chan Event)
	errs := make(chan error)

	go func() {
		defer close(events) //Clean up
		defer close(errs)

		ticker := time.NewTicker(m.interval)
		defer ticker.Stop() //ticker cleanup

		for {
			select {
			case <-ctx.Done(): //channel has been cancelled likely by user SIGTERM
				return
			case <-ticker.C: //if ticker channel
				if skipBeat() { //simulate missing heartbeat
					errs <- fmt.Errorf("simulated dropped reading (device silent this cycle)")
					continue
				}

				state := "clear"
				if triggered() { //simulate motion
					state = "motion"
				}
				payload := []byte(fmt.Sprintf(`{"state":"%s"}`, state)) //current state in payload

				ev, err := m.Sign(payload, map[string]string{"zone": "front_entrance"}) //signing for authentication
				if err != nil {
					errs <- err
					continue
				}
				events <- ev //send to events channel
			}
		}
	}()

	return events, errs
}

func (m *MotionSensor) Close() error { return nil }

// --- small helpers for simulation randomness, no math/rand global state ---

func skipBeat() bool  { return randN(6) == 0 }
func triggered() bool { return randN(3) == 0 }

func randN(n int64) int64 {
	v, _ := rand.Int(rand.Reader, big.NewInt(n))
	return v.Int64()
}
