// Package gateway runs one goroutine per connected device adapter and
// fans their signed events into a single stream published over ZeroMQ.
package gateway

import (
	"context"
	"log"

	"secure-ops-pipeline/services/device-gateway/adapter"
)

type Gateway struct {
	publish func(adapter.Event) error // ZeroMQ publish hook
}

func New(publish func(adapter.Event) error) *Gateway {
	return &Gateway{publish: publish}
}

// Run starts one goroutine per adapter. Each adapter connects and streams
// its own signed events independently — a slow or disconnected camera
// never blocks a motion sensor's readings, since each device's I/O is
// isolated in its own goroutine.
func (g *Gateway) Run(ctx context.Context, adapters []adapter.Adapter) {
	events := make(chan adapter.Event, 100)

	for _, a := range adapters { //"_", is actual value of event "a"
		go g.runAdapter(ctx, a, events) //run go routine for current adapter
	}

	for {
		select {
		case ev := <-events: // got something from events channel
			if err := g.publish(ev); err != nil { // try to publsih event
				log.Printf("publish failed for device %s: %v", ev.DeviceID, err)
			}
		case <-ctx.Done(): //terminate if we program killed
			return
		}
	}
}

func (g *Gateway) runAdapter(ctx context.Context, a adapter.Adapter, out chan<- adapter.Event) {
	defer a.Close()

	if err := a.Connect(ctx); err != nil { //simulate conncting to hardware device
		log.Printf("connect failed for %s (%s): %v", a.DeviceID(), a.DeviceType(), err)
		return
	}

	evCh, errCh := a.Listen(ctx) // listen on events channel and listen on error channel
	for {
		select {
		case ev, ok := <-evCh: //get event and check if any problems with channel with "ok" boolean
			if !ok {
				return
			}
			out <- ev //send this on output only channel
		case err, ok := <-errCh:
			if !ok {
				continue
			}
			log.Printf("adapter warning for %s: %v", a.DeviceID(), err)
		case <-ctx.Done():
			return
		}
	}
}
