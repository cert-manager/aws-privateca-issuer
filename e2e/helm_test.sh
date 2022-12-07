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
  DIR="$( dirname -- "$0"; )"
  K8S_NAMESPACE="aws-privateca-issuer"
  AWS_REGION="us-east-1"
  DEPLOYMENT_NAME="aws-privateca-issuer"
}

clean_up() {
  set +e
  helm uninstall --namespace "$K8S_NAMESPACE" "$DEPLOYMENT_NAME" >/dev/null 2>&1
  kubectl delete namespace "$K8S_NAMESPACE" >/dev/null 2>&1

}

main() {
  HELM_CHART_NAME="$1"
  VALUES_FILE="$2"

  set -eo pipefail

  check_is_installed kubectl "kubectl is not installed"
  check_is_installed helm "helm is not installed"

  clean_up

  set -e

  echo "Installing the Helm Chart $HELM_CHART_NAME in namespace $K8S_NAMESPACE ... "

  helm repo add awspca https://cert-manager.github.io/aws-privateca-issuer

  helm install "$DEPLOYMENT_NAME" "$HELM_CHART_NAME" --create-namespace --namespace "$K8S_NAMESPACE" -f $VALUES_FILE 1>/dev/null || exit 1

  echo "Helm chart installed."

  DEPLOYMENT_NAME=$(kubectl get deployments -n $K8S_NAMESPACE -ojson | jq -r ".items[0].metadata.name")

  if [ -z "$DEPLOYMENT_NAME" ]; then
    echo "[ERROR] Found empty ACK controller deployment name. Exiting ..."
    exit 1
  fi

  echo "$DEPLOYMENT_NAME deployment found."

  POD_NAME=$(kubectl get pods -n $K8S_NAMESPACE -ojson | jq -r ".items[0].metadata.name")

  if [ -z "$POD_NAME" ]; then
    echo "[ERROR] Found empty ACK controller pod name. Exiting ..."
    exit 1
  fi

  # check if volume and volumeMount for 'cache-volume'
  POD_VOLUMES=$(kubectl get pod/"$POD_NAME" -n $K8S_NAMESPACE -ojson | jq -r '.spec.volumes[] | select( .name == "cache-volume" )')
  POD_VOLUME_MOUNTS=$(kubectl get pod/"$POD_NAME" -n $K8S_NAMESPACE -ojson | jq -r '.spec.containers[0].volumeMounts[] | select( .name == "cache-volume")')

  [ -z "$POD_VOLUMES" ] && echo "Volume 'cache-volume' has not been found" && exit 1
  [ -z "$POD_VOLUME_MOUNTS" ] && echo "Volume mount 'cache-volume' has not been found" && exit 1

  kubectl wait --for=condition=ready pod  "$POD_NAME" -n $K8S_NAMESPACE --timeout=30s 1>/dev/null || exit 1

  POD_STATUS=$(kubectl get pod/"$POD_NAME" -n $K8S_NAMESPACE -ojson | jq -r ".status.phase")
  [[ $POD_STATUS != Running ]] && echo "pod status is $POD_STATUS . Exiting ... " && exit 1
  echo "$POD_NAME pod found and status is $POD_STATUS"

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

  echo "uninstalling the Helm Chart $HELM_CHART_NAME in namespace $K8S_NAMESPACE ... "
  helm uninstall --namespace "$K8S_NAMESPACE" "$DEPLOYMENT_NAME" 1>/dev/null || exit 1

  echo "removing awspca repo"
  helm repo remove awspca

  echo "deleting $K8S_NAMESPACE namespace ... "
  kubectl delete namespace "$K8S_NAMESPACE" 1>/dev/null || exit 1

  echo "Helm Test Finished Successfully"

}

set_variables

if [ "$1" = "-p" ]; then
    echo "Running against prod ecr"
    main awspca/aws-privateca-issuer $DIR/test-values.yaml
    exit
fi

main $DIR/../charts/aws-pca-issuer $DIR/test-values.yaml
main awspca/aws-privateca-issuer $DIR/test-ecr.yaml
