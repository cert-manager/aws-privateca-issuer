# The version which will be reported by the --version argument of each binary
# and which will be used as the Docker image tag
VERSION := $(shell git remote add mainRepo https://github.com/cert-manager/aws-privateca-issuer.git && git fetch mainRepo --tags && git describe --tags | awk -F"-" '{print $$1}' && git remote remove mainRepo)

# Default bundle image tag
BUNDLE_IMG ?= controller-bundle:$(VERSION)

# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

MAKEFLAGS += --warn-undefined-variables
SHELL := bash
.DELETE_ON_ERROR:
.SUFFIXES:
.ONESHELL:

# The Docker repository name, overridden in CI.
DOCKER_REGISTRY ?= ghcr.io
DOCKER_IMAGE_NAME ?= cert-manager/aws-privateca-issuer/controller

# Image URL to use all building/pushing image targets
IMG ?= ${DOCKER_REGISTRY}/${DOCKER_IMAGE_NAME}:${VERSION}
# Produce CRDs that work back to Kubernetes 1.13 (no version conversion)
CRD_OPTIONS ?= "crd"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GOCACHE ?= $(shell go env GOCACHE)
GOMODCACHE ?= $(shell go env GOMODCACHE)

# BIN is the directory where tools will be installed
export BIN ?= ${CURDIR}/bin

OS := $(shell go env GOOS)
ARCH := $(shell go env GOARCH)

# Kind
KIND_VERSION := 0.19.0
KIND := ${BIN}/kind-${KIND_VERSION}
K8S_CLUSTER_NAME := pca-external-issuer

# cert-manager
CERT_MANAGER_VERSION ?= v1.17.1

# Controller tools
CONTROLLER_GEN_VERSION := 0.17.2
CONTROLLER_GEN := ${BIN}/controller-gen-${CONTROLLER_GEN_VERSION}

# Helm tools
HELM_TOOL_VERSION := v0.2.2

INSTALL_YAML ?= build/install.yaml

all: manager

# Run tests
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: generate fmt vet lint manifests
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.0/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test -v ./pkg/... -coverprofile cover.out

e2etest: test
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.0/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test -v ./e2e/... -coverprofile cover.out

helm-test: manager kind-cluster
	$$SHELL e2e/helm_test.sh

blog-test:
	$$SHELL e2e/blog_test.sh

# Build manager binary
manager: generate fmt vet lint
	go build \
	-ldflags="-X github.com/cert-manager/aws-privateca-issuer/pkg/api/injections.PlugInVersion=${VERSION}" \
	-o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet lint manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f - --kubeconfig=${TEST_KUBECONFIG_LOCATION}

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f - --kubeconfig=${TEST_KUBECONFIG_LOCATION}

.PHONY: ${INSTALL_YAML} kustomize
${INSTALL_YAML}: kustomize
	mkdir -p $(dir ${INSTALL_YAML})
	rm -rf kustomization.yaml
	$(KUSTOMIZE) create --resources ./config/default
	$(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build . > ${INSTALL_YAML}

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: ${INSTALL_YAML}
	kubectl apply -f ${INSTALL_YAML} --kubeconfig=${TEST_KUBECONFIG_LOCATION}

# UnDeploy controller from the configured Kubernetes cluster in ~/.kube/config
undeploy:
	$(KUSTOMIZE) build config/default | kubectl delete -f - --kubeconfig=${TEST_KUBECONFIG_LOCATION}

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

helm-docs: helm-tool
	$(HELM_TOOL) inject -i charts/aws-pca-issuer/values.yaml -o charts/aws-pca-issuer/README.md --header-search "^<!-- AUTO-GENERATED -->" --footer-search "<!-- /AUTO-GENERATED -->"

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

lint:
	echo "Linter is deprecated with go1.18!"

#lint: golangci-lint golint
	#$(GOLANGCILINT) run --timeout 10m
	#$(GOLINT) ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build \
		--build-arg go_cache=${GOCACHE} \
		--build-arg go_mod_cache=${GOMODCACHE} \
		--build-arg pkg_version=${VERSION} \
		--tag ${IMG} \
		--file Dockerfile \
		--platform=linux/amd64,linux/arm64 \
		${CURDIR}

# Push the docker image
docker-push:
	docker push ${IMG}

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen:
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v$(CONTROLLER_GEN_VERSION))

HELM_TOOL = $(shell pwd)/bin/helm-tool
helm-tool:
	$(call go-install-tool,$(HELM_TOOL),github.com/cert-manager/helm-tool@$(HELM_TOOL_VERSION))

# Download kustomize locally if necessary
KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize:
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

GOLINT = $(shell pwd)/bin/golint
golint:
	echo "golint is deprecated, skipping"
	#$(call go-install-tool,$(GOLINT),golang.org/x/lint/golint)

GOLANGCILINT = $(shell pwd)/bin/golangci-lint
golangci-lint:
	$(call go-install-tool,$(GOLANGCILINT),github.com/golangci/golangci-lint/cmd/golangci-lint@v1.35.2)

# go-install-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-install-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests kustomize
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# ==================================
# E2E testing
# ==================================

REGISTRY_NAME := "kind-registry"
REGISTRY_PORT := 5000
LOCAL_IMAGE := "localhost:${REGISTRY_PORT}/aws-privateca-issuer"
NAMESPACE := aws-privateca-issuer
SERVICE_ACCOUNT := ${NAMESPACE}-${ARCH}-sa
TEST_KUBECONFIG_LOCATION := /tmp/pca_kubeconfig

create-local-registry:
	RUNNING=$$(docker inspect -f '{{.State.Running}}' ${REGISTRY_NAME} 2>/dev/null || true)
	if [ "$$RUNNING" != 'true' ]; then
		docker run -d --restart=always -p "127.0.0.1:${REGISTRY_PORT}:5000" --name ${REGISTRY_NAME} registry:2
	fi
	sleep 15

docker-push-local:
	docker tag ${IMG} ${LOCAL_IMAGE}
	docker push ${LOCAL_IMAGE}

.PHONY: kind-cluster
kind-cluster: ## Use Kind to create a Kubernetes cluster for E2E tests
kind-cluster: ${KIND}
	if [[ -z "$$OIDC_S3_BUCKET_NAME" ]]; then \
		echo "OIDC_S3_BUCKET_NAME env var is not set, the cluster will not be enabled for IRSA"; \
		echo "If you wish to have IRSA enabled, recreate the cluster with OIDC_S3_BUCKET_NAME  set"; \
		cp e2e/kind_config/config.yaml /tmp/config.yaml;
	else \
		cat e2e/kind_config/config.yaml | sed "s/S3_BUCKET_NAME_PLACEHOLDER/$$OIDC_S3_BUCKET_NAME/g" \
		> /tmp/config.yaml
	fi

	${KIND} get clusters | grep ${K8S_CLUSTER_NAME} || \
	${KIND} create cluster --name ${K8S_CLUSTER_NAME} --config=/tmp/config.yaml
	${KIND} get kubeconfig --name ${K8S_CLUSTER_NAME} > ${TEST_KUBECONFIG_LOCATION}
	docker network connect "kind" ${REGISTRY_NAME} || true
	kubectl apply -f e2e/kind_config/registry_configmap.yaml --kubeconfig=${TEST_KUBECONFIG_LOCATION}
	#Create namespace and service account
	kubectl get namespace ${NAMESPACE} --kubeconfig=${TEST_KUBECONFIG_LOCATION} || \
	kubectl create namespace ${NAMESPACE} --kubeconfig=${TEST_KUBECONFIG_LOCATION}
	kubectl get serviceaccount ${SERVICE_ACCOUNT} -n ${NAMESPACE} --kubeconfig=${TEST_KUBECONFIG_LOCATION} || \
	kubectl create serviceaccount ${SERVICE_ACCOUNT} -n ${NAMESPACE} --kubeconfig=${TEST_KUBECONFIG_LOCATION}

.PHONY: setup-eks-webhook
setup-eks-webhook:
	#Ensure that there is a OIDC role and S3 bucket available
	if [[ -z "$$OIDC_S3_BUCKET_NAME" || -z "$$OIDC_IAM_ROLE" ]]; then \
		echo "Please set OIDC_S3_BUCKET_NAME and  OIDC_IAM_ROLE to use IRSA"; \
		exit 1; \
	fi
	#Get open id configuration from API server
	kubectl apply -f e2e/kind_config/unauth_role.yaml --kubeconfig=${TEST_KUBECONFIG_LOCATION}
	APISERVER=$$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}' --kubeconfig=${TEST_KUBECONFIG_LOCATION})
	TOKEN=$$(kubectl get secret $(kubectl get serviceaccount default -o jsonpath='{.secrets[0].name}' --kubeconfig=${TEST_KUBECONFIG_LOCATION}) \
	-o jsonpath='{.data.token}' --kubeconfig=${TEST_KUBECONFIG_LOCATION} | base64 --decode )
	curl $$APISERVER/.well-known/openid-configuration --header "Authorization: Bearer $$TOKEN" --insecure -o openid-configuration
	curl $$APISERVER/openid/v1/jwks --header "Authorization: Bearer $$TOKEN" --insecure -o jwks
	#Put idP configuration in public S3 bucket
	aws s3 cp --acl public-read jwks s3://$$OIDC_S3_BUCKET_NAME/cluster/my-oidc-cluster/openid/v1/jwks
	aws s3 cp --acl public-read openid-configuration s3://$$OIDC_S3_BUCKET_NAME/cluster/my-oidc-cluster/.well-known/openid-configuration
	sleep 60
	kubectl apply -f e2e/kind_config/install_eks.yaml --kubeconfig=${TEST_KUBECONFIG_LOCATION}
	kubectl wait --for=condition=Available --timeout 300s deployment pod-identity-webhook --kubeconfig=${TEST_KUBECONFIG_LOCATION}
	kubectl annotate serviceaccount ${SERVICE_ACCOUNT} -n ${NAMESPACE} eks.amazonaws.com/role-arn=$$OIDC_IAM_ROLE --kubeconfig=${TEST_KUBECONFIG_LOCATION}

