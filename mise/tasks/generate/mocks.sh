#!/usr/bin/env bash
#MISE description="Generate mocks"
#MISE depends_post=["fmt"]
set -euo pipefail

find internal -type f -name "mock_*.go" -delete
go run github.com/vektra/mockery/v2 --config ./.configs/mockery.yaml
