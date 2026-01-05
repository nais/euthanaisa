#!/usr/bin/env bash
#MISE description="Update versions for all GitHub actions"
set -euo pipefail

go tool github.com/sethvargo/ratchet update .github/workflows/*.yaml
