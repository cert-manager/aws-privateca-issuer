# The version which will be reported by the --version argument of each binary
# and which will be used as the Docker image tag
VERSION ?= $(shell git describe --tags)

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
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# BIN is the directory where tools will be installed
export BIN ?= ${CURDIR}/bin

OS := $(shell go env GOOS)
ARCH := $(shell go env GOARCH)

# Kind
KIND_VERSION := 0.9.0
KIND := ${BIN}/kind-${KIND_VERSION}
K8S_CLUSTER_NAME := pca-external-issuer

# cert-manager
CERT_MANAGER_VERSION ?= 1.3.0

# Controller tools
CONTROLLER_GEN_VERSION := 0.5.0
CONTROLLER_GEN := ${BIN}/controller-gen-${CONTROLLER_GEN_VERSION}

INSTALL_YAML ?= build/install.yaml

all: manager

# Run tests
ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: generate fmt vet lint manifests
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.0/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

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
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

.PHONY: ${INSTALL_YAML} kustomize
${INSTALL_YAML}: kustomize
	mkdir -p $(dir ${INSTALL_YAML})
	rm -rf kustomization.yaml
	$(KUSTOMIZE) create --resources ./config/default
	$(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build . > ${INSTALL_YAML}

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: ${INSTALL_YAML}
	kubectl apply -f ${INSTALL_YAML}

# UnDeploy controller from the configured Kubernetes cluster in ~/.kube/config
undeploy:
	$(KUSTOMIZE) build config/default | kubectl delete -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

lint: golangci-lint golint
	$(GOLANGCILINT) run --timeout 10m
	$(GOLINT) ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build \
		--build-arg pkg_version=${VERSION} \
		--tag ${IMG} \
		--file Dockerfile \
		${CURDIR}

# Push the docker image
docker-push:
	docker push ${IMG}

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen:
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

# Download kustomize locally if necessary
KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize:
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

GOLINT = $(shell pwd)/bin/golint
golint:
	$(call go-get-tool,$(GOLINT),golang.org/x/lint/golint)

GOLANGCILINT = $(shell pwd)/bin/golangci-lint
golangci-lint:
	$(call go-get-tool,$(GOLANGCILINT),github.com/golangci/golangci-lint/cmd/golangci-lint@v1.35.2)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
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
.PHONY: kind-cluster
kind-cluster: ## Use Kind to create a Kubernetes cluster for E2E tests
kind-cluster: ${KIND}
	 ${KIND} get clusters | grep ${K8S_CLUSTER_NAME} || ${KIND} create cluster --name ${K8S_CLUSTER_NAME}

.PHONY: kind-cluster-delete
kind-cluster-delete:
	${KIND} delete cluster --name ${K8S_CLUSTER_NAME}

.PHONY: kind-load
kind-load: ## Load all the Docker images into Kind
	${KIND} load docker-image --name ${K8S_CLUSTER_NAME} ${IMG}

.PHONY: kind-export-logs
kind-export-logs:
	${KIND} export logs --name ${K8S_CLUSTER_NAME} ${E2E_ARTIFACTS_DIRECTORY}

.PHONY: deploy-cert-manager
deploy-cert-manager: ## Deploy cert-manager in the configured Kubernetes cluster in ~/.kube/config
	kubectl apply --filename=https://github.com/jetstack/cert-manager/releases/download/v${CERT_MANAGER_VERSION}/cert-manager.yaml
	kubectl wait --for=condition=Available --timeout=300s apiservice v1.cert-manager.io

.PHONY: e2e
e2e:
	./test_utils/e2e_test.sh

#Run the unit tests, then setup the e2e test, run the tests, and then clean up
.PHONY: runtests
runtests: manager kind-cluster deploy-cert-manager docker-build kind-load deploy e2e kind-cluster-delete

# ==================================
# Download: tools in ${BIN}
# ==================================
${BIN}:
	mkdir -p ${BIN}

${KIND}: ${BIN}
	curl -sSL -o ${KIND} https://github.com/kubernetes-sigs/kind/releases/download/v${KIND_VERSION}/kind-${OS}-${ARCH}
	chmod +x ${KIND}
