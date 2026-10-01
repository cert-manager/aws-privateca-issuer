#!/usr/bin/env bash
set -euo pipefail

REPO_URL=${1:?usage: wait-for-chart.sh <repo-url> <version>}
VERSION=${2:?usage: wait-for-chart.sh <repo-url> <version>}
ATTEMPTS=${ATTEMPTS:-40}
DELAY=${DELAY:-15}

for ((i = 1; i <= ATTEMPTS; i++)); do
  if helm show chart aws-privateca-issuer --repo "$REPO_URL" --version "$VERSION" >/dev/null 2>&1; then
    echo "Found aws-privateca-issuer $VERSION in $REPO_URL"
    exit 0
  fi
  echo "Waiting for aws-privateca-issuer $VERSION in $REPO_URL ($i/$ATTEMPTS)"
  sleep "$DELAY"
done

echo "aws-privateca-issuer $VERSION did not appear in $REPO_URL" >&2
exit 1
