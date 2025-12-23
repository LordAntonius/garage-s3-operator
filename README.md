# Garage S3 Operator

Garage is a Kubernetes operator that makes it easy to provision and manage Garage S3 storage resources for applications running on your cluster. The project (Garage) aims to provide a simple CR-driven model to create S3 buckets, manage access credentials, lifecycle policies, and integrate with on-prem or cloud S3-compatible backends.

More info: https://garagehq.deuxfleurs.fr

## Key Features

- **CR-driven provisioning**: Define storage resources using Kubernetes Custom Resources.
- **Credential management**: Automatic creation and rotation of S3 credentials (secrets) scoped to resources.
- **Bucket lifecycle**: Express common lifecycle rules (retention, expiration) via CR fields.
- **Kubernetes-native**: Integrates with RBAC, Secrets, and standard k8s tooling.

## Intent & Scope

This operator is intended to be a cluster-native operator that: provision S3 buckets on S3-compatible endpoints, manage credentials as Kubernetes Secrets, and expose an easy-to-use API for developers and platform operators. This repository hosts the operator source and related manifests.

This project is currently in active development — consider this README a living document.

## Prerequisites

- A Kubernetes cluster (v1.20+ recommended).
- `kubectl` configured to access the cluster.
- An S3-compatible endpoint and credentials (unless you're running a local backend like MinIO).
- (Optional) `helm` or `operator-sdk` if you plan to build and deploy from source.

## Quickstart (example)

1. Install the CRDs and operator manifests (example):

```bash
# Apply CRDs and controller manifest (replace with packaged YAML or helm chart when available)
kubectl apply -f config/crd/bases/
kubectl apply -f config/manager/manager.yaml
```

2. Create a sample S3 resource (example custom resource):

```yaml
apiVersion: garage.deuxfleurs.fr/v1alpha1
kind: S3Bucket
metadata:
	name: demo-bucket
spec:
	backend:
		endpoint: "https://minio.example.local"
		region: "us-east-1"
	bucketName: demo-bucket
	versioning: true
	lifecycle:
		- id: expire-logs
			prefix: logs/
			expirationDays: 30
	credentials:
		createSecret: true
		secretName: demo-bucket-credentials
```

Apply the resource:

```bash
kubectl apply -f examples/s3bucket-sample.yaml
```

3. Inspect created objects:

```bash
kubectl get s3buckets
kubectl get secrets demo-bucket-credentials -o yaml
```

Note: The CRD `S3Bucket` and field names above are an example schema to illustrate usage. Check the actual CRD definitions in `config/crd/` for the exact API and field names once the operator is generated or installed from a released bundle.

## Installation Options

- Install from a release (TBD): When releases are published we will provide a `kubectl apply -f` bundle and a Helm chart.
- Install from source:

```bash
# build the operator binary or container image
# (example flow; project may use Go + controller-runtime / operator-sdk)
make build
make docker-build IMAGE=registry.example.com/garage-s3-operator:dev
make deploy IMG=registry.example.com/garage-s3-operator:dev
```

Replace the commands above with the repository's specific build and deploy steps once the project contains the build tooling.

## Development

- Project layout will follow common operator patterns (e.g. `config/`, `api/`, `controllers/`).
- Recommended stack: Go + controller-runtime (operator-sdk), plus unit & integration tests.
- Run unit tests:

```bash
go test ./...
```

- Local development tips:
	- Use `kind` or `k3d` for a quick local cluster.
	- Use `kubectl port-forward` and logs to debug the controller: `kubectl logs -l control-plane=controller-manager -n system` (adjust selector for actual deployment).

## CRD Examples & Best Practices

- Keep resource names predictable and include environment/context when needed (e.g., `teamA-backup-bucket`).
- Store credentials in Kubernetes `Secrets` and limit RBAC to the least privilege required.
- Use lifecycle settings to control retention and reduce storage costs.

## Roadmap / Next Steps

- Define and publish stable CRD schema and examples.
- Provide a Helm chart and release bundles for easy installation.
- Add automated tests and CI for builds and integration tests.
- Implement credential rotation, multi-backend connectors, and metrics.

## Support & Contact

For questions, feature requests, or support, open an issue in this repository.

## License

This repository will use an open-source license. If no `LICENSE` file is present, please consult the repository owner or maintainer to add the appropriate license (e.g., MIT, Apache-2.0).

---

If you'd like, I can also:
- generate example CRD YAMLs under `config/crd/` and an example `examples/` resource,
- scaffold a basic Go operator layout using the controller-runtime/`operator-sdk`, or
- add a Helm chart and CI workflow for releases.

Tell me which next step you want and I'll proceed.
