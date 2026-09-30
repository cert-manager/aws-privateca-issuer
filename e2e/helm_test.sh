#!/usr/bin/env bash

check_is_installed() {
  local __name="$1"
  local __extra_msg="$2"
  if ! is_installed "$__name"; then
    echo "FATAL: Missing requirement '$__name'"
    echo "Please install $__name before running this script."
    if [[ -n $__extra_msg ]]; then
      echo ""
      echo "$__extra_msg"
      echo ""
    fi
    exit 1
  else
    echo "$__name installed"
  fi
}

is_installed() {
  local __name="$1"
  if $(which $__name >/dev/null 2>&1); then
    return 0
  else
    return 1
  fi
}

get_num_columns() {
  echo "80"
}

print_line_separation() {
  local __num_cols=$(get_num_columns)
  printf %"$__num_cols"s\\n | tr " " "="
}

set_variables() {
  HELM_REPO="${HELM_REPO:?HELM_REPO must be set to the Helm repository URL to test}"
  HELM_CHART_VERSION="${HELM_CHART_VERSION:-}"
  EXPECTED_IMAGE="${EXPECTED_IMAGE:-}"
  HELM_REPO_NAME="awspca"
  K8S_NAMESPACE="default"
  HELM_CHART_NAME="$HELM_REPO_NAME/aws-privateca-issuer"
  HELM_INSTALL_ARGS=(--generate-name)
  if [[ -n "$HELM_CHART_VERSION" ]]; then
    HELM_INSTALL_ARGS+=(--version "$HELM_CHART_VERSION")
  fi
}

clean_up() {
  set +e
  helm list --all-namespaces -o json | jq -r '.[] | select(.chart | startswith("aws-privateca-issuer-")) | "\(.namespace) \(.name)"' | while read -r namespace release; do
    helm uninstall --namespace "$namespace" "$release" >/dev/null 2>&1
  done
  set -e
}

main() {
  set -eo pipefail

  check_is_installed kubectl "kubectl is not installed"
  check_is_installed helm "helm is not installed"
  check_is_installed jq "jq is not installed"

  set_variables

  clean_up

  echo "Adding Helm repository $HELM_REPO as $HELM_REPO_NAME ... "
  helm repo add "$HELM_REPO_NAME" "$HELM_REPO" --force-update 1>/dev/null
  helm repo update "$HELM_REPO_NAME" 1>/dev/null

  echo "Installing the Helm Chart $HELM_CHART_NAME in namespace $K8S_NAMESPACE ... "

  RELEASE_NAME=$(helm install ${HELM_INSTALL_ARGS[@]+"${HELM_INSTALL_ARGS[@]}"} "$HELM_CHART_NAME" --namespace "$K8S_NAMESPACE" -o json | jq -r ".name")

  echo "Helm release $RELEASE_NAME installed."
  trap 'helm uninstall --namespace "$K8S_NAMESPACE" "$RELEASE_NAME" >/dev/null 2>&1' EXIT

  SELECTOR="app.kubernetes.io/instance=$RELEASE_NAME"

  DEPLOYMENT_NAME=$(kubectl get deployments -n $K8S_NAMESPACE -l "$SELECTOR" -ojson | jq -r ".items[0].metadata.name // empty")

  if [ -z "$DEPLOYMENT_NAME" ]; then
    echo "[ERROR] No deployment found for release $RELEASE_NAME. Exiting ..."
    exit 1
  fi

  echo "$DEPLOYMENT_NAME deployment found."

  if ! kubectl rollout status deployment/"$DEPLOYMENT_NAME" -n $K8S_NAMESPACE --timeout=120s 1>/dev/null; then
    echo "[ERROR] $DEPLOYMENT_NAME did not become ready. Exiting ..."
    print_line_separation
    kubectl describe deployment/"$DEPLOYMENT_NAME" -n $K8S_NAMESPACE
    kubectl describe pods -n $K8S_NAMESPACE -l "$SELECTOR"
    kubectl get events -n $K8S_NAMESPACE --sort-by=.lastTimestamp
    print_line_separation
    exit 1
  fi

  POD_NAME=$(kubectl get pods -n $K8S_NAMESPACE -l "$SELECTOR" -ojson | jq -r ".items[0].metadata.name // empty")

  if [ -z "$POD_NAME" ]; then
    echo "[ERROR] No pod found for release $RELEASE_NAME. Exiting ..."
    exit 1
  fi

  POD_STATUS=$(kubectl get pod/"$POD_NAME" -n $K8S_NAMESPACE -ojson | jq -r ".status.phase")
  [[ $POD_STATUS != Running ]] && echo "pod status is $POD_STATUS . Exiting ... " && exit 1
  echo "$POD_NAME pod found and status is $POD_STATUS"

  POD_IMAGE=$(kubectl get pod/"$POD_NAME" -n $K8S_NAMESPACE -ojson | jq -r ".spec.containers[0].image")
  echo "Pod image is $POD_IMAGE"
  if [[ -n "$EXPECTED_IMAGE" && "$POD_IMAGE" != "$EXPECTED_IMAGE" ]]; then
    echo "[ERROR] Expected pod image $EXPECTED_IMAGE. Exiting ..."
    exit 1
  fi

  LOGS=$(kubectl logs pod/"$POD_NAME" -n $K8S_NAMESPACE)
  if [ -z "$LOGS" ]; then
    echo "[ERROR] No controller logs found for pod $POD_NAME. Exiting ..."
    exit 1
  fi
  echo "Logs found."

  if echo "$LOGS" | grep -q "ERROR"; then
    echo "[ERROR] Found following ERROR statements in controller logs."
    print_line_separation
    echo "$LOGS" | grep "ERROR"
    print_line_separation
    echo "[ERROR] Exiting ..."
    exit 1
  fi
  echo "No error statements found in Logs"

  echo "uninstalling Helm release $RELEASE_NAME in namespace $K8S_NAMESPACE ... "
  trap - EXIT
  helm uninstall --namespace "$K8S_NAMESPACE" "$RELEASE_NAME" 1>/dev/null || exit 1

  echo "Helm Test Finished Successfully"

}

main
