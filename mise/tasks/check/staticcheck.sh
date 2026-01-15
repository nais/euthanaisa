#!/usr/bin/env bash
#MISE description="Check go code using static analysis"
set -euo pipefail

go tool staticcheck ./...
