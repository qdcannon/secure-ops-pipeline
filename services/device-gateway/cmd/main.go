// Command device-gateway wires configured device adapters into the
// Gateway and publishes their signed events over ZeroMQ for the
// Processor to consume.
//
// Which devices actually run is driven entirely by devices.json — this
// file never references a concrete adapter type (like MotionSensor)
// directly. Adding a new device instance, or even a new device type
// (once its adapter file exists and registers itself), never requires
// editing this file.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-zeromq/zmq4"

	"secure-ops-pipeline/services/device-gateway/adapter"
	"secure-ops-pipeline/services/device-gateway/gateway"
)

// deviceConfig mirrors one entry in devices.json.
type deviceConfig struct {
	Type   adapter.DeviceType `json:"type"`
	ID     string             `json:"id"`
	Config map[string]string  `json:"config"`
}

// devicesFile mirrors the top-level shape of devices.json.
type devicesFile struct {
	Devices []deviceConfig `json:"devices"`
}

// loadAdapters reads devices.json and builds one Adapter per entry via
// the registry (adapter.Build), rather than constructing concrete types
// directly. This is the only place in the whole gateway that knows
// devices.json exists.
func loadAdapters(path string) ([]adapter.Adapter, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var df devicesFile
	if err := json.Unmarshal(data, &df); err != nil {
		return nil, err
	}

	var adapters []adapter.Adapter
	for _, dc := range df.Devices {
		a, err := adapter.Build(dc.Type, dc.ID, dc.Config)
		if err != nil {
			//return nil, err
			continue //don't append unregistered device to adapter list.
		}
		adapters = append(adapters, a)
	}
	return adapters, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pub := zmq4.NewPub(ctx) //create new zmq publisher
	defer pub.Close()

	endpoint := "tcp://127.0.0.1:5555" //currently localhost should eventually change to container endpoint
	if err := pub.Listen(endpoint); err != nil {
		log.Fatalf("failed to bind PUB socket on %s: %v", endpoint, err)
	}
	log.Printf("device-gateway publishing on %s", endpoint)

	// Give any slow-starting subscriber a moment to connect before the
	// first event goes out (ZeroMQ PUB/SUB has no built-in "wait for
	// subscriber" handshake).
	time.Sleep(1 * time.Second)

	publish := func(ev adapter.Event) error {
		body, err := json.Marshal(ev)
		if err != nil {
			return err
		}
		log.Printf("-> published event device=%s type=%s ts=%s", ev.DeviceID, ev.DeviceType, ev.Timestamp.Format(time.RFC3339))
		return pub.Send(zmq4.NewMsg(body))
	}

	adapters, err := loadAdapters("devices.json")
	if err != nil {
		log.Fatalf("failed to load devices.json: %v", err)
	}
	log.Printf("loaded %d device(s) from devices.json", len(adapters))

	gw := gateway.New(publish)
	gw.Run(ctx, adapters)
}
