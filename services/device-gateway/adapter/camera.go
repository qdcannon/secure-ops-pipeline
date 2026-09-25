package adapter

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// init registers this adapter type so main.go can construct it dynamically
// by name (from devices.json), the same as motion_sensor.go does.
func init() {
	Register(DeviceCamera, func(deviceID string, config map[string]string) (Adapter, error) {
		interval := 5 * time.Second
		if raw, ok := config["interval_seconds"]; ok {
			secs, err := strconv.Atoi(raw)
			if err != nil {
				return nil, fmt.Errorf("invalid interval_seconds %q for %s: %w", raw, deviceID, err)
			}
			interval = time.Duration(secs) * time.Second
		}
		resolution := "1920x1080"
		if raw, ok := config["resolution"]; ok {
			resolution = raw
		}
		return NewCamera(deviceID, interval, resolution)
	})
}

// Camera is a simulated camera adapter. In place of a real video feed, it
// periodically emits a synthetic "frame" payload. Roughly 1-in-5 cycles it
// re-emits the *exact same* payload as the previous frame -- simulating
// looped/replayed footage, the specific tamper pattern the future
// camera_dedup processor is meant to catch (see docs/architecture.md,
// threat model).
type Camera struct {
	deviceID   string
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	interval   time.Duration
	resolution string

	frameNum  int
	lastFrame []byte
}

func NewCamera(deviceID string, interval time.Duration, resolution string) (*Camera, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("key generation failed: %w", err)
	}
	return &Camera{
		deviceID:   deviceID,
		privateKey: priv,
		publicKey:  pub,
		interval:   interval,
		resolution: resolution,
	}, nil
}

func (c *Camera) DeviceType() DeviceType { return DeviceCamera }
func (c *Camera) DeviceID() string       { return c.deviceID }

func (c *Camera) Connect(ctx context.Context) error {
	// Real hardware: open the RTSP/USB/network stream here.
	return nil
}

func (c *Camera) Sign(payload []byte, metadata map[string]string) (Event, error) {
	sum := sha256.Sum256(payload)
	checksum := hex.EncodeToString(sum[:])
	sig := ed25519.Sign(c.privateKey, []byte(checksum))

	return Event{
		DeviceID:   c.deviceID,
		DeviceType: DeviceCamera,
		Timestamp:  time.Now().UTC(),
		Payload:    payload,
		Checksum:   checksum,
		Signature:  sig,
		PublicKey:  c.publicKey,
		Metadata:   metadata,
	}, nil
}

// nextFrame produces the next simulated frame payload. Most cycles it's a
// fresh frame; occasionally it re-emits the prior frame's exact bytes,
// simulating looped/replayed footage.
func (c *Camera) nextFrame() []byte {
	if c.lastFrame != nil && randN(5) == 0 {
		// Looped footage: identical bytes to the previous frame, so its
		// checksum will exactly match too -- this is precisely what a
		// duplicate-detection processor should catch.
		return c.lastFrame
	}

	c.frameNum++
	frame := []byte(fmt.Sprintf(
		`{"frame_id":%d,"resolution":%q}`,
		c.frameNum, c.resolution,
	))
	c.lastFrame = frame
	return frame
}

func (c *Camera) Listen(ctx context.Context) (<-chan Event, <-chan error) {
	events := make(chan Event)
	errs := make(chan error)

	go func() {
		defer close(events)
		defer close(errs)

		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				payload := c.nextFrame()
				ev, err := c.Sign(payload, map[string]string{
					"resolution": c.resolution,
				})
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

func (c *Camera) Close() error { return nil }
