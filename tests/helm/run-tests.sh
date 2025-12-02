#!/bin/bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Running Helm Chart End-to-End Tests"
echo "Note: Use 'make e2eHelmTest' from repository root for full setup"

go mod tidy
go test ./... -v -timeout=15m
