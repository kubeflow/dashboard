ARG GOLANG_VERSION=1.22
FROM --platform=${BUILDPLATFORM} golang:${GOLANG_VERSION} AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Copy the Go Modules manifests
COPY access-management/go.mod go.mod
COPY access-management/go.sum go.sum

# Copy profile-controller directory
COPY profile-controller /workspace/profile-controller

# Rewrite the go.mod to update the replace directive and download dependencies
RUN go mod edit -replace=github.com/kubeflow/dashboard/components/profile-controller=./profile-controller && \
    go mod download

# Copy the go source
COPY access-management/main.go main.go
COPY access-management/api/ api/
COPY access-management/kfam/ kfam/
COPY access-management/pkg/ pkg/

# Build the Go application
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o access-management main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot AS serve

WORKDIR /
COPY access-management/third_party third_party
COPY --from=builder /workspace/access-management .
COPY --from=builder /go/pkg/mod/github.com/hashicorp third_party/library/

USER 65532:65532

EXPOSE 8081

CMD ["/access-management"]
