REGISTRY ?= hopeitworks
BASE_TAG ?= latest
# Multi-arch target platforms (parity with CI). Override for a single arch.
PLATFORMS ?= linux/amd64,linux/arm64
# Dedicated buildx builder so multi-arch is reproducible locally and in CI.
BUILDER ?= hopeitworks-agent-images

IMAGES = agent-base agent-go agent-node agent-go-node agent-python

.PHONY: base stacks all buildx-base buildx-stacks buildx buildx-setup digests clean

## --- Local single-arch builds (loadable into the local docker store) ---
base:
	docker build -t $(REGISTRY)/agent-base:$(BASE_TAG) -f base/Dockerfile ..

stacks: base
	docker build -t $(REGISTRY)/agent-go:$(BASE_TAG) -f stacks/go/Dockerfile .
	docker build -t $(REGISTRY)/agent-node:$(BASE_TAG) -f stacks/node/Dockerfile .
	docker build -t $(REGISTRY)/agent-go-node:$(BASE_TAG) -f stacks/go-node/Dockerfile .
	docker build -t $(REGISTRY)/agent-python:$(BASE_TAG) -f stacks/python/Dockerfile .

all: base stacks

## --- Multi-arch builds via buildx (linux/amd64,linux/arm64) ---
## Multi-arch manifest lists cannot be loaded into the local docker store,
## so these push. Set REGISTRY to a real registry (e.g. ghcr.io/...).
buildx-setup:
	docker buildx inspect $(BUILDER) >/dev/null 2>&1 || \
		docker buildx create --name $(BUILDER) --driver docker-container --use
	docker buildx use $(BUILDER)

buildx-base: buildx-setup
	docker buildx build --builder $(BUILDER) --platform $(PLATFORMS) \
		-t $(REGISTRY)/agent-base:$(BASE_TAG) -f base/Dockerfile --push ..

buildx-stacks: buildx-base
	docker buildx build --builder $(BUILDER) --platform $(PLATFORMS) \
		--build-context hopeitworks/agent-base=docker-image://$(REGISTRY)/agent-base:$(BASE_TAG) \
		-t $(REGISTRY)/agent-go:$(BASE_TAG) -f stacks/go/Dockerfile --push .
	docker buildx build --builder $(BUILDER) --platform $(PLATFORMS) \
		--build-context hopeitworks/agent-base=docker-image://$(REGISTRY)/agent-base:$(BASE_TAG) \
		-t $(REGISTRY)/agent-node:$(BASE_TAG) -f stacks/node/Dockerfile --push .
	docker buildx build --builder $(BUILDER) --platform $(PLATFORMS) \
		--build-context hopeitworks/agent-base=docker-image://$(REGISTRY)/agent-base:$(BASE_TAG) \
		-t $(REGISTRY)/agent-go-node:$(BASE_TAG) -f stacks/go-node/Dockerfile --push .
	docker buildx build --builder $(BUILDER) --platform $(PLATFORMS) \
		--build-context hopeitworks/agent-base=docker-image://$(REGISTRY)/agent-base:$(BASE_TAG) \
		-t $(REGISTRY)/agent-python:$(BASE_TAG) -f stacks/python/Dockerfile --push .

buildx: buildx-base buildx-stacks

## Print the (manifest list) digest of each published image.
digests:
	@for img in $(IMAGES); do \
		echo "== $$img =="; \
		docker buildx imagetools inspect $(REGISTRY)/$$img:$(BASE_TAG) 2>/dev/null | grep -i -E 'Name|Digest' || echo "  (not published)"; \
	done

clean:
	@for img in $(IMAGES); do \
		docker rmi $(REGISTRY)/$$img:$(BASE_TAG) || true; \
	done
