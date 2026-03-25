ARG GOLANG_VERSION=1.26
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

USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/manager"]
