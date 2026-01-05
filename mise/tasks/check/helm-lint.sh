#!/usr/bin/env bash
#MISE description="Validate Helm charts"
set -euo pipefail

helm lint --strict ./charts
