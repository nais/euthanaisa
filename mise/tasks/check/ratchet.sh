#!/usr/bin/env bash
#MISE description="Check that all GitHub actions are pinned"
go tool github.com/sethvargo/ratchet lint .github/workflows/*.yaml
