#!/usr/bin/env bash
# Run the Go test suite for the device gateway.
#
# GOPROXY=direct / GOSUMDB=off are only needed in network-restricted
# environments that can't reach proxy.golang.org (e.g. this project's dev
# sandbox). On a machine with normal network access, plain `go test ./...`
# works fine without them -- they're harmless to leave in either way.
set -euo pipefail

cd "$(dirname "$0")/../services/device-gateway"

echo "==> Checking gofmt"
UNFORMATTED=$(gofmt -l .)
if [ -n "$UNFORMATTED" ]; then
	echo "The following files are not gofmt-formatted:"
	echo "$UNFORMATTED"
	echo "Run: gofmt -w ."
	exit 1
fi

echo "==> Running Go tests (with race detector)"
GOPROXY=direct GOSUMDB=off go test ./... -v -cover -race
