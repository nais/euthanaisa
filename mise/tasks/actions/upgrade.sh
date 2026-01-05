#!/usr/bin/env bash
#MISE description="Upgrade all GitHub actions to their latest versions"
set -euo pipefail

go tool github.com/sethvargo/ratchet upgrade .github/workflows/*.yaml
