# Build the manager binary
FROM --platform=${BUILDPLATFORM} golang:1.22 as builder
WORKDIR /workspace

ARG TARGETARCH
ARG TARGETOS

ENV GOPROXY=direct
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY pkg/ pkg/

ENV CGO_ENABLED=0
ENV GOOS=${TARGETOS:-linux}
ENV GOARCH=${TARGETARCH:-amd64}
ENV GO111MODULE=on

# Do an initial compilation before setting the version so that there is less to
# re-compile when the version changes
RUN go build -mod=readonly ./...

ARG pkg_version

# Build
RUN VERSION=$pkg_version && \
    go build \
    -ldflags="-X=github.com/cert-manager/acm-pca-issuer/internal/version.Version=${VERSION} \
    -X github.com/cert-manager/aws-privateca-issuer/pkg/api/injections.PlugInVersion=${VERSION}" \
    -mod=readonly \
    -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
# FROM --platform=${TARGETPLATFORM:-linux/amd64} gcr.io/distroless/static:nonroot
FROM alpine:latest
RUN apk update && apk add --no-cache coreutils aws-cli
LABEL org.opencontainers.image.authors="Jochen Ullrich <kontakt@ju-hh.de>"
LABEL org.opencontainers.image.source=https://github.com/cert-manager/aws-privateca-issuer
WORKDIR /
COPY --from=builder /workspace/manager .

ENTRYPOINT ["/bin/sh", "-c", "/manager"]
