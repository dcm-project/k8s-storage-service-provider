# Test Plan: K8s Storage SP — Integration Tests

## Overview

- **Related Spec:** .ai/specs/k8s-storage-sp.spec.md
- **Framework:** Ginkgo v2 + Gomega
- **Created:** 2026-07-02
- **Last Updated:** 2026-07-05

Integration tests verify components working together with **real external services**.
These tests require a real Kubernetes cluster (Kind), real NATS server, and real
StorageClasses configured.

**Real services required:**
- ✅ Kind Kubernetes cluster (or similar test cluster)
- ✅ NATS server (for CloudEvents)
- ✅ Multiple StorageClasses configured (ceph-rbd, nfs, etc.)
- ✅ SP running as real process

**What integration tests verify:**
- SP lifecycle (startup, registration, shutdown)
- Real PVC operations against Kubernetes
- Status reporting to real NATS
- Informer watches real PVC changes
- Volume expansion with real StorageClass constraints
- Multiple backend scenarios

---

## Test Environment Setup

### Required Infrastructure

1. **Kind Cluster**
   ```bash
   kind create cluster --name sp-test
   ```

2. **NATS Server**
   ```bash
   docker run -d -p 4222:4222 -p 8222:8222 nats:2-alpine --jetstream
   ```

3. **StorageClasses**
   - **ceph-rbd** (with `allowVolumeExpansion: true`)
   - **nfs** (with `allowVolumeExpansion: false`)
   - **default** (cluster default)

4. **SP Configuration (env vars)**
   ```bash
   SP_K8S_NAMESPACE=dcm-test
   SP_K8S_DEFAULT_STORAGE_CLASS=""   # cluster default
   SP_K8S_DEFAULT_ATTACHMENT_MODE=exclusive
   SP_NATS_URL=nats://localhost:4222
   DCM_REGISTRATION_URL=http://localhost:8080/api/v1alpha1
   SP_NAME=k8s-storage-sp
   SP_ENDPOINT=http://localhost:8080
   ```

### Test Data Fixtures

- Sample PVC manifests
- Sample StorageClass definitions
- Sample DCM requests (JSON payloads)

---

## 1 · Server Lifecycle

> **Suggested Ginkgo structure:** `Describe("Server Lifecycle")`

### TC-I001: SP starts up successfully

- **Priority:** High
- **Type:** Integration
- **Given:** Kind cluster is running and SP is not started
- **When:** SP process starts with valid configuration
- **Then:**
  - Process starts without errors
  - Logs indicate successful startup
  - Health endpoint becomes reachable

### TC-I002: All API endpoints are accessible

- **Priority:** High
- **Type:** Integration
- **Given:** SP is running
- **When:** HTTP requests are made to each endpoint
- **Then:**
  - `GET /api/v1alpha1/health` returns 200
  - `POST /api/v1alpha1/volumes` does not return 404/405
  - `GET /api/v1alpha1/volumes` does not return 404/405
  - `GET /api/v1alpha1/volumes/{id}` does not return 404/405
  - `PATCH /api/v1alpha1/volumes/{id}` does not return 404/405
  - `DELETE /api/v1alpha1/volumes/{id}` does not return 404/405

### TC-I003: SP shuts down gracefully on SIGTERM

- **Priority:** High
- **Type:** Integration
- **Given:** SP is running with active informer
- **When:** SIGTERM is sent to process
- **Then:**
  - Informer stops gracefully
  - In-flight requests complete
  - Process exits with code 0

### TC-I004: SP logs startup and shutdown events

- **Priority:** Medium
- **Type:** Integration
- **Given:** SP lifecycle (start → stop)
- **When:** Observing logs
- **Then:**
  - Startup log includes listen address
  - Shutdown log indicates graceful termination

---

## 2 · SP Registration

> **Suggested Ginkgo structure:** `Describe("SP Registration")`

### TC-I010: SP registers with DCM on startup

- **Priority:** High
- **Type:** Integration
- **Given:** 
  - DCM control-plane (or WireMock stub) is running
  - SP configured with DCM registrar URL
