IMG ?= ghcr.io/kubeflow/dashboard/dashboard
TAG ?= $(shell git describe --tags --always --dirty)
COMMIT = $(shell git rev-parse HEAD)
ARCH ?= linux/amd64,linux/ppc64le,linux/arm64/v8


all: build

clean:
	rm -rf coverage/ dist/ node_modules/ .nyc_output/

# Builds the package locally
build-local:
	npm install

# Runs unit tests with coverage
test: build-local
	npm run coverage

.PHONY: docker-build
docker-build:
	docker build -t $(IMG):$(TAG) . \
	  --build-arg kubeflowversion=$(shell git describe --abbrev=0 --tags 2>/dev/null || echo "no-tag") \
	  --build-arg commit=$(COMMIT) \
	  --label=git-version=$(TAG)

.PHONY: docker-push
docker-push:
	docker push $(IMG):$(TAG)

.PHONY: docker-build-multi-arch
docker-build-multi-arch: ##  Build multi-arch docker images with docker buildx
	docker buildx build --platform ${ARCH} --tag ${IMG}:${TAG} . \
	  --build-arg kubeflowversion=$(shell git describe --abbrev=0 --tags 2>/dev/null || echo "no-tag") \
	  --build-arg commit=$(COMMIT) \
	  --label=git-version=$(TAG)

.PHONY: docker-build-push-multi-arch
docker-build-push-multi-arch: ## Build multi-arch docker images with docker buildx and push to docker registry
	docker buildx build --platform ${ARCH} --tag ${IMG}:${TAG} --push . \
	  --build-arg kubeflowversion=$(shell git describe --abbrev=0 --tags 2>/dev/null || echo "no-tag") \
	  --build-arg commit=$(COMMIT) \
	  --label=git-version=$(TAG)
