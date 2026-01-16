#!/usr/bin/env bash
#MISE description="Check for known vulnerabilities in go code"
set -euo pipefail

go tool govulncheck ./...
