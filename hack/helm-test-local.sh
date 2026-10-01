#!/usr/bin/env bash
set -euo pipefail

KIND=${KIND:-kind}
K8S_CLUSTER_NAME=${K8S_CLUSTER_NAME:?K8S_CLUSTER_NAME must be set}
PORT=${PORT:-8879}
VERSION=$(git describe --tags)
IMAGE_REPOSITORY=aws-privateca-issuer
REPO_URL="http://127.0.0.1:$PORT"

REPO_ROOT=$(git rev-parse --show-toplevel)
WORK=$(mktemp -d)
SERVER_PID=
cleanup() {
  if [[ -n "$SERVER_PID" ]]; then
    kill "$SERVER_PID" 2>/dev/null || true
  fi
  rm -rf "$WORK"
}
trap cleanup EXIT

echo "==> building $IMAGE_REPOSITORY:$VERSION"
docker build --build-arg pkg_version="$VERSION" --tag "$IMAGE_REPOSITORY:$VERSION" "$REPO_ROOT"
"$KIND" load docker-image "$IMAGE_REPOSITORY:$VERSION" --name "$K8S_CLUSTER_NAME"

echo "==> packaging chart $VERSION"
cp -r "$REPO_ROOT/charts/aws-pca-issuer" "$WORK/chart"
"$REPO_ROOT/hack/set-chart-image-repository.sh" "$WORK/chart/values.yaml" "$IMAGE_REPOSITORY"
mkdir "$WORK/repo"
helm package "$WORK/chart" --version "$VERSION" --app-version "$VERSION" --destination "$WORK/repo"
helm repo index "$WORK/repo" --url "$REPO_URL"

echo "==> serving chart repository at $REPO_URL"
python3 -m http.server "$PORT" --bind 127.0.0.1 --directory "$WORK/repo" >/dev/null 2>&1 &
SERVER_PID=$!
for _ in $(seq 1 20); do
  curl -fsS "$REPO_URL/index.yaml" >/dev/null 2>&1 && break
  sleep 0.5
done

HELM_REPO="$REPO_URL" HELM_CHART_VERSION="$VERSION" EXPECTED_IMAGE="$IMAGE_REPOSITORY:$VERSION" "$REPO_ROOT/e2e/helm_test.sh"
