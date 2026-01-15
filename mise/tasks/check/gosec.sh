#!/usr/bin/env bash
#MISE description="Check for security problems in go code"
set -euo pipefail

go tool gosec --exclude-generated -terse ./...
