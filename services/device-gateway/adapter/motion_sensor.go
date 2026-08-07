package adapter

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

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

func (m *MotionSensor) DeviceType() DeviceType { return DeviceMotionSensor }
func (m *MotionSensor) DeviceID() string        { return m.deviceID }

func (m *MotionSensor) Connect(ctx context.Context) error {
	// Real hardware: open serial/GPIO/network connection here.
	return nil
}

func (m *MotionSensor) Sign(payload []byte, metadata map[string]string) (Event, error) {
	sum := sha256.Sum256(payload)
	checksum := hex.EncodeToString(sum[:])
	sig := ed25519.Sign(m.privateKey, []byte(checksum))

	return Event{
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
		defer close(events)
		defer close(errs)

		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if skipBeat() {
					errs <- fmt.Errorf("simulated dropped reading (device silent this cycle)")
					continue
				}

				state := "clear"
				if triggered() {
					state = "motion"
				}
				payload := []byte(fmt.Sprintf(`{"state":"%s"}`, state))

				ev, err := m.Sign(payload, map[string]string{"zone": "front_entrance"})
				if err != nil {
					errs <- err
					continue
				}
				events <- ev
			}
		}
	}()

	return events, errs
}

func (m *MotionSensor) Close() error { return nil }

// --- small helpers for simulation randomness, no math/rand global state ---

func skipBeat() bool    { return randN(6) == 0 }
func triggered() bool   { return randN(3) == 0 }

func randN(n int64) int64 {
	v, _ := rand.Int(rand.Reader, big.NewInt(n))
	return v.Int64()
}
