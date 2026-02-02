#!/usr/bin/env bash

set -euo pipefail

if ! command -v doppler &> /dev/null; then
    echo "Error: Doppler CLI not found. Install it first."
    exit 1
fi

doppler run -p roster-sync -c stg -- go run ./cmd/roster-sync/main.go "$@"