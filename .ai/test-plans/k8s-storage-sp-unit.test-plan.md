# Test Plan: K8s Storage SP — Unit Tests

## Overview

- **Related Spec:** .ai/specs/k8s-storage-sp.spec.md
- **Framework:** Ginkgo v2 + Gomega
- **Created:** 2026-07-02
- **Last Updated:** 2026-07-05

Unit tests verify individual components in isolation. All external dependencies
(Kubernetes client, NATS, DCM control-plane) are replaced with mocks or fakes.
Tests use `httptest.NewRecorder` for handler tests and direct function calls for
pure logic.

**Field naming:** SP OpenAPI uses `attachment_mode` (portable) and
`provider_hints.kubernetes.storage_class` / `volume_mode` (snake_case). Catalog
schema may use `provider_hints.kubernetes.access_mode`; placement translation is
out of scope for SP unit tests unless explicitly testing catalog payloads.

**No real external services:**
- ❌ No real Kubernetes cluster
- ❌ No real NATS
- ❌ No real DCM control-plane

**Mocking approach:**
- Mock Kubernetes client: `k8s.io/client-go/kubernetes/fake`
- Mock NATS publisher: custom mock interface
- HTTP handlers: `httptest.NewRecorder`

---

## 1 · Configuration

> **Suggested Ginkgo structure:** `Describe("Configuration")`

### TC-U001: Load configuration from environment variables

- **Priority:** High
- **Type:** Unit
- **Given:** Environment variables are set:
  - `SP_SERVER_ADDRESS=":9090"`
  - `SP_K8S_NAMESPACE="team-a"`
  - `SP_K8S_DEFAULT_STORAGE_CLASS="ceph-rbd"`
  - `SP_K8S_DEFAULT_ATTACHMENT_MODE="multiReadWrite"`
- **When:** Config is loaded
- **Then:** 
  - `server.address = ":9090"`
  - `kubernetes.namespace = "team-a"`
  - `kubernetes.defaultStorageClass = "ceph-rbd"`
  - `kubernetes.defaultAttachmentMode = "multiReadWrite"`

### TC-U002: Default values applied when no config specified

- **Priority:** Medium
- **Type:** Unit
- **Given:** No environment variables are set
- **When:** Config is loaded
- **Then:**
  - `server.address` defaults to `":8080"`
  - `kubernetes.namespace` defaults to `"default"`
  - `kubernetes.defaultAttachmentMode` defaults to `"exclusive"`
  - `kubernetes.defaultStorageClass` is empty (uses cluster default)

### TC-U003: Validate required configuration fields

- **Priority:** High
- **Type:** Unit
- **Given:** Config is missing required field (e.g., kubeconfig path and not in-cluster)
- **When:** Config validation runs
- **Then:** Error is returned indicating missing required field

### TC-U004: Namespace configuration validation

- **Priority:** High
- **Type:** Unit
- **Given:** Namespace is set to invalid value (e.g., contains uppercase, special chars)
- **When:** Config validation runs
- **Then:** Error is returned with invalid namespace message

---

## 2 · PVC Spec Building

> **Suggested Ginkgo structure:** `Describe("PVC Spec Building")`

### TC-U010: Build PVC spec from minimal DCM request

- **Priority:** High
- **Type:** Unit
- **Given:** DCM request with only `capacity`, `metadata.name`, and `service_type: storage`
  ```json
  {
    "service_type": "storage",
    "capacity": "100Gi",
    "metadata": {"name": "test-volume"}
  }
  ```
- **When:** PVC spec is built
- **Then:**
  - PVC `spec.resources.requests.storage = "100Gi"`
  - PVC `metadata.name = "test-volume"`
  - PVC uses SP default storageClass
  - PVC `spec.accessModes = ["ReadWriteOnce"]` (default `attachment_mode: exclusive`)
  - PVC `volumeMode = "Filesystem"` (Kubernetes default when unset)

### TC-U011: Apply storage_class from provider_hints

- **Priority:** High
- **Type:** Unit
- **Given:** DCM request with `provider_hints.kubernetes.storage_class = "ceph-rbd"`
- **When:** PVC spec is built
- **Then:** PVC `spec.storageClassName = "ceph-rbd"`

### TC-U012: Apply attachment_mode on StorageSpec

- **Priority:** High
- **Type:** Unit
- **Given:** DCM request with `attachment_mode: "multiReadWrite"`
- **When:** PVC spec is built
- **Then:** PVC `spec.accessModes = ["ReadWriteMany"]`

### TC-U013: Apply volume_mode from provider_hints

- **Priority:** High
- **Type:** Unit
- **Given:** DCM request with `provider_hints.kubernetes.volume_mode = "Block"`
- **When:** PVC spec is built
- **Then:** PVC `spec.volumeMode = "Block"`

