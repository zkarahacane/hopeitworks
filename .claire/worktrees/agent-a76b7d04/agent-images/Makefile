REGISTRY ?= hopeitworks
BASE_TAG ?= latest

.PHONY: base stacks all clean

base:
	docker build -t $(REGISTRY)/agent-base:$(BASE_TAG) -f base/Dockerfile ..

stacks: base
	docker build -t $(REGISTRY)/agent-go-node:$(BASE_TAG) -f stacks/go-node/Dockerfile .
	docker build -t $(REGISTRY)/agent-node:$(BASE_TAG) -f stacks/node/Dockerfile .
	docker build -t $(REGISTRY)/agent-go:$(BASE_TAG) -f stacks/go/Dockerfile .
	docker build -t $(REGISTRY)/agent-python:$(BASE_TAG) -f stacks/python/Dockerfile .

all: base stacks

clean:
	docker rmi $(REGISTRY)/agent-base:$(BASE_TAG) || true
	docker rmi $(REGISTRY)/agent-go-node:$(BASE_TAG) || true
	docker rmi $(REGISTRY)/agent-node:$(BASE_TAG) || true
	docker rmi $(REGISTRY)/agent-go:$(BASE_TAG) || true
	docker rmi $(REGISTRY)/agent-python:$(BASE_TAG) || true
