# Secure Ops Pipeline

A distributed, cryptographically-attested ingest and processing platform
for physical security systems (cameras, motion sensors, and — as future
adapters — RFID/door access, alarms, and radios). Modeled on real PSIM
(Physical Security Information Management) platforms.

See [`docs/architecture.md`](docs/architecture.md) for the full design and
[`docs/roadmap.md`](docs/roadmap.md) for the phased build plan.

## Status: Phase 1 (in progress)

The `motion_sensor` device adapter and Go device-gateway are implemented
and working end-to-end against a Python processor that verifies each
event's signature/checksum and detects reporting gaps.

## Quickstart (local demo)

Requires Go 1.22+ and Python 3.10+.

```bash
# 1. Install Python dependencies
pip install -r requirements.txt

# 2. Start the processor (subscribes for events)
python3 services/processor/sensor_anomaly/main.py

# 3. In a second terminal, start the device gateway
#    (publishes signed, simulated motion-sensor events)
cd services/device-gateway
GOPROXY=direct GOSUMDB=off go run ./cmd
```

You should see the processor log each verified event, and periodically an
`[ALERT]` when the simulated sensor "goes silent" for longer than the
heartbeat threshold — the concrete defense against a sensor being disabled
or tampered with.

## Architecture at a glance

```
Device (camera / motion sensor)
        |  signed + checksummed payload
        v
Device Gateway (Go) -- goroutine per device, ZeroMQ pub
        |
        v
Processor (Python) -- verifies signature/checksum, detects
        |             tampering (dedup/loop) and anomalies (gaps)
        v
FastAPI + RBAC layer  (planned)
```

## Why Go for the gateway, Python elsewhere

The gateway's job is handling many independent, concurrent device
connections — a natural fit for goroutines/channels. The processor and API
layers are I/O-bound in a different way (business logic, DB/storage,
RBAC), where Python's ecosystem (FastAPI, crypto libs) is the better fit.
