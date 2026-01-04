LATEST_IMG=ghcr.io/lordantonius/garage-s3-operator:latest
VERSION=1.0.0
COMMIT_IMG=ghcr.io/lordantonius/garage-s3-operator:$(VERSION)
DOCKER=podman

KIND_CONFIG=
#KIND_CONFIG=--config ./hack/kind-config.yaml

build:
	$(DOCKER) build -t $(LATEST_IMG) .

push:
	$(DOCKER) push $(LATEST_IMG)

push-commit: build
	$(DOCKER) tag $(LATEST_IMG) $(COMMIT_IMG)
	$(DOCKER) push $(COMMIT_IMG)

run:
	go run ./cmd/controller/*.go

fmt:
	gofmt -w .

start-podman-kind: stop-podman-kind
	KIND_EXPERIMENTAL_PROVIDER=podman kind create cluster --name garage-s3-operator $(KIND_CONFIG)

deploy-garage: start-podman-kind
	kubectl apply --server-side -k ./hack/kind-config/garage
	sleep 5
	kubectl wait --for=condition=ready pod -l app.kubernetes.io/instance=garage -n garage --timeout=1m
	kubectl apply -f ./hack/kind-config/garage-init-job.yaml 

stop-podman-kind:
	KIND_EXPERIMENTAL_PROVIDER=podman kind delete cluster --name garage-s3-operator || true

deploy-test-env: deploy-garage
	kubectl apply -k ./config/overlays/test

deploy:
	kubectl apply -k ./config/default
	
clean: stop-podman-kind