### TC-U014: Apply attachment_mode and provider_hints together

- **Priority:** High
- **Type:** Unit
- **Given:** DCM request with `attachment_mode`, `provider_hints.kubernetes.storage_class`, and `provider_hints.kubernetes.volume_mode` set
- **When:** PVC spec is built
- **Then:** All three settings applied to PVC spec

### TC-U015: Generate DCM labels on PVC

- **Priority:** High
- **Type:** Unit
- **Given:** DCM instance ID `"abc-123"`
- **When:** PVC spec is built
- **Then:** PVC has labels:
  - `dcm.project/managed-by: "dcm"`
  - `dcm.project/dcm-instance-id: "abc-123"`
  - `dcm.project/dcm-service-type: "storage"`

### TC-U016: PVC created in configured namespace

- **Priority:** High
- **Type:** Unit
- **Given:** SP configured with `SP_K8S_NAMESPACE=team-a`
- **When:** PVC spec is built
- **Then:** PVC `metadata.namespace = "team-a"`

---

## 3 · Request Validation

> **Suggested Ginkgo structure:** `Describe("Request Validation")`

### TC-U020: Validate capacity is required

- **Priority:** High
- **Type:** Unit
- **Given:** DCM request without `capacity` field
- **When:** Request is validated
- **Then:** Validation error returned: "capacity is required"

### TC-U021: Validate capacity format

- **Priority:** High
- **Type:** Unit
- **Given:** DCM request with invalid capacity values:
  - `"100"` (no unit)
  - `"invalid"` (not a number)
  - `"100 Gi"` (space in unit)
- **When:** Request is validated
- **Then:** Validation error returned for each case

### TC-U022: Accept valid capacity formats

- **Priority:** High
- **Type:** Unit
- **Given:** DCM request with valid capacity values:
  - `"100Gi"`, `"1Ti"`, `"500GB"`, `"2TB"`, `"100Mi"`
- **When:** Request is validated
- **Then:** All formats pass validation

### TC-U023: Validate metadata.name is required

- **Priority:** High
- **Type:** Unit
- **Given:** DCM request without `metadata.name`
- **When:** Request is validated
- **Then:** Validation error returned: "metadata.name is required"

### TC-U024: Validate metadata.name format (DNS-1123)

- **Priority:** Medium
- **Type:** Unit
- **Given:** DCM request with invalid names:
  - `"Test-Volume"` (uppercase)
  - `"test_volume"` (underscore)
  - `"test.volume"` (dot - invalid for PVC names)
- **When:** Request is validated
- **Then:** Validation error returned for each case

### TC-U025: Validate attachment_mode enum values

- **Priority:** Medium
- **Type:** Unit
- **Given:** DCM request with `attachment_mode: "InvalidMode"`
- **When:** Request is validated
- **Then:** Validation error: attachment_mode must be one of: `exclusive`, `multiReadWrite`, `multiReadOnly`

### TC-U026: Validate volume_mode enum values

- **Priority:** Medium
- **Type:** Unit
- **Given:** DCM request with `provider_hints.kubernetes.volume_mode = "InvalidMode"`
- **When:** Request is validated
- **Then:** Validation error: volume_mode must be `Filesystem` or `Block`

---

## 4 · Status Mapping

> **Suggested Ginkgo structure:** `Describe("Status Mapping")`

### TC-U030: Map PVC Pending to PROVISIONING

- **Priority:** High
- **Type:** Unit
- **Given:** PVC with `status.phase = "Pending"`
- **When:** Status is mapped to DCM status
- **Then:** DCM status = `"PROVISIONING"`

### TC-U031: Map PVC Bound to RUNNING

- **Priority:** High
- **Type:** Unit
- **Given:** PVC with `status.phase = "Bound"`
- **When:** Status is mapped to DCM status
- **Then:** DCM status = `"RUNNING"`

### TC-U032: Map PVC with resize condition to PROVISIONING

- **Priority:** Medium
- **Type:** Unit
- **Given:** PVC with:
  - `status.phase = "Bound"`
  - Condition `type: "Resizing"` with `status: "True"`
- **When:** Status is mapped to DCM status
- **Then:** DCM status = `"PROVISIONING"` (expansion in progress)

### TC-U033: Map PVC Lost to FAILED

- **Priority:** High
- **Type:** Unit
- **Given:** PVC with `status.phase = "Lost"`
- **When:** Status is mapped to DCM status
- **Then:** DCM status = `"FAILED"`

### TC-U034: Map PVC terminating to DELETING

- **Priority:** High
- **Type:** Unit
- **Given:** PVC with `metadata.deletionTimestamp` set
- **When:** Status is mapped to DCM status
- **Then:** DCM status = `"DELETING"`

### TC-U035: Map PVC not found to DELETED

- **Priority:** High
- **Type:** Unit
- **Given:** PVC does not exist in cluster
- **When:** Status is queried
- **Then:** DCM status = `"DELETED"`

### TC-U036: Extract status message from PVC conditions

- **Priority:** Medium
- **Type:** Unit
- **Given:** PVC with condition containing error message
- **When:** Status is mapped
- **Then:** Status message includes condition message

---

## 5 · CloudEvent Construction

> **Suggested Ginkgo structure:** `Describe("CloudEvent Construction")`

### TC-U040: Build CloudEvent with correct attributes

- **Priority:** High
- **Type:** Unit
- **Given:** PVC status change (Pending → Bound)
- **When:** CloudEvent is constructed
- **Then:** CloudEvent has:
  - `id`: non-empty UUID
  - `source`: `"dcm/providers/{provider-name}"`
  - `type`: `"dcm.status.storage"`
  - `subject`: `"dcm.storage"`
  - `datacontenttype`: `"application/json"`
  - `specversion`: `"1.0"`

### TC-U041: Build CloudEvent data payload

- **Priority:** High
- **Type:** Unit
- **Given:** PVC with `dcm-instance-id = "abc-123"` and status `"RUNNING"`
- **When:** CloudEvent is constructed
- **Then:** CloudEvent data contains:
  ```json
  {
    "id": "abc-123",
    "status": "RUNNING",
    "message": "PVC is bound to PV"
  }
  ```

### TC-U042: CloudEvent includes volume details when bound

- **Priority:** Medium
- **Type:** Unit
- **Given:** Bound PVC with `spec.volumeName = "pv-xyz"`
- **When:** CloudEvent is constructed
- **Then:** CloudEvent data includes `volumeName` field

---

## 6 · Volume Expansion Validation

> **Suggested Ginkgo structure:** `Describe("Volume Expansion Validation")`

### TC-U050: Validate expansion size is larger than current

- **Priority:** High
- **Type:** Unit
- **Given:** 
  - Current PVC capacity: `100Gi`
  - Expansion request: `150Gi`
- **When:** Expansion is validated
- **Then:** Validation passes

### TC-U051: Reject expansion to smaller size

- **Priority:** High
- **Type:** Unit
- **Given:**
  - Current PVC capacity: `100Gi`
  - Expansion request: `50Gi`
- **When:** Expansion is validated
- **Then:** Validation error: "cannot decrease volume size"

### TC-U052: Reject expansion to same size

- **Priority:** Medium
- **Type:** Unit
- **Given:**
  - Current PVC capacity: `100Gi`
  - Expansion request: `100Gi`
- **When:** Expansion is validated
- **Then:** Validation error: "new size must be larger than current size"

### TC-U053: Validate StorageClass allows expansion (mocked)

- **Priority:** High
- **Type:** Unit
- **Given:**
  - PVC uses StorageClass with `allowVolumeExpansion = false`
  - Expansion requested
- **When:** Expansion is validated (StorageClass mocked)
- **Then:** Validation error: "StorageClass does not allow volume expansion"

---

## 7 · API Handlers (with Mocked K8s Client)

> **Suggested Ginkgo structure:** `Describe("API Handlers")` with nested contexts

### TC-U060: POST /volumes creates PVC via mocked client

- **Priority:** High
- **Type:** Unit
- **Given:** 
  - Mocked K8s client
  - Valid DCM request
- **When:** POST /volumes handler is called
- **Then:**
  - Mocked K8s client `Create(PVC)` called once
  - Response status: 201 Created
  - Response body contains PVC details

### TC-U061: POST /volumes returns 409 if PVC name exists

- **Priority:** High
- **Type:** Unit
- **Given:**
  - Mocked K8s client configured to return "already exists" error
- **When:** POST /volumes handler is called
- **Then:**
  - Response status: 409 Conflict
  - Error message indicates PVC already exists

### TC-U062: POST /volumes returns 400 for invalid request

- **Priority:** High
- **Type:** Unit
- **Given:** DCM request missing required field (capacity)
- **When:** POST /volumes handler is called
- **Then:**
  - Response status: 400 Bad Request
  - Error message indicates missing capacity

### TC-U063: GET /volumes lists PVCs from mocked client

- **Priority:** High
- **Type:** Unit
- **Given:** Mocked K8s client returns 3 PVCs with DCM labels
- **When:** GET /volumes handler is called
- **Then:**
  - Response status: 200 OK
  - Response contains 3 volumes

### TC-U064: GET /volumes filters by DCM labels

- **Priority:** High
- **Type:** Unit
- **Given:** Mocked K8s client returns mixed PVCs (some with DCM labels, some without)
- **When:** GET /volumes handler is called
- **Then:** Response contains only PVCs with `dcm.project/managed-by=dcm` and `dcm.project/dcm-service-type=storage`

