IMG=ghcr.io/lordantonius/garage-s3-operator:latest
DOCKER=podman

build:
	$(DOCKER) build -t $(IMG) .

push:
	$(DOCKER) push $(IMG)

run:
	go run ./cmd/controller/main.go

fmt:
	gofmt -w .
