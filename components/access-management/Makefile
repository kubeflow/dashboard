REGISTRY_PROJECT ?= kubeflow-dev
PROJECT ?= kubeflow-dev
GO_SWAGGER_URL ?= https://github.com/go-swagger/go-swagger/releases/download/v0.21.0/swagger_linux_amd64

GIT_VERSION := $(shell git describe --tags --always --dirty)

IMG ?= ghcr.io/kubeflow/dashboard/access-management
TAG ?= $(GIT_VERSION)
ARCH ?= linux/amd64
generate-go-client: bin/swagger
	bin/swagger generate client -f api/swagger.yaml -t api/go_client

bin/swagger:
	mkdir -p bin
	wget -O bin/swagger '$(GO_SWAGGER_URL)'
	chmod +x bin/swagger

docker-build:
	docker build -t $(IMG):$(TAG) .

docker-push:
	docker push $(IMG):$(TAG)

.PHONY: docker-build-multi-arch
docker-build-multi-arch: ##  Build multi-arch docker images with docker buildx
	docker buildx build --load --platform ${ARCH} --tag ${IMG}:${TAG} .

.PHONY: docker-build-push-multi-arch
docker-build-push-multi-arch: ## Build multi-arch docker images with docker buildx and push to docker registry 
	docker buildx build --platform ${ARCH} --tag ${IMG}:${TAG} --push .

image: docker-build docker-push

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
ENVTEST_VERSION?=release-0.20
ENVTEST_K8S_VERSION = 1.23

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_VERSION)
