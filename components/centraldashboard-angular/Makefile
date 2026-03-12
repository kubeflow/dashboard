IMG ?= ghcr.io/kubeflow/dashboard/dashboard-angular
TAG ?= $(shell git describe --tags --always --dirty)
COMMIT = $(shell git rev-parse HEAD)
ARCH ?= linux/amd64,linux/ppc64le,linux/arm64/v8

COMMON_LIB_COMMIT ?= $(shell head -n 1 frontend/COMMON_LIB_COMMIT)
COMMON_LIB_TARGET_DIR = kubeflow-common-lib
COMMON_LIB_TMP_TGZ := /tmp/kubeflow-$(COMMON_LIB_COMMIT).tar.gz

.PHONY:fetch-common-lib
fetch-common-lib:
	curl -L -f -o $(COMMON_LIB_TMP_TGZ) https://github.com/kubeflow/kubeflow/archive/$(COMMON_LIB_COMMIT).tar.gz

	rm -rf $(COMMON_LIB_TARGET_DIR)
	mkdir -p $(COMMON_LIB_TARGET_DIR)
	tar -C /tmp -xzf $(COMMON_LIB_TMP_TGZ) kubeflow-$(COMMON_LIB_COMMIT)/components/crud-web-apps/common/frontend/kubeflow-common-lib/
	mv /tmp/kubeflow-$(COMMON_LIB_COMMIT)/components/crud-web-apps/common/frontend/kubeflow-common-lib/* $(COMMON_LIB_TARGET_DIR)/

	rm -f $(COMMON_LIB_TMP_TGZ)
	rm -rf /tmp/kubeflow-$(COMMON_LIB_COMMIT)

.PHONY: build-common-lib
build-common-lib: fetch-common-lib
	cd $(COMMON_LIB_TARGET_DIR) && npm i && npm run build && cd dist/kubeflow && npm link

.PHONY: docker-build
docker-build: fetch-common-lib
	docker build -t $(IMG):$(TAG) . \
	  --build-arg kubeflowversion=$(shell git describe --abbrev=0 --tags 2>/dev/null || echo "no-tag") \
	  --build-arg commit=$(COMMIT) \
	  --label=git-version=$(TAG)

.PHONY: docker-push
docker-push:
	docker push $(IMG):$(TAG)

.PHONY: docker-build-multi-arch
docker-build-multi-arch: fetch-common-lib ##  Build multi-arch docker images with docker buildx
	docker buildx build --platform ${ARCH} --tag ${IMG}:${TAG} . \
	  --build-arg kubeflowversion=$(shell git describe --abbrev=0 --tags 2>/dev/null || echo "no-tag") \
	  --build-arg commit=$(COMMIT) \
	  --label=git-version=$(TAG)

.PHONY: docker-build-push-multi-arch
docker-build-push-multi-arch: fetch-common-lib ## Build multi-arch docker images with docker buildx and push to docker registry
	docker buildx build --platform ${ARCH} --tag ${IMG}:${TAG} --push . \
	  --build-arg kubeflowversion=$(shell git describe --abbrev=0 --tags 2>/dev/null || echo "no-tag") \
	  --build-arg commit=$(COMMIT) \
	  --label=git-version=$(TAG)
