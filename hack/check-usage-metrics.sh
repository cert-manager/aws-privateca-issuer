#!/usr/bin/env bash
set -euo pipefail

usage="usage: check-usage-metrics.sh <function-name> <version> <arch> [region]"
FUNCTION_NAME=${1:?$usage}
VERSION=${2:?$usage}
ARCH=${3:?$usage}
REGION=${4:-us-east-1}
ATTEMPTS=${ATTEMPTS:-20}
INTERVAL=${INTERVAL:-60}
LOOKBACK_MINUTES=${LOOKBACK_MINUTES:-60}
SINCE=${SINCE:-}

module_version() {
  awk -v module="$1" '$1 == module { sub(/^v/, "", $2); print $2; exit }' go.mod
}

sdk_version=$(module_version github.com/aws/aws-sdk-go-v2)
acmpca_version=$(module_version github.com/aws/aws-sdk-go-v2/service/acmpca)
go_version=$(sed -n 's/^FROM .*golang:\([0-9.]*\).*/\1/p' Dockerfile | head -n 1)

lookback_minutes() {
  if [[ -n "$SINCE" ]]; then
    echo $(( ($(date +%s) - SINCE + 59) / 60 ))
  else
    echo "$LOOKBACK_MINUTES"
  fi
}

build_payload() {
  jq -cn \
    --arg version "$VERSION" \
    --arg sdkVersion "$sdk_version" \
    --arg goVersion "$go_version" \
    --arg acmpcaVersion "$acmpca_version" \
    --arg region "$REGION" \
    --arg arch "$ARCH" \
    --argjson lookbackMinutes "$(lookback_minutes)" \
    '{version: $version, sdkVersion: $sdkVersion, goVersion: $goVersion, acmpcaVersion: $acmpcaVersion, region: $region, arch: $arch, lookbackMinutes: $lookbackMinutes}'
}

result=$(mktemp)
trap 'rm -f "$result"' EXIT

for ((attempt = 1; attempt <= ATTEMPTS; attempt++)); do
  payload=$(build_payload)
  echo "Querying $FUNCTION_NAME with $payload"
  meta=$(aws lambda invoke \
    --region us-east-1 \
    --function-name "$FUNCTION_NAME" \
    --cli-binary-format raw-in-base64-out \
    --payload "$payload" \
    "$result")
  if jq -e '.FunctionError' <<<"$meta" >/dev/null; then
    echo "$FUNCTION_NAME failed:" >&2
    cat "$result" >&2
    exit 1
  fi
  if jq -e '.found' "$result" >/dev/null; then
    jq . "$result"
    echo "Found usage metrics for $VERSION on $ARCH in $REGION"
    exit 0
  fi
  echo "Attempt $attempt/$ATTEMPTS: no usage metrics yet for $VERSION on $ARCH in $REGION"
  jq -c '.messages' "$result"
  if ((attempt < ATTEMPTS)); then
    sleep "$INTERVAL"
  fi
done

echo "No usage metrics for $VERSION on $ARCH in $REGION after $ATTEMPTS attempts" >&2
exit 1
