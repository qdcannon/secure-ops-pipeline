"""
Sensor anomaly processor.

Subscribes to signed motion-sensor events published by the Go device
gateway over ZeroMQ, verifies each event's Ed25519 signature and SHA-256
checksum (rejecting anything tampered or unsigned), and tracks a
per-device heartbeat to flag reporting gaps -- the concrete defense
against a sensor being disabled or "going silent."
"""

import base64
import hashlib
import json
import time
from datetime import datetime, timezone

import zmq
from cryptography.exceptions import InvalidSignature
from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey

ENDPOINT = "tcp://127.0.0.1:5555"

# If a device hasn't reported within this many seconds, flag a gap alert.
# The simulated gateway ticks every 2s, so 6s means "missed ~3 beats."
HEARTBEAT_GAP_THRESHOLD_SECONDS = 6

# Tracks last-seen timestamp per device, to detect silence.
last_seen: dict[str, float] = {}


def verify_event(event: dict) -> bool:
    """Verify checksum integrity and Ed25519 signature authenticity."""
    payload = base64.b64decode(event["payload"])
    checksum = hashlib.sha256(payload).hexdigest()
    if checksum != event["checksum"]:
        print(f"  [REJECTED] checksum mismatch for {event['device_id']}")
        return False

    try:
        pubkey = Ed25519PublicKey.from_public_bytes(base64.b64decode(event["public_key"]))
        signature = base64.b64decode(event["signature"])
        pubkey.verify(signature, checksum.encode())
    except InvalidSignature:
        print(f"  [REJECTED] invalid signature for {event['device_id']}")
        return False

    return True


def check_heartbeat_gap(device_id: str, now: float):
    """Flag if too much time has passed since this device last reported."""
    prev = last_seen.get(device_id)
    if prev is not None:
        gap = now - prev
        if gap > HEARTBEAT_GAP_THRESHOLD_SECONDS:
            print(
                f"  [ALERT] {device_id} was silent for {gap:.1f}s "
                f"(threshold {HEARTBEAT_GAP_THRESHOLD_SECONDS}s) -- "
                f"possible tampering or device failure"
            )
    last_seen[device_id] = now


def main():
    ctx = zmq.Context()
    sub = ctx.socket(zmq.SUB)
    sub.connect(ENDPOINT)
    sub.setsockopt_string(zmq.SUBSCRIBE, "")

    print(f"sensor-anomaly processor subscribed to {ENDPOINT}")
    print(f"heartbeat gap threshold: {HEARTBEAT_GAP_THRESHOLD_SECONDS}s\n")

    while True:
        msg = sub.recv()
        event = json.loads(msg)

        ts = datetime.now(timezone.utc).isoformat()
        device_id = event["device_id"]
        state = json.loads(base64.b64decode(event["payload"]))["state"]

        print(f"[{ts}] received event device={device_id} state={state}")

        if not verify_event(event):
            continue

        check_heartbeat_gap(device_id, time.time())


if __name__ == "__main__":
    main()