- **When:** SP starts
- **Then:**
  - POST request sent to DCM `/providers` endpoint
  - Request body includes:
    - `name`: configured provider name
    - `endpoint`: SP API URL
    - `service_type`: "storage"
    - `schema_version`: "v1alpha1"

### TC-I011: Health endpoint responds after registration

- **Priority:** High
- **Type:** Integration
- **Given:** SP has registered with DCM
- **When:** DCM polls `/health` endpoint
- **Then:**
  - Response: 200 OK
  - Body: `{"status": "healthy", "version": "...", "uptime": ...}`

---

## 3 · PVC Creation (Real Kubernetes)

> **Suggested Ginkgo structure:** `Describe("PVC Creation")`

### TC-I020: Create PVC with minimal spec

- **Priority:** High
- **Type:** Integration
- **Given:** Kind cluster is running, SP is running
- **When:** POST `/api/v1alpha1/volumes`:
  ```json
  {
    "capacity": "10Gi",
    "metadata": {"name": "test-vol-minimal"},
    "service_type": "storage"
  }
  ```
- **Then:**
  - Response: 201 Created with volume details
  - PVC exists in configured namespace (`dcm-test`)
  - PVC has correct labels:
    - `dcm.project/managed-by: dcm`
    - `dcm.project/dcm-instance-id: <uuid>`
    - `dcm.project/dcm-service-type: storage`
  - PVC `spec.resources.requests.storage = "10Gi"`
  - PVC uses cluster default StorageClass

### TC-I021: Create PVC with storage_class hint

- **Priority:** High
- **Type:** Integration
- **Given:** StorageClass `ceph-rbd` exists in cluster
- **When:** POST with `provider_hints.kubernetes.storage_class: "ceph-rbd"`
- **Then:**
  - PVC created with `spec.storageClassName: "ceph-rbd"`
  - PVC binds to PV (if provisioner is available)

### TC-I022: Create PVC with attachment_mode

- **Priority:** High
- **Type:** Integration
- **Given:** StorageClass supports ReadWriteMany
- **When:** POST with `attachment_mode: "multiReadWrite"`
- **Then:** PVC created with `spec.accessModes: ["ReadWriteMany"]`

### TC-I023: Create PVC with volume_mode hint

- **Priority:** High
- **Type:** Integration
- **Given:** Valid volume_mode value
- **When:** POST with `provider_hints.kubernetes.volume_mode: "Block"`
- **Then:** PVC created with `spec.volumeMode: "Block"`

### TC-I024: Create PVC with attachment_mode and provider_hints

- **Priority:** High
- **Type:** Integration
- **Given:** `attachment_mode`, `provider_hints.kubernetes.storage_class`, and `provider_hints.kubernetes.volume_mode` specified
- **When:** POST with all hints set
- **Then:** PVC created with all three settings applied correctly

### TC-I025: Create duplicate PVC name returns 409

- **Priority:** High
- **Type:** Integration
- **Given:** PVC with name `"app-data"` already exists
- **When:** POST with same name `"app-data"`
- **Then:**
  - Response: 409 Conflict
  - Error message indicates PVC already exists

### TC-I026: Create PVC with invalid StorageClass

- **Priority:** Medium
- **Type:** Integration
- **Given:** StorageClass `"non-existent"` does not exist
- **When:** POST with `storageClass: "non-existent"`
- **Then:**
  - Either: 422 Unprocessable Entity (if validated by SP)
  - Or: PVC created but stuck in Pending with error event

### TC-I027: Create multiple PVCs in same namespace

- **Priority:** High
- **Type:** Integration
- **Given:** SP configured with namespace `dcm-test`
- **When:** Create 5 PVCs with different names
- **Then:**
  - All 5 PVCs created successfully
  - All exist in namespace `dcm-test`
  - All have correct DCM labels

---

## 4 · PVC Reading (Real Kubernetes)

> **Suggested Ginkgo structure:** `Describe("PVC Reading")`

### TC-I030: GET single volume returns PVC details

- **Priority:** High
- **Type:** Integration
- **Given:** PVC exists with `dcm-instance-id = "abc-123"`
- **When:** GET `/api/v1alpha1/volumes/abc-123`
- **Then:**
  - Response: 200 OK
  - Response body includes:
    - `requestId: "abc-123"`
    - `name`: PVC name
    - `capacity`: PVC size
    - `status`: Current status (PROVISIONING or RUNNING)
    - `metadata.storageClass`: StorageClass name

