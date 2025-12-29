# garage-init — initialize the layout of a Garage S3 cluster

This small project contains a Python application packaged in a container whose purpose is to initialize the layout of a Garage S3 cluster (see: `garagehq.deuxfleurs.fr`). The container talks to the Garage administration API (`/v2/GetClusterStatus`, `/v2/UpdateClusterLayout`, `/v2/ApplyClusterLayout`) to check node status and, if needed, push and apply an initial cluster layout (roles, capacity, zones, and tags).

## Contents and design

- `app.py`: main script that orchestrates HTTP calls to the Garage admin API.
- `config.py`: helper that reads configuration from environment variables and parameters and falls back to `/etc/garage.toml` to get admin token.
- `Dockerfile`: builds the container image and installs dependencies (managed with `uv`).

## Parameters (quick reference)

| Env var | Parameter | Type / format | Description |
|---|---|---:|---|
| `GARAGE_URL` | `--url` | string | Hostname or full URL of the Garage admin API. Can be a host (e.g. `garage.garage.svc.cluster.local`) or a URL with scheme (e.g. `http://garage.example:3903`). |
| `GARAGE_PORT` | `--port` | integer/string | Optional port to append when `GARAGE_URL` is a host without scheme. Example: `3903`. |
| `GARAGE_TOKEN` | `--token` | string | Bearer token used for admin API requests. If not provided, the container will try to read `token` from `/etc/garage.toml`. For security, prefer mounting a `Secret`. |
| `GARAGE_CAPACITY` | `--capacity` | string/integer | Capacity in bytes assigned during initialization (examples: `100M`, `1000000000`). Interpreted by the initializer according to service conventions. |

The initializer prefers parameters then environment variables.
Additionally, it can read token in `/etc/garage.toml` if mounted.

## Docker and Kubernetes usage (examples)

Container is available at address `ghcr.io/lordantonius/garage-init:latest`.
It can be used in Kubernetes as a Job.

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: garage-init
  namespace: garage
  labels:
    app: garage-init
spec:
  backoffLimit: 3
  activeDeadlineSeconds: 600
  ttlSecondsAfterFinished: 300
  template:
    metadata:
      labels:
        app: garage-init
    spec:
      restartPolicy: Never
      containers:
        - name: garage-init
          image: ghcr.io/lordantonius/garage-init:latest
          imagePullPolicy: Always
          env:
            - name: GARAGE_URL
              value: "garage.garage.svc.cluster.local"
            - name: GARAGE_PORT
              value: "3903"
            - name: GARAGE_CAPACITY
              value: "100M"
          volumeMounts:
            - name: garage-config
              mountPath: /etc/garage.toml
              subPath: garage.toml
      volumes:
        - name: garage-config
          configMap:
            name: garage-config
            items:
              - key: garage.toml
                path: garage.toml
```


