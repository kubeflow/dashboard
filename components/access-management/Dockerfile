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

# Force download and extract all HashiCorp modules so they exist under /go/pkg/mod
RUN set -e; cd /workspace; \
  for m in $(go list -m all | awk '/^github.com\/hashicorp\// {print $1 "@" $2}'); do \
    # download with retry (these downloads are flakey) \
    for i in $(seq 1 5); do \
      if go mod download "$m"; then \
        echo "INFO: Successfully downloaded $m"; \
        break; \
      fi; \
      if [ $i -eq 5 ]; then \
        echo "ERROR: Failed to download $m after $i attempts"; \
        exit 1; \
      fi; \
      echo "WARN: Retrying download of $m (attempt $i)"; \
      sleep $((i * 2)); \
    done; \
    pkg="${m%@*}"; \
    go list "$pkg/..." >/dev/null 2>&1 || true; \
  done

# Copy the go source
COPY access-management/main.go main.go
COPY access-management/api/ api/
COPY access-management/kfam/ kfam/
COPY access-management/pkg/ pkg/

# Build the Go application
# NOTE: we set the GOARCH to TARGETARCH, but the build stage's FROM sets --platform to BUILDPLATFORM.
#       this lets us use go's cross-compilation ability to build for a forign architeture,
#       which is much faster than building under emulation.
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
