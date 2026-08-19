#!/usr/bin/env bash
# Run the device gateway (reads services/device-gateway/devices.json,
# publishes signed device events over ZeroMQ).
set -euo pipefail

cd "$(dirname "$0")/../services/device-gateway"

echo "==> Starting device-gateway"
GOPROXY=direct GOSUMDB=off go run ./cmd