### TC-U065: GET /volumes/{id} returns single PVC

- **Priority:** High
- **Type:** Unit
- **Given:** Mocked K8s client returns PVC with `dcm-instance-id = "abc-123"`
- **When:** GET /volumes/abc-123 handler is called
- **Then:**
  - Response status: 200 OK
  - Response contains PVC details

### TC-U066: GET /volumes/{id} returns 404 if not found

- **Priority:** High
- **Type:** Unit
- **Given:** Mocked K8s client returns "not found" error
- **When:** GET /volumes/nonexistent handler is called
- **Then:**
  - Response status: 404 Not Found

### TC-U067: PATCH /volumes/{id} updates PVC capacity

- **Priority:** High
- **Type:** Unit
- **Given:**
  - Mocked K8s client with PVC (capacity: 100Gi, allowVolumeExpansion: true)
  - Expansion request: 150Gi
- **When:** PATCH /volumes/{id} handler is called
- **Then:**
  - Mocked K8s client `Update(PVC)` called once
  - Response status: 200 OK
  - Response shows new capacity: 150Gi

### TC-U068: PATCH /volumes/{id} rejects when StorageClass disallows

- **Priority:** High
- **Type:** Unit
- **Given:**
  - Mocked K8s client with PVC (StorageClass: allowVolumeExpansion = false)
  - Expansion request
- **When:** PATCH /volumes/{id} handler is called
- **Then:**
  - Response status: 400 Bad Request
  - Error message: "StorageClass does not allow volume expansion"

### TC-U069: DELETE /volumes/{id} deletes PVC

- **Priority:** High
- **Type:** Unit
- **Given:** Mocked K8s client with existing PVC
- **When:** DELETE /volumes/{id} handler is called
- **Then:**
  - Mocked K8s client `Delete(PVC)` called once
  - Response status: 204 No Content

### TC-U070: DELETE /volumes/{id} returns 404 if not found

- **Priority:** High
- **Type:** Unit
- **Given:** Mocked K8s client returns "not found" error
- **When:** DELETE /volumes/nonexistent handler is called
- **Then:**
  - Response status: 404 Not Found

---

## 8 · Health Endpoint

> **Suggested Ginkgo structure:** `Describe("Health Endpoint")`

### TC-U080: GET /api/v1alpha1/health returns healthy status

- **Priority:** High
- **Type:** Unit
- **Given:** SP is running with mocked K8s client (healthy)
- **When:** GET `/api/v1alpha1/health` handler is called
- **Then:**
  - Response status: 200 OK
  - Response body:
    ```json
    {
      "status": "healthy",
      "type": "k8s-storage-service-provider.dcm.io/health",
      "path": "health",
      "version": "<version>",
      "uptime": <seconds>
    }
    ```

### TC-U081: GET /api/v1alpha1/health returns unhealthy when K8s unreachable

- **Priority:** High
- **Type:** Unit
- **Given:** Mocked K8s client health check returns error
- **When:** GET `/api/v1alpha1/health` handler is called
- **Then:**
  - Response status: 200 OK (per DCM convention)
  - Response body: `{"status": "unhealthy"}`

---

## 9 · Error Handling

> **Suggested Ginkgo structure:** `Describe("Error Handling")`

### TC-U090: Handle K8s API errors gracefully

- **Priority:** High
- **Type:** Unit
- **Given:** Mocked K8s client returns various errors (timeout, unauthorized, internal)
- **When:** Any handler is called
- **Then:** Appropriate HTTP error code and message returned

### TC-U091: Handle invalid JSON in request body

- **Priority:** Medium
- **Type:** Unit
- **Given:** POST request with malformed JSON
- **When:** Handler is called
- **Then:**
  - Response status: 400 Bad Request
  - Error message indicates JSON parsing error

### TC-U092: Handle missing required headers

- **Priority:** Low
- **Type:** Unit
- **Given:** Request without Content-Type header
- **When:** POST handler is called
- **Then:** Request handled or appropriate error returned

---

## Test Execution

### Running Unit Tests

```bash
cd k8s-storage-service-provider
make test-unit
```

Or directly with Ginkgo:

```bash
ginkgo -r --skip-package=test/integration
```

### Coverage Target

- **Minimum:** 80% code coverage
- **Target:** 90% code coverage
- **Focus areas:** Request validation, status mapping, PVC spec building

### Mocking Libraries

- **Kubernetes client:** `k8s.io/client-go/kubernetes/fake`
- **NATS publisher:** Custom interface mock
- **HTTP testing:** `net/http/httptest`

---

## Notes

- All unit tests should execute in < 1 second total
- No network calls to real services
- No file I/O except for config loading tests
- Tests should be deterministic and repeatable
- Use table-driven tests where applicable (e.g., capacity format validation)
