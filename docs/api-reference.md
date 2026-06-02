# API Reference

LightBoot exposes a REST API for programmatic management. All authenticated endpoints require a Bearer token.

---

## Authentication

All management endpoints require an `Authorization` header:

```http
Authorization: Bearer <your-api-token>
```

The token is:
- Printed to console on first run
- Stored in `data/.api_token`
- Overridable via `LIGHTBOOT_API_TOKEN` environment variable

Requests without a valid token receive:

```json
{"error": "missing authorization header"}
```
— HTTP 401

---

## Public Endpoints (No Auth)

### Health Check

```http
GET /api/health
```

**Response** `200 OK`
```json
{"status": "ok"}
```

---

### iPXE Boot Script

```http
GET /api/boot/ipxe
```

**Response** `200 OK` — `text/plain`
```
#!ipxe
set menu-timeout 15000
menu LightBoot — Select an OS
item ubuntu Ubuntu 24.04.1 Server (amd64)
...
```

Returns the full iPXE menu script with `kernel` and `initrd` directives for all ready ISOs.

---

### Chainload Endpoint

```http
GET /api/boot/chain
```

**Response** `200 OK` — `text/plain`

Returns an iPXE script that chains to the full boot menu. Useful for integrating with existing PXE servers.

---

### Boot Menu (JSON)

```http
GET /api/boot/menu
```

**Response** `200 OK`
```json
{
  "items": [
    {
      "label": "Ubuntu 24.04.1 Server (amd64)",
      "kernel_url": "http://192.168.1.10:8080/cache/1/vmlinuz",
      "initrd_url": "http://192.168.1.10:8080/cache/1/initrd.gz",
      "cmdline": "root=/dev/ram0 ramdisk_size=1500000 ip=dhcp"
    }
  ]
}
```

---

### Log Stream (SSE)

```http
GET /api/logs/stream
```

**Response** `200 OK` — `text/event-stream`

Server-Sent Events stream. Each event is a JSON log entry:

```
data: {"ts":"2024-01-15T10:30:00Z","level":"info","source":"scanner","msg":"Scanning iso/ directory"}

data: {"ts":"2024-01-15T10:30:01Z","level":"info","source":"scanner","msg":"Found: ubuntu-24.04-server-amd64.iso (2.5 GB)"}
```

> **Note**: The browser `EventSource` API cannot send custom headers. This endpoint is public. Use it for real-time log monitoring in the Web UI.

---

## Authenticated Endpoints (Bearer Token Required)

---

### List ISOs

```http
GET /api/isos
Authorization: Bearer <token>
```

**Response** `200 OK`
```json
[
  {
    "id": 1,
    "name": "ubuntu-24.04.1-live-server-amd64.iso",
    "size": 2701131776,
    "status": "ready",
    "distro": "Ubuntu",
    "version": "24.04.1",
    "architecture": "amd64",
    "sha256": "a1b2c3d4e5f6..."
  }
]
```

| Field | Type | Description |
|-------|------|-------------|
| `id` | int64 | Unique database ID |
| `name` | string | Original filename |
| `size` | int64 | File size in bytes |
| `status` | string | `pending`, `processing`, `ready`, `error`, or `unknown` |
| `distro` | string | Detected distribution name |
| `version` | string | Distribution version |
| `architecture` | string | CPU architecture (amd64, arm64) |
| `sha256` | string | SHA256 hex digest |

---

### Upload ISO

```http
POST /api/isos/upload
Authorization: Bearer <token>
Content-Type: multipart/form-data
```

**Form Field**: `file` — the ISO file

**Response** `200 OK`
```json
{
  "status": "uploaded",
  "name": "ubuntu-24.04-server-amd64.iso",
  "size": 2701131776,
  "path": "./iso/ubuntu-24.04-server-amd64.iso"
}
```

**Limits**:
- Max upload size: configurable via `max_upload_size` (default 20 GB)
- Only `.iso` extension accepted
- Returns `413 Request Entity Too Large` if the file exceeds the limit
- Returns `400 Bad Request` if no file field or non-`.iso` extension

**Example with curl**:
```bash
curl -X POST http://localhost:8080/api/isos/upload \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@/path/to/image.iso"
```

