#!/usr/bin/env bash

set -euo pipefail

if ! command -v doppler &> /dev/null; then
    echo "Error: Doppler CLI not found. Install it first."
    exit 1
fi

export LOCAL_DEV_ENVIRONMENT="true"

doppler run -p roster-sync -c stg -- go run ./cmd/roster-sync/main.go "$@"