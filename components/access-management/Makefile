# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/kubeflow/dashboard/access-management
TAG ?= $(shell git describe --tags --always --dirty)
ARCH ?= linux/amd64,linux/ppc64le,linux/arm64/v8

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet ## Run tests.
	go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: fmt vet ## Build kfam binary.
	go build -o bin/access-management main.go

.PHONY: run
run: fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build:
	# NOTE: we are building in the context of the parent directory (..)
	docker build --tag ${IMG}:${TAG} --file Dockerfile ..

.PHONY: docker-push
docker-push:
	docker push ${IMG}:${TAG}

.PHONY: docker-build-multi-arch
docker-build-multi-arch:
	# NOTE: we are building in the context of the parent directory (..)
	docker buildx build --platform ${ARCH} --tag ${IMG}:${TAG} --file Dockerfile ..

.PHONY: docker-build-push-multi-arch
docker-build-push-multi-arch:
	# NOTE: we are building in the context of the parent directory (..)
	docker buildx build --platform ${ARCH} --tag ${IMG}:${TAG} --file Dockerfile --push ..