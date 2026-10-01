package adapter

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"
)

var _ Adapter = (*Camera)(nil)

func TestCamera_DeviceIdentity(t *testing.T) {
	c, err := NewCamera("test-camera-001", time.Second, "1920x1080")
	if err != nil {
		t.Fatalf("NewCamera failed: %v", err)
	}

	if got := c.DeviceType(); got != DeviceCamera {
		t.Errorf("DeviceType() = %q, want %q", got, DeviceCamera)
	}
	if got := c.DeviceID(); got != "test-camera-001" {
		t.Errorf("DeviceID() = %q, want %q", got, "test-camera-001")
	}
}

func TestCamera_Sign_ProducesValidSignature(t *testing.T) {
	c, err := NewCamera("test-camera-002", time.Second, "1280x720")
	if err != nil {
		t.Fatalf("NewCamera failed: %v", err)
	}

	ev, err := c.Sign([]byte(`{"frame_id":1}`), map[string]string{"resolution": "1280x720"})
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	pub := ed25519.PublicKey(ev.PublicKey)
	if !ed25519.Verify(pub, []byte(ev.Checksum), ev.Signature) {
		t.Error("signature failed to verify against event checksum + public key")
	}
}

func TestCamera_RepeatedFrameProducesIdenticalChecksum(t *testing.T) {
	// This is the property the future camera_dedup processor relies on:
	// looped/replayed footage must checksum identically to the original,
	// so a naive duplicate-checksum check is enough to flag it.
	c, err := NewCamera("test-camera-003", time.Second, "1920x1080")
	if err != nil {
		t.Fatalf("NewCamera failed: %v", err)
	}

	first := c.nextFrame()
	ev1, err := c.Sign(first, nil)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	// Force a repeat by directly reusing lastFrame, bypassing the random
	// chance in nextFrame() so the test is deterministic.
	repeated := c.lastFrame
	ev2, err := c.Sign(repeated, nil)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if ev1.Checksum != ev2.Checksum {
		t.Errorf("checksums differ for identical frame bytes: %q vs %q", ev1.Checksum, ev2.Checksum)
	}
}

func TestCamera_NextFrame_EventuallyRepeats(t *testing.T) {
	// Statistical check: over enough cycles, nextFrame() should produce at
	// least one repeated payload (the simulated "looped footage" case),
	// given the 1-in-5 chance built into it.
	c, err := NewCamera("test-camera-004", time.Second, "1920x1080")
	if err != nil {
		t.Fatalf("NewCamera failed: %v", err)
	}

	seen := map[string]bool{}
	repeatFound := false
	for i := 0; i < 200; i++ {
		frame := c.nextFrame()
		key := string(frame)
		if seen[key] {
			repeatFound = true
			break
		}
		seen[key] = true
	}

	if !repeatFound {
		t.Error("nextFrame() never produced a repeated payload across 200 cycles -- loop simulation may be broken")
	}
}

func TestCamera_Listen_EmitsEvents(t *testing.T) {
	c, err := NewCamera("test-camera-005", 50*time.Millisecond, "1920x1080")
	if err != nil {
		t.Fatalf("NewCamera failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if err := c.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer c.Close()

	evCh, _ := c.Listen(ctx)

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
		case <-ctx.Done():
			if received == 0 {
				t.Error("Listen produced no events within the test window")
			}
			return
		}
	}
}
