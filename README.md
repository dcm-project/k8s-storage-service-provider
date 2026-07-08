# Kubernetes Storage Service Provider

A [DCM](https://github.com/dcm-project) service provider for managing persistent
storage volumes on Kubernetes clusters using `PersistentVolumeClaim` resources.

## Overview

This repository bootstraps the K8s Storage SP with DCM-standard layout: OpenAPI
v1alpha1 contract, oapi-codegen generation, CI workflows, and AI specs/test
plans. Application implementation (handlers, K8s store, monitoring,
registration) lands in follow-up PRs.

See the [k8s-storage-sp enhancement](https://github.com/dcm-project/enhancements/blob/main/enhancements/k8s-storage-sp/k8s-storage-sp.md)
for the full design.

## Features (target)

- Volume lifecycle: CREATE, READ, UPDATE (capacity expand), DELETE
- Portable `storage` service type mapped to PVCs
- Status reporting via CloudEvents on NATS subject `dcm.storage`
- DCM auto-registration and health polling

## Development

### Prerequisites

- Go 1.25.5+
- `make`

### Build

```bash
make build
```

### Test

```bash
make test
```

### Code Generation

```bash
make generate-api         # Regenerate types, server, and client from OpenAPI
make check-generate-api # Verify generated code is up to date (CI)
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
| GET | `/api/v1alpha1/health` | Health check |
| POST | `/api/v1alpha1/volumes` | Create volume |
| GET | `/api/v1alpha1/volumes` | List volumes |
| GET | `/api/v1alpha1/volumes/{volume_id}` | Get volume |
| PATCH | `/api/v1alpha1/volumes/{volume_id}` | Expand capacity |
| DELETE | `/api/v1alpha1/volumes/{volume_id}` | Delete volume |

## Project Structure

```
.
├── api/v1alpha1/              # OpenAPI spec and generated types
├── cmd/k8s-storage-service-provider/  # Entry point (bootstrap stub)
├── internal/api/server/       # Generated strict server interface
├── pkg/client/                # Generated HTTP client
├── .ai/
│   ├── specs/                 # Requirements specification
│   └── test-plans/            # Unit and integration test plans
└── Makefile
```

## License

Apache License 2.0 — see [LICENSE](LICENSE).
