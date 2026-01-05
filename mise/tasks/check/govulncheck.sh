#!/usr/bin/env bash
#MISE description="Check for known vulnerabilities in go code"
set -euo pipefail

go tool golang.org/x/vuln/cmd/govulncheck ./...
