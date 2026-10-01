#!/usr/bin/env bash
set -euo pipefail

VALUES_FILE=${1:?usage: set-chart-image-repository.sh <values-file> <repository>}
export IMAGE_REPOSITORY=${2:?usage: set-chart-image-repository.sh <values-file> <repository>}

yq -e '.image.repository == strenv(IMAGE_REPOSITORY)' "$VALUES_FILE" >/dev/null 2>&1 || \
  yq -i '.image.repository = strenv(IMAGE_REPOSITORY)' "$VALUES_FILE"