### TC-I031: GET volume returns PROVISIONING while pending

- **Priority:** High
- **Type:** Integration
- **Given:** PVC exists but is not yet bound (Pending)
- **When:** GET volume
- **Then:** Response status field is `"PROVISIONING"`

### TC-I032: GET volume returns RUNNING when bound

- **Priority:** High
- **Type:** Integration
- **Given:** PVC is bound to PV
- **When:** GET volume
- **Then:**
  - Response status field is `"RUNNING"`
  - Response includes `volumeName` (PV name)

### TC-I033: GET volume returns 404 for non-existent

- **Priority:** High
- **Type:** Integration
- **Given:** No PVC with `dcm-instance-id = "invalid"`
- **When:** GET `/api/v1alpha1/volumes/invalid`
- **Then:** Response: 404 Not Found

### TC-I034: LIST volumes returns all managed PVCs

- **Priority:** High
- **Type:** Integration
- **Given:** 3 PVCs with DCM labels exist
- **When:** GET `/api/v1alpha1/volumes`
- **Then:**
  - Response: 200 OK
  - Response contains 3 volumes in results array

### TC-I035: LIST volumes filters by DCM labels only

- **Priority:** High
- **Type:** Integration
- **Given:**
  - 2 PVCs with `dcm.project/managed-by=dcm` label
  - 1 PVC without DCM labels (manually created)
- **When:** GET `/api/v1alpha1/volumes`
- **Then:** Response contains only the 2 PVCs with DCM labels

### TC-I036: LIST volumes with pagination

- **Priority:** Medium
- **Type:** Integration
- **Given:** 10 PVCs exist
- **When:** GET `/api/v1alpha1/volumes?max_page_size=5`
- **Then:**
  - Response contains 5 volumes
  - `next_page_token` is present
  - Second request with page_token returns next 5

---

## 5 · PVC Expansion (Real Kubernetes)

> **Suggested Ginkgo structure:** `Describe("PVC Expansion")`

### TC-I040: Expand volume when StorageClass allows

- **Priority:** High
- **Type:** Integration
- **Given:**
  - PVC exists with capacity `10Gi`
  - PVC uses StorageClass `ceph-rbd` (allowVolumeExpansion: true)
- **When:** PATCH `/api/v1alpha1/volumes/{id}` with `capacity: "20Gi"`
- **Then:**
  - Response: 200 OK
  - PVC `spec.resources.requests.storage` updated to `20Gi`
  - PVC status may show `Resizing` condition

### TC-I041: Reject expansion when StorageClass disallows

- **Priority:** High
- **Type:** Integration
- **Given:**
  - PVC uses StorageClass `nfs` (allowVolumeExpansion: false)
- **When:** PATCH with larger capacity
- **Then:**
  - Response: 400 Bad Request
  - Error message: "StorageClass 'nfs' does not allow volume expansion"

### TC-I042: Reject expansion to smaller size

- **Priority:** High
- **Type:** Integration
- **Given:** PVC with capacity `100Gi`
- **When:** PATCH with `capacity: "50Gi"`
- **Then:**
  - Response: 400 Bad Request
  - Error message: "cannot decrease volume size"

### TC-I043: Reject expansion to same size

- **Priority:** Medium
- **Type:** Integration
- **Given:** PVC with capacity `100Gi`
- **When:** PATCH with `capacity: "100Gi"`
- **Then:**
  - Response: 400 Bad Request
  - Error message: "new size must be larger than current size"

### TC-I044: Sequential expansions work

- **Priority:** High
- **Type:** Integration
- **Given:** PVC with capacity `10Gi` (expandable StorageClass)
- **When:**
  - PATCH to `20Gi`
  - Wait for expansion complete
  - PATCH to `50Gi`
- **Then:** Both expansions succeed

### TC-I045: Expansion of non-existent volume returns 404

- **Priority:** Medium
- **Type:** Integration
- **Given:** No PVC with given ID
- **When:** PATCH volume
- **Then:** Response: 404 Not Found

