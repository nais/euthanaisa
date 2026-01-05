#!/usr/bin/env bash
#MISE description="Check for security problems in go code"
set -euo pipefail

go tool github.com/securego/gosec/v2/cmd/gosec --exclude-generated -terse ./...
