#!/usr/bin/env bash
#MISE description="Create a binary for euthanaisa"
set -euo pipefail

go build -o ./bin/euthanaisa ./cmd/euthanaisa