---

## 6 · PVC Deletion (Real Kubernetes)

> **Suggested Ginkgo structure:** `Describe("PVC Deletion")`

### TC-I050: Delete unattached PVC

- **Priority:** High
- **Type:** Integration
- **Given:** PVC exists and is not attached to any Pod
- **When:** DELETE `/api/v1alpha1/volumes/{id}`
- **Then:**
  - Response: 204 No Content
  - PVC is removed from Kubernetes cluster
  - GET on same ID returns 404

### TC-I051: Delete non-existent volume returns 404

- **Priority:** High
- **Type:** Integration
- **Given:** No PVC with given ID
- **When:** DELETE volume
- **Then:** Response: 404 Not Found

### TC-I052: Delete and verify removal

- **Priority:** High
- **Type:** Integration
- **Given:** PVC exists
- **When:**
  - DELETE volume
  - Wait for deletion to complete
  - GET volume
- **Then:** GET returns 404 Not Found

---

## 7 · Status Reporting (Real NATS)

> **Suggested Ginkgo structure:** `Describe("Status Reporting")`

### TC-I060: CloudEvent published on PVC creation

- **Priority:** High
- **Type:** Integration
- **Given:**
  - NATS server running
  - NATS subscriber listening on `dcm.storage`
  - SP connected to NATS
- **When:** Create PVC
- **Then:**
  - CloudEvent published to `dcm.storage`
  - Event has `type: "dcm.status.storage"`
  - Event data: `{"id": "<uuid>", "status": "PROVISIONING", "message": "..."}`

### TC-I061: CloudEvent published on PVC bind

- **Priority:** High
- **Type:** Integration
- **Given:** 
  - PVC transitions from Pending to Bound
  - Informer is watching
- **When:** PVC binds to PV
- **Then:**
  - CloudEvent published with `status: "RUNNING"`
  - Event data includes `volumeName`

### TC-I062: CloudEvent published on PVC deletion

- **Priority:** High
- **Type:** Integration
- **Given:** PVC exists
- **When:** DELETE PVC
- **Then:**
  - CloudEvent with `status: "DELETING"` published
  - After deletion completes, CloudEvent with `status: "DELETED"` published

### TC-I063: CloudEvent format validation

- **Priority:** High
- **Type:** Integration
- **Given:** Any PVC status change
- **When:** CloudEvent is published
- **Then:** Event has required fields:
  - `id` (event ID, UUID)
  - `source: "dcm/providers/{provider-name}"`
  - `type: "dcm.status.storage"`
  - `subject: "dcm.storage"`
  - `datacontenttype: "application/json"`
  - `specversion: "1.0"`
  - `data: {"id": "...", "status": "...", "message": "..."}`

### TC-I064: Multiple status updates published in sequence

- **Priority:** Medium
- **Type:** Integration
- **Given:** NATS subscriber tracking all events
- **When:** Create PVC → wait for bind → delete PVC
- **Then:** Receive events in order:
  1. PROVISIONING
  2. RUNNING
  3. DELETING
  4. DELETED

---

## 8 · Informer Behavior

> **Suggested Ginkgo structure:** `Describe("Informer Behavior")`

### TC-I070: Informer starts and watches PVCs

- **Priority:** High
- **Type:** Integration
- **Given:** SP starts with informer configured
- **When:** Informer starts
- **Then:**
  - Informer lists existing PVCs with DCM labels
  - Informer watches for changes

### TC-I071: Informer detects new PVC

- **Priority:** High
- **Type:** Integration
- **Given:** Informer is running
- **When:** New PVC with DCM labels is created
- **Then:** Informer Add event fires, status published to NATS

### TC-I072: Informer detects PVC update

- **Priority:** High
- **Type:** Integration
- **Given:** PVC exists and informer is watching
- **When:** PVC status changes (Pending → Bound)
- **Then:** Informer Update event fires, new status published

### TC-I073: Informer detects PVC deletion

- **Priority:** High
- **Type:** Integration
- **Given:** PVC exists
- **When:** PVC is deleted
- **Then:** Informer Delete event fires, DELETED status published

### TC-I074: Informer ignores non-DCM PVCs

- **Priority:** High
- **Type:** Integration
- **Given:** Informer is running
- **When:** PVC without `dcm.project/managed-by=dcm` label is created
- **Then:** Informer does not publish event for this PVC

---

## 9 · Multiple StorageClass Scenarios

> **Suggested Ginkgo structure:** `Describe("Multiple StorageClass Scenarios")`

### TC-I080: Use different StorageClasses

- **Priority:** High
- **Type:** Integration
- **Given:** Multiple StorageClasses exist (ceph-rbd, nfs)
- **When:** Create PVCs with different storageClass hints
- **Then:** Each PVC uses the specified StorageClass

### TC-I081: Expansion behavior differs by backend

- **Priority:** High
- **Type:** Integration
- **Given:**
  - PVC-A uses `ceph-rbd` (allowVolumeExpansion: true)
  - PVC-B uses `nfs` (allowVolumeExpansion: false)
- **When:** Attempt to expand both
- **Then:**
  - PVC-A expansion succeeds
  - PVC-B expansion fails with clear error

### TC-I082: Verify default StorageClass used when not specified

- **Priority:** Medium
- **Type:** Integration
- **Given:** Cluster has default StorageClass configured
- **When:** Create PVC without storageClass hint
- **Then:** PVC uses cluster default StorageClass

---

## 10 · Error Scenarios

> **Suggested Ginkgo structure:** `Describe("Error Scenarios")`

### TC-I090: Handle NATS connection failure gracefully

- **Priority:** High
- **Type:** Integration
- **Given:** NATS server is stopped
- **When:** PVC status changes
- **Then:**
  - SP logs error about NATS connection
  - SP continues operating (CRUD operations still work)
  - Status updates queue or are dropped (no crash)

### TC-I091: Handle Kubernetes API unavailable

- **Priority:** High
- **Type:** Integration
- **Given:** Kubernetes API becomes unreachable
- **When:** API request is made
- **Then:**
  - Appropriate error response (503 Service Unavailable or 500)
  - SP logs connection error
  - Health endpoint reports unhealthy

### TC-I092: Handle invalid kubeconfig

- **Priority:** High
- **Type:** Integration
- **Given:** SP configured with invalid kubeconfig path
- **When:** SP starts
- **Then:**
  - SP fails to start with clear error message
  - Error log indicates kubeconfig problem

---

## Test Execution

### Running Integration Tests

```bash
cd k8s-storage-service-provider

# Start test infrastructure
make integration-test-up  # Starts Kind + NATS

# Run integration tests
make test-integration

# Stop infrastructure
make integration-test-down
```

Or with Ginkgo directly:

```bash
ginkgo -r test/integration --tags=integration
```

### Test Infrastructure Setup

Create `test/integration/setup.sh`:

```bash
#!/bin/bash
set -e

# Create Kind cluster
kind create cluster --name sp-test

# Install test StorageClasses
kubectl apply -f test/fixtures/storageclass-ceph-rbd.yaml
kubectl apply -f test/fixtures/storageclass-nfs.yaml

# Start NATS
docker run -d --name nats-test -p 4222:4222 nats:2-alpine --jetstream

echo "Test infrastructure ready"
```

### Test Fixtures

Example StorageClass fixtures in `test/fixtures/`:

**storageclass-ceph-rbd.yaml:**
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: ceph-rbd
provisioner: rook-ceph.rbd.csi.ceph.com
allowVolumeExpansion: true
parameters:
  pool: replicapool
```

**storageclass-nfs.yaml:**
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: nfs
provisioner: nfs.csi.k8s.io
allowVolumeExpansion: false
parameters:
  server: nfs-server.default.svc
```

---

## Test Duration and Performance

- **Total suite duration:** < 5 minutes (with Kind cluster already running)
- **Individual test timeout:** 30 seconds max
- **PVC creation latency:** < 2 seconds typical
- **Status propagation latency:** < 1 second to NATS

---

## Notes

- Integration tests require Docker for Kind and NATS
- Tests should clean up resources after each run
- Use unique namespaces or resource names to avoid conflicts
- Tests can run in parallel where possible (use different PVC names)
- Monitor resource usage to avoid overwhelming Kind cluster
