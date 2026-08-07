// Command device-gateway wires configured device adapters into the
// Gateway and publishes their signed events over ZeroMQ for the
// Processor to consume.
package main

import (
	"context"
	"encoding/json"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-zeromq/zmq4"

	"secure-ops-pipeline/services/device-gateway/adapter"
	"secure-ops-pipeline/services/device-gateway/gateway"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pub := zmq4.NewPub(ctx)
	defer pub.Close()

	endpoint := "tcp://127.0.0.1:5555"
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

	sensor, err := adapter.NewMotionSensor("motion-sensor-001", 2*time.Second)
	if err != nil {
		log.Fatalf("failed to init motion sensor adapter: %v", err)
	}

	gw := gateway.New(publish)
	gw.Run(ctx, []adapter.Adapter{sensor})
}
