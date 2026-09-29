#!/usr/bin/env bash
# Build the device-gateway binary.

#Bash strict mode to exit on errors
set -euo pipefail

cd "$(dirname "$0")/../services/device-gateway"

echo "==> Building device-gateway"
GOPROXY=direct GOSUMDB=off go build -o device-gateway ./cmd

echo "==> Built: services/device-gateway/device-gateway"
