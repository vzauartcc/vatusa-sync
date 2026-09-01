#!/usr/bin/env bash

set -euo pipefail

if ! command -v golangci-lint &> /dev/null; then
    echo "Error: golangci-lint not found. Install it first."
    exit 1
fi

golangci-lint run "$@"