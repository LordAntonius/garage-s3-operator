IMG=ghcr.io/lordantonius/garage-s3-operator:latest
DOCKER=podman

KIND_CONFIG=
#KIND_CONFIG=--config ./hack/kind-config.yaml

build:
	$(DOCKER) build -t $(IMG) .

push:
	$(DOCKER) push $(IMG)

run:
	go run ./cmd/controller/*.go

fmt:
	gofmt -w .

start-podman-kind: stop-podman-kind
	KIND_EXPERIMENTAL_PROVIDER=podman kind create cluster --name garage-s3-operator $(KIND_CONFIG)

deploy-garage: start-podman-kind
	kubectl apply -k ./hack/kind-config/garage

stop-podman-kind:
	KIND_EXPERIMENTAL_PROVIDER=podman kind delete cluster --name garage-s3-operator || true

test: deploy-garage build push
	kubectl apply -f ./deploy/crd/

clean: stop-podman-kind

