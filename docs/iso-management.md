# ISO Management

LightBoot automatically detects, analyzes, and caches ISO files. This page covers the full lifecycle — from upload to bootable menu entry.

---

## Adding ISOs

There are two ways to add ISOs:

### 1. Drag-and-Drop (Web UI)

1. Log into the Web UI at `http://localhost:8080`
2. Navigate to **ISO Library**
3. Drag an ISO file onto the drop zone, or click to browse
4. A progress bar shows upload status
5. Once uploaded, LightBoot automatically scans and processes the ISO

> **Size limit**: Default 20 GB, configurable via `max_upload_size` in [Configuration](configuration.md).

### 2. File Copy

Simply copy ISO files into the `iso_dir` directory (default: `./iso`). The scanner detects new files within the interval set by `scanner_interval` (default: 5 minutes).

```bash
cp ~/Downloads/ubuntu-24.04-server-amd64.iso ./iso/
```

To trigger an immediate scan, use the **Rescan** button in the Web UI Dashboard, or call the API:

```bash
curl -X POST http://localhost:8080/api/scan \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## Detection Process

When a new ISO is detected, LightBoot performs these steps:

1. **Calculate SHA256** — computes the file hash for deduplication
2. **Check database** — if an ISO with the same SHA256 already exists, skip it
3. **Open ISO** — reads the ISO9660 filesystem using the internal Go reader (falls back to `xorriso` if configured)
4. **Run profiles** — tests the ISO against each [detection profile](profiles.md) in order
5. **First match wins** — the first profile whose `files` and `contents` rules match is used
6. **Extract boot files** — copies kernel, initrd, and other required files to the cache
7. **Update database** — sets status to `ready` and stores the boot configuration

### ISO Statuses

| Status | Description |
|--------|-------------|
| `pending` | File detected, awaiting processing |
| `processing` | Currently being analyzed or extracted |
| `ready` | Successfully detected and cached — appears in boot menu |
| `error` | Detection or extraction failed — check logs |
| `unknown` | No profile matched — stored in DB but not bootable |

---

## Viewing ISOs

### Web UI

The **ISO Library** page shows a table with:

| Column | Description |
|--------|-------------|
| Name | Original filename |
| Distro | Detected distribution (Ubuntu, Debian, etc.) |
| Version | Distribution version (24.04, 12, etc.) |
| Size | File size in human-readable format |
| Architecture | amd64, arm64, etc. |
| Status | Ready / Processing / Error / Unknown |
| Actions | Delete button |

You can filter the list using the search bar (searches name, distro, version).

### API

```bash
curl http://localhost:8080/api/isos \
  -H "Authorization: Bearer YOUR_TOKEN"
```

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
    "sha256": "abc123..."
  }
]
```

---

## Deleting ISOs

### Web UI

Click the **Delete** button (trash icon) next to an ISO. A confirmation dialog appears. Deleting an ISO also removes its cached kernel and initrd files.

### API

```bash
curl -X DELETE http://localhost:8080/api/isos/1 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

---

## Re-extracting an ISO

If you update a profile or suspect corruption, you can force re-extraction:

```bash
# Delete the ISO and re-upload (or re-copy)
curl -X DELETE http://localhost:8080/api/isos/1 \
  -H "Authorization: Bearer YOUR_TOKEN"

# Then copy the ISO again or trigger a rescan
```

Currently there is no explicit "re-extract" button — this will be added in a future release.

---

## Cache Directory Structure

Extracted files are stored in `cache/<iso-id>/`:

```
cache/
└── 1/
    ├── vmlinuz          # Linux kernel
    ├── initrd.gz         # Initial ramdisk
    └── ...               # Other profile-specific files
```

The cache is managed automatically:
- Created during ISO processing
- Deleted when the ISO is removed
- You can safely clear the entire cache directory (it will be rebuilt on demand)

---

## Supported Distributions

LightBoot ships with built-in profiles for these distributions:

| Distribution | Versions | Architecture |
|-------------|---------|-------------|
| Ubuntu | 24.04+ (Desktop & Server) | amd64 |
| Debian | 12+ | amd64 |
| Fedora | 40+ | amd64 |
| Arch Linux | Rolling | amd64 |
| Alpine Linux | 3.20+ | amd64 |

Additional profiles can be added as [custom profiles](profiles.md#custom-profiles).

---

## Troubleshooting ISO Detection

### ISO Stays "Unknown"

- The ISO format may not be recognized. Check that it's a valid ISO9660 filesystem.
- The distribution may not have a matching profile. See [Profiles](profiles.md) to write one.
- Check the logs: `curl http://localhost:8080/api/logs/recent -H "Authorization: Bearer TOKEN"`

### ISO Shows "Error" Status

- Extraction may have failed. Common causes:
  - Disk full — check `cache_dir` has enough space
  - Corrupt ISO — verify the SHA256 checksum
  - Missing kernel/initrd — the profile may reference files not in the ISO
- Check logs for details on the failure

### Upload Fails

- File exceeds `max_upload_size` — increase the limit in config
- File is not an `.iso` — only `.iso` extension is accepted
- Disk full — check `iso_dir` has enough space

### ISO Fails to Boot

- Verify the kernel and initrd were extracted correctly: check `cache/<iso-id>/`
- Check the boot menu syntax: visit the **Boot Menu** page in the Web UI
- Try a different PXE client or check the client's architecture (BIOS vs UEFI)
