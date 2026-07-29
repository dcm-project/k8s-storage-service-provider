# Kubernetes Storage Service Provider

A [DCM](https://github.com/dcm-project) service provider for managing persistent
storage volumes on Kubernetes clusters using `PersistentVolumeClaim` resources.

## Overview

This service provider maps the portable `storage` service type to Kubernetes
PVCs. It exposes an AEP-compliant REST API and registers with the DCM control
plane. Volume lifecycle status via CloudEvents is planned but not implemented yet.

See the [k8s-storage-sp enhancement](https://github.com/dcm-project/enhancements/blob/main/enhancements/k8s-storage-sp/k8s-storage-sp.md)
for the full design.

## Features

- **Volume lifecycle** — create, read, and delete volumes via REST API (v1; no
  UPDATE/day-2 capacity expansion)
- **Kubernetes-native** — each volume maps to a `PersistentVolumeClaim`
- **Portable contract** — implements the DCM `storage` service type with
  `provider_hints.kubernetes` for StorageClass, volume mode, and access mode
- **Status monitoring** — watches PVCs and publishes status changes via
  CloudEvents on NATS subject `dcm.storage` *(not implemented yet)*
- **Auto-registration** — registers with the DCM Service Provider Manager on
  startup, with exponential backoff retry
- **Health check** — exposes a resource-relative health endpoint for DCM polling
- **AEP-compliant API** — OpenAPI v1alpha1 contract with request validation
- **RFC 9457 errors** — problem details for all error responses

## Development

### Prerequisites

- Go 1.26.0+
- `make`
- `golangci-lint` (for `make lint`)

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
│   └── registration/
├── pkg/client/                # Generated HTTP client
├── .ai/
│   ├── specs/
│   └── test-plans/
└── Makefile
```

## License

Apache License 2.0 — see [LICENSE](LICENSE).
