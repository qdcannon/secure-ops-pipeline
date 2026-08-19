#!/usr/bin/env bash
# Run the sensor-anomaly processor. Creates/activates a local .venv and
# installs dependencies automatically if not already set up.
set -euo pipefail

cd "$(dirname "$0")/.."

if [ ! -d ".venv" ]; then
	echo "==> Creating virtual environment (.venv)"
	python3 -m venv .venv
fi

# shellcheck disable=SC1091
source .venv/bin/activate

echo "==> Installing/checking Python dependencies"
pip install -q -r requirements.txt

echo "==> Starting sensor-anomaly processor"
python3 services/processor/sensor_anomaly/main.py