.PHONY: install-eks-webhook
install-eks-webhook: setup-eks-webhook upgrade-local

.PHONY: kind-cluster-delete
kind-cluster-delete:
	${KIND} delete cluster --name ${K8S_CLUSTER_NAME}

.PHONY: kind-export-logs
kind-export-logs:
	${KIND} export logs --name ${K8S_CLUSTER_NAME} ${E2E_ARTIFACTS_DIRECTORY}

.PHONY: deploy-cert-manager
deploy-cert-manager: ## Deploy cert-manager in the configured Kubernetes cluster in ~/.kube/config
	helm repo add jetstack https://charts.jetstack.io --force-update
	helm install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --version ${CERT_MANAGER_VERSION} --set crds.enabled=true --set config.apiVersion=controller.config.cert-manager.io/v1alpha1 --set config.kind=ControllerConfiguration --set config.kubernetesAPIQPS=10000 --set config.kubernetesAPIBurst=10000 --kubeconfig=${TEST_KUBECONFIG_LOCATION}
	kubectl wait --for=condition=Available --timeout=300s apiservice v1.cert-manager.io --kubeconfig=${TEST_KUBECONFIG_LOCATION}

.PHONY: install-local
install-local: docker-build docker-push-local
	#install plugin from local docker repo
	sleep 15
	helm install issuer ./charts/aws-pca-issuer -n ${NAMESPACE} \
	--set serviceAccount.create=false --set serviceAccount.name=${SERVICE_ACCOUNT} \
	--set image.repository=${LOCAL_IMAGE} --set image.tag=latest --set image.pullPolicy=Always

.PHONY: install-beta-ecr
install-beta-ecr: 
	#install plugin from local docker repo
	sleep 15
	helm install issuer ./charts/aws-pca-issuer -n ${NAMESPACE} \
	--set serviceAccount.create=false --set serviceAccount.name=${SERVICE_ACCOUNT} \
	--set image.repository=public.ecr.aws/cert-manager-aws-privateca-issuer/cert-manager-aws-privateca-issuer-test \
	--set image.tag=latest --set image.pullPolicy=Always

.PHONY: uninstall-local
uninstall-local:
	helm uninstall issuer -n ${NAMESPACE}

.PHONY: upgrade-local
upgrade-local: uninstall-local install-local

#Sets up a kind cluster using the latest commit on the current branch
.PHONY: cluster
cluster: manager create-local-registry kind-cluster deploy-cert-manager install-local

.PHONY: cluster-beta
cluster-beta: manager kind-cluster deploy-cert-manager install-beta-ecr
# ==================================
# Download: tools in ${BIN}
# ==================================
${BIN}:
	mkdir -p ${BIN}

${KIND}: ${BIN}
	curl -sSL -o ${KIND} https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-${OS}-${ARCH}
	chmod +x ${KIND}
