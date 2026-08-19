package adapter

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

// Compile-time check: this line fails to build if *MotionSensor ever
// stops satisfying the Adapter interface (e.g. a method signature drifts
// out of sync). Catches the mistake at compile time, before any test runs.
var _ Adapter = (*MotionSensor)(nil)

func TestMotionSensor_DeviceIdentity(t *testing.T) {
	m, err := NewMotionSensor("test-sensor-001", time.Second)
	if err != nil {
		t.Fatalf("NewMotionSensor failed: %v", err)
	}

	if got := m.DeviceType(); got != DeviceMotionSensor {
		t.Errorf("DeviceType() = %q, want %q", got, DeviceMotionSensor)
	}
	if got := m.DeviceID(); got != "test-sensor-001" {
		t.Errorf("DeviceID() = %q, want %q", got, "test-sensor-001")
	}
}

func TestMotionSensor_Sign_ProducesValidChecksumAndSignature(t *testing.T) {
	m, err := NewMotionSensor("test-sensor-002", time.Second)
	if err != nil {
		t.Fatalf("NewMotionSensor failed: %v", err)
	}

	payload := []byte(`{"state":"motion"}`)
	ev, err := m.Sign(payload, map[string]string{"zone": "test_zone"})
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// The checksum must actually be the SHA-256 of the payload we passed in
	// -- this is what the processor relies on to detect tampering.
	wantSum := sha256.Sum256(payload)
	wantChecksum := hex.EncodeToString(wantSum[:])
	if ev.Checksum != wantChecksum {
		t.Errorf("Checksum = %q, want %q", ev.Checksum, wantChecksum)
	}

	// The signature must verify against the event's own public key and
	// checksum -- this is what the processor relies on to reject spoofed
	// or altered events.
	pub := ed25519.PublicKey(ev.PublicKey)
	if !ed25519.Verify(pub, []byte(ev.Checksum), ev.Signature) {
		t.Error("signature failed to verify against event checksum + public key")
	}

	if ev.DeviceID != "test-sensor-002" {
		t.Errorf("Event.DeviceID = %q, want %q", ev.DeviceID, "test-sensor-002")
	}
	if ev.DeviceType != DeviceMotionSensor {
		t.Errorf("Event.DeviceType = %q, want %q", ev.DeviceType, DeviceMotionSensor)
	}
	if ev.Metadata["zone"] != "test_zone" {
		t.Errorf("Event.Metadata[zone] = %q, want %q", ev.Metadata["zone"], "test_zone")
	}
}

func TestMotionSensor_Sign_TamperedPayloadFailsVerification(t *testing.T) {
	// This is the negative case that actually matters most: prove a
	// tampered payload does NOT verify, i.e. the security property holds.
	m, err := NewMotionSensor("test-sensor-003", time.Second)
	if err != nil {
		t.Fatalf("NewMotionSensor failed: %v", err)
	}

	ev, err := m.Sign([]byte(`{"state":"motion"}`), nil)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// Simulate an attacker altering the checksum after signing.
	ev.Checksum = "tampered0000000000000000000000000000000000000000000000000000"

	pub := ed25519.PublicKey(ev.PublicKey)
	if ed25519.Verify(pub, []byte(ev.Checksum), ev.Signature) {
		t.Error("signature verified against a tampered checksum -- should have failed")
	}
}

func TestMotionSensor_Listen_EmitsEvents(t *testing.T) {
	m, err := NewMotionSensor("test-sensor-004", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewMotionSensor failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if err := m.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer m.Close()

	evCh, errCh := m.Listen(ctx)

	received := 0
	for {
		select {
		case _, ok := <-evCh:
			if !ok {
				if received == 0 {
					t.Error("Listen closed evCh without emitting any events")
				}
				return
			}
			received++
		case _, ok := <-errCh:
			if !ok {
				continue
			}
			// Expected occasionally (simulated dropped reading) -- not a failure.
		case <-ctx.Done():
			if received == 0 {
				t.Error("Listen produced no events within the test window")
			}
			return
		}
	}
}

func TestMotionSensor_KeysAreUniquePerDevice(t *testing.T) {
	// Each device instance should get its own signing identity -- this is
	// the whole basis for device attestation. Two sensors must not share
	// a keypair.
	a, err := NewMotionSensor("device-a", time.Second)
	if err != nil {
		t.Fatalf("NewMotionSensor failed: %v", err)
	}
	b, err := NewMotionSensor("device-b", time.Second)
	if err != nil {
		t.Fatalf("NewMotionSensor failed: %v", err)
	}

	if string(a.publicKey) == string(b.publicKey) {
		t.Error("two independently constructed devices share a public key")
	}
}
