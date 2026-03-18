ARG GOLANG_VERSION=1.22
FROM --platform=${BUILDPLATFORM} golang:${GOLANG_VERSION} AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

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
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build the Go application
# NOTE: we set the GOARCH to TARGETARCH, but the build stage's FROM sets --platform to BUILDPLATFORM.
#       this lets us use go's cross-compilation ability to build for a forign architeture,
#       which is much faster than building under emulation.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY third_party third_party
COPY --from=builder /workspace/manager .
COPY --from=builder /go/pkg/mod/github.com/hashicorp third_party/library/

USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/manager"]
