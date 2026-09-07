# Kubernetes Storage Service Provider

A [DCM](https://github.com/dcm-project) service provider for managing persistent
storage volumes on Kubernetes clusters using `PersistentVolumeClaim` resources.

## Overview

This service provider maps the portable `storage` service type to Kubernetes
PVCs. It exposes an AEP-compliant REST API, registers as an external provider
with the DCM environment-agent, and publishes volume lifecycle status as
CloudEvents on NATS.

See the [k8s-storage-sp enhancement](https://github.com/dcm-project/enhancements/blob/main/enhancements/k8s-storage-sp/k8s-storage-sp.md)
for the full design.

## Features

- **Volume lifecycle** — create, read, and delete volumes via REST API (v1; no
  UPDATE/day-2 capacity expansion)
- **Kubernetes-native** — each volume maps to a `PersistentVolumeClaim`
- **Portable contract** — implements the DCM `storage` service type with
  `provider_hints.kubernetes` for StorageClass, volume mode, and access mode
- **Status monitoring** — watches PVCs and publishes status changes via
  CloudEvents on NATS subject `dcm.storage`
- **Auto-registration** — registers with the environment-agent provider registry
  on startup (`POST /api/v1alpha1/providers`), with exponential backoff retry
- **Health check** — exposes a resource-relative health endpoint for agent health polling
- **AEP-compliant API** — OpenAPI v1alpha1 contract with request validation
- **RFC 9457 errors** — problem details for all error responses

## Development

### Prerequisites

- Go 1.26.0+
- `make`
- `golangci-lint` (for `make lint`)
- NATS server (for status event publishing when running the SP)

### Build

```bash
make build
```

### Test

```bash
make test
```

### Lint

```bash
make lint       # Run golangci-lint
make check      # fmt + vet + lint + test (full validation)
```

### Container Image

```bash
make image-build # Build container image using podman/docker
```

### Code Generation

```bash
make generate-api         # Regenerate types, server, and client from OpenAPI
make check-generate-api   # Verify generated code is up to date (CI)
make check-aep            # Validate OpenAPI against AEP (requires spectral)
```

Generated files (do not edit manually):

- `api/v1alpha1/types.gen.go`
- `api/v1alpha1/spec.gen.go`
- `internal/api/server/server.gen.go`
- `pkg/client/client.gen.go`

## Configuration

Copy `.env.example` and adjust for your environment.

### Provider identity

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SP_NAME` | **Yes** | — | Provider name (unique per agent) |
| `SP_ENDPOINT` | **Yes** | — | Base URL reachable **from the environment-agent** (e.g. `http://k8s-storage-service-provider:8080` in compose, `http://host.docker.internal:8089` when the agent runs in a container and the SP on the host) |
| `SP_DISPLAY_NAME` | No | — | Human-readable provider name |
| `SP_REGION` / `SP_ZONE` | No | — | Optional metadata in the registration payload |

Registered API path is `{SP_ENDPOINT}/api/v1alpha1/volumes` (derived from OpenAPI).

### Registration (environment-agent)

Standalone SPs register with the **environment-agent**, not the control-plane.
Set `DCM_REGISTRATION_URL` to the agent API base; the registrar appends `/providers`.

| Variable | Required | Example (compose network) | Example (local agent on host) |
|----------|----------|---------------------------|-------------------------------|
| `DCM_REGISTRATION_URL` | **Yes** | `http://environment-agent:8080/api/v1alpha1` | `http://localhost:8090/api/v1alpha1` |

Do **not** use `http://control-plane:8080/api/v1alpha1` — the control-plane no longer
exposes `POST /providers` (registration returns HTTP 404).

When the agent runs embedded storage (`AGENT_EMBEDDED_SPS=storage`), do not deploy a
standalone storage SP against the same agent (only one storage provider per agent).

### Kubernetes

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SP_K8S_NAMESPACE` | No | `default` | Namespace for PVCs |
| `SP_K8S_KUBECONFIG` | No | in-cluster config | Path to kubeconfig |
| `SP_K8S_DEFAULT_STORAGE_CLASS` | No | — | Fallback when `provider_hints.kubernetes.storage_class` is omitted |
| `SP_K8S_DEFAULT_ACCESS_MODE` | No | `ReadWriteOnce` | Fallback access mode |

### NATS

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SP_NATS_URL` | **Yes** | — | NATS server URL for publishing status events (e.g. `nats://nats:4222`) |

### Monitoring

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SP_MONITOR_DEBOUNCE_MS` | No | `500` | Debounce interval in milliseconds before publishing a status change |
| `SP_MONITOR_RESYNC_PERIOD` | No | `10m` | How often the informer performs a full resync of PVCs |
| `SP_MONITOR_PUBLISH_MAX_ATTEMPTS` | No | `5` | Max attempts when publishing a status event to NATS fails |

## API

Contract: `api/v1alpha1/openapi.yaml`

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1alpha1/volumes/health` | Health check |
| POST | `/api/v1alpha1/volumes` | Create volume |
| GET | `/api/v1alpha1/volumes` | List volumes |
| GET | `/api/v1alpha1/volumes/{volume_id}` | Get volume |
| DELETE | `/api/v1alpha1/volumes/{volume_id}` | Delete volume |

## Project Structure

```
.
├── api/v1alpha1/              # OpenAPI spec and generated types
├── cmd/k8s-storage-service-provider/
├── internal/
│   ├── api/server/            # Generated strict server interface
│   ├── apiserver/
│   ├── config/
│   ├── handlers/
│   ├── kubernetes/
│   ├── monitoring/            # PVC informer + CloudEvents over NATS
│   └── registration/
├── pkg/client/                # Generated HTTP client
├── .ai/
│   ├── specs/
│   └── test-plans/
└── Makefile
```

### Releasing

Images are pushed to `quay.io/dcm-project/k8s-storage-service-provider`.
See [Releasing](https://github.com/dcm-project/shared-workflows#release-flow)
in shared-workflows for the full release process, tag behavior, and version conventions.

## License

Apache License 2.0 — see [LICENSE](LICENSE).
