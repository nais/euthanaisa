#!/usr/bin/env bash
#MISE description="Check for dead go code"
set -euo pipefail

go tool golang.org/x/tools/cmd/deadcode -test ./...