**Example with progress (JavaScript)**:
```javascript
const formData = new FormData()
formData.append('file', file)

const xhr = new XMLHttpRequest()
xhr.upload.onprogress = (e) => {
  if (e.lengthComputable) {
    console.log(`${Math.round(e.loaded / e.total * 100)}%`)
  }
}
xhr.open('POST', '/api/isos/upload')
xhr.setRequestHeader('Authorization', `Bearer ${token}`)
xhr.send(formData)
```

---

### Delete ISO

```http
DELETE /api/isos/{id}
Authorization: Bearer <token>
```

**Path Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| `id` | int64 | ISO database ID (from list response) |

**Response** `200 OK`
```json
{"status": "deleted"}
```

**Errors**:
- `400 Bad Request` — invalid or missing ID
- `500 Internal Server Error` — database error

---

### Trigger Scan

```http
POST /api/scan
Authorization: Bearer <token>
```

**Response** `200 OK`
```json
{"status": "scan triggered"}
```

Forces an immediate rescan of the ISO directory, bypassing the `scanner_interval` timer.

---

### Get Configuration

```http
GET /api/config
Authorization: Bearer <token>
```

**Response** `200 OK`
```json
{
  "version": "0.1.0",
  "http_listen": "0.0.0.0:8080",
  "tftp_listen": "0.0.0.0:69",
  "dhcp_listen": "0.0.0.0:67",
  "iso_dir": "./iso",
  "cache_dir": "./cache",
  "data_dir": "./data",
  "bootfiles_dir": "./bootfiles",
  "dhcp_proxy_enabled": true,
  "max_upload_size": 21474836480,
  "scanner_interval": 300,
  "log_level": "info"
}
```

> The API token is **never** returned in this response. Use the regenerate endpoint to get a new one.

---

### Regenerate API Token

```http
POST /api/config/regenerate-token
Authorization: Bearer <token>
```

**Response** `200 OK`
```json
{"token": "new-64-char-hex-token..."}
```

Generates a new random token, overwrites the file at `api_token_path`, and returns it. **The old token is immediately invalidated.** Save the new token.

---

### Recent Logs

```http
GET /api/logs/recent
Authorization: Bearer <token>
```

**Response** `200 OK`
```json
[
  {
    "ts": "2024-01-15T10:30:00Z",
    "level": "info",
    "source": "scanner",
    "msg": "Scanning iso/ directory"
  },
  {
    "ts": "2024-01-15T10:30:01Z",
    "level": "warn",
    "source": "detector",
    "msg": "No profile matched for unknown-image.iso"
  }
]
```

Returns the 50 most recent log entries from the ring buffer.

---

## Error Response Format

All errors follow this format:

```json
{"error": "human-readable error message"}
```

| Status Code | Meaning |
|-------------|--------|
| `200 OK` | Success |
| `400 Bad Request` | Invalid input (missing file, wrong extension, invalid ID) |
| `401 Unauthorized` | Missing or invalid Bearer token |
| `404 Not Found` | Unknown route |
| `413 Request Entity Too Large` | Upload exceeds `max_upload_size` |
| `500 Internal Server Error` | Database error, disk I/O failure, etc. |

---

## REST API Summary

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/health` | No | Health check |
| `GET` | `/api/boot/ipxe` | No | iPXE menu script |
| `GET` | `/api/boot/chain` | No | Chainload redirect script |
| `GET` | `/api/boot/menu` | No | Boot menu as JSON |
| `GET` | `/api/logs/stream` | No | SSE log stream |
| `GET` | `/api/isos` | Yes | List all ISOs |
| `POST` | `/api/isos/upload` | Yes | Upload ISO file |
| `DELETE` | `/api/isos/{id}` | Yes | Delete an ISO |
| `POST` | `/api/scan` | Yes | Trigger ISO scan |
| `GET` | `/api/config` | Yes | Get configuration |
| `POST` | `/api/config/regenerate-token` | Yes | Regenerate API token |
| `GET` | `/api/logs/recent` | Yes | Recent log entries |
| `GET` | `/cache/*` | No | Static cached boot files |
| `GET` | `/docs/*` | No | Documentation site |
| `GET` | `/*` | No | SPA Web UI (fallback) |
