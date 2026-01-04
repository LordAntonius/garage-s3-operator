# Garage S3 Operator

[![Image](https://img.shields.io/badge/image-ghcr.io%2Flordantonius%2Fgarage--s3--operator-blue)](https://ghcr.io/lordantonius/garage-s3-operator)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)

Garage is a Kubernetes operator that makes it easy to provision and manage Garage S3 storage resources for applications running on your cluster. The project (Garage) aims to provide a simple CR-driven model to create S3 buckets and manage access credentials.

More info: https://garagehq.deuxfleurs.fr

## Key Features

- **CR-driven provisioning**: Define storage resources using Kubernetes Custom Resources.
- **Credential management**: Automatic creation and rotation of S3 credentials (secrets) scoped to resources.
- **Bucket lifecycle**: Express common lifecycle rules (retention, expiration) via CR fields.
- **Kubernetes-native**: Integrates with Secrets, and standard k8s tooling.

## Intent & Scope

This operator is intended to be a cluster-native operator that: provision S3 buckets on S3-compatible endpoints, manage credentials as Kubernetes Secrets, and expose an easy-to-use API for developers and platform operators. This repository hosts the operator source and related manifests.

## Installation

### Prerequisites

- A Kubernetes cluster (v1.20+ recommended).
- `kubectl` configured to access the cluster.

### Kustomize
Install the CRDs and operator manifests via:
```bash
kubectl apply -k ./config/default
```

The default used tag is 1.0.0.
It can be changed using overlays.

## Quickstart

1. Create Garage S3 instance corresponding to your S3 installation:

```yaml
apiVersion: garage-s3-operator.abucquet.com/v1
kind: GarageS3Instance
metadata:
  name: garage-instance
  namespace: garage
spec:
  # Accessible URL of the Garage S3 instance (optional, default: 127.0.0.1)
  url: "garage.garage.svc.cluster.local"
  # Admin API port (optional, default: 3903)
  port: 3903
  # Name of the Secret containing the admin token (required field)
  adminTokenSecret: garage-admin-token
```

Kubernetes `adminTokenSecret` must have the following structure:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: example-admin-token
  namespace: garage
type: Opaque
stringData:
  # Replace the value below with the real admin token.
  token: <YOUR-REAL-ADMIN-TOKEN>
```

2. Create S3 buckets:

```yaml
apiVersion: garage-s3-operator.abucquet.com/v1
kind: GarageS3Bucket
metadata:
  name: my-bucket
  namespace: default
spec:
  # Name of the GarageS3Instance this Access Key refers to (required field)
  instanceRef:
    name: garage-instance
    namespace: garage
  permissions:
  - accessKeyName: alice
    owner: true
    read: true
    write: true
  # quota:
  #   maxBytes: 10000 # in bytes
  #   maxObjects: 10
  # websiteAccess:
  #   enabled: true
  #   indexDocument: index.html
  #   errorDocument: 404.html
  # additionalAliases:
  #  - alice-bucket.garage.com
```

3. Create AccessKeys:
```yaml
apiVersion: garage-s3-operator.abucquet.com/v1
kind: GarageS3AccessKey
metadata:
  name: alice
  namespace: default
spec:
  # Name of the GarageS3Instance this Access Key refers to (required field)
  instanceRef:
    name: garage-instance
    namespace: garage
  # canCreateBucket: true
  # # Expiration in RFC3339 format. Example: 2026-01-30T12:00:00Z
  #expiration: "2035-01-30T12:00:00Z"
  # # Set to false to use the expiration above. If true, the key never expires.
  #neverExpires: false
```

## Development

Project layout follows common operator patterns (e.g. `config/`, `api/`, `controllers/`).

This repository includes a `Makefile` that provides convenient targets for building, running and testing locally as well as helpers to create a local `kind` cluster and deploy a test environment.

Common targets (run from repository root):

- `make build` — build a container image tagged `ghcr.io/lordantonius/garage-s3-operator:latest` (uses `podman` by default).
- `make push` — push the `latest` image to the registry.
- `make push-commit` — build then tag and push an image using the `VERSION` variable (e.g. `1.0.0`) set in the Makefile.
- `make run` — run the controller locally with `go run ./cmd/controller/*.go` (useful for local debugging against a cluster referenced by your kubeconfig).
- `make fmt` — run `gofmt -w .` to format the code.

Kind & test environment helpers:

- `make start-podman-kind` — create a `kind` cluster using the Podman provider (uses optional `KIND_CONFIG`).
- `make stop-podman-kind` — delete the `kind` cluster created above.
- `make deploy-garage` — start `kind` (via `start-podman-kind`) and deploy the Garage test components required by the integration environment.
- `make deploy-test-env` — deploy the Garage test environment and apply the `config/overlays/test` kustomize overlay.
- `make deploy` — apply `config/default` via `kubectl apply -k ./config/default`.
- `make clean` — stop the kind cluster (alias to `stop-podman-kind`).

Local development tips:

- Use `kind` or `k3d` for a quick local cluster.
- Use `kubectl logs -l app=garage-s3-operator -n garage-s3-operator` (adjust namespace if you changed it)

## Support & Contact

For questions, feature requests, or support, open an issue in this repository.

## License

This repository will use an open-source license. If no `LICENSE` file is present, please consult the repository owner or maintainer to add the appropriate license (e.g., MIT, Apache-2.0).
