IMAGE ?= garage-init:local
BUILD_CONTEXT ?= .
DOCKER = podman
PUBLISH_IMAGE ?= ghcr.io/lordantonius/garage-init:latest

.PHONY: build run publish

build:
	$(DOCKER) build -t $(IMAGE) $(BUILD_CONTEXT)

run: build
	$(DOCKER) run --rm $(IMAGE)

test:
	uv run app.py --url 127.0.0.1 --port 3903 --token 9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d --capacity 100M

publish: build
	$(DOCKER) tag $(IMAGE) $(PUBLISH_IMAGE)
	$(DOCKER) push $(PUBLISH_IMAGE)
