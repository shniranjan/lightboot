# Installation

LightBoot ships as a single static binary. No runtime dependencies, no package manager required.

## Supported Platforms

| Platform | Architecture | Status |
|----------|-------------|--------|
| Linux | amd64 (x86_64) | ✅ Primary |
| Linux | arm64 (aarch64) | ✅ Supported |
| Linux | armv7 | ✅ Supported |
| macOS | amd64 / arm64 | ⚠️ Testing (no DHCP proxy) |
| Windows | amd64 | ❌ Not supported |

> **Note**: The DHCP proxy requires raw socket access (Linux `SO_BROADCAST` + `CAP_NET_BIND_SERVICE`). On macOS and Windows, only HTTP + TFTP services are functional.

---

## Download

### Pre-built Binaries

Download the latest release from [GitHub Releases](https://github.com/shniranjan/lightboot/releases):

- `lightboot-linux-amd64` — 64-bit Intel/AMD
- `lightboot-linux-arm64` — 64-bit ARM (Raspberry Pi 4/5, AWS Graviton)
- `lightboot-linux-armv7` — 32-bit ARM (Raspberry Pi 2/3)

### Quick Install from GitHub Releases

Download the latest binary for your architecture directly from GitHub:

```bash
# On x86_64 (amd64):
curl -L -o lightboot https://github.com/shniranjan/lightboot/releases/latest/download/lightboot-linux-amd64

# On ARM64 (aarch64):
curl -L -o lightboot https://github.com/shniranjan/lightboot/releases/latest/download/lightboot-linux-arm64

# On ARMv7:
curl -L -o lightboot https://github.com/shniranjan/lightboot/releases/latest/download/lightboot-linux-armv7

chmod +x lightboot
sudo setcap cap_net_bind_service=+ep ./lightboot
mkdir -p iso cache data profiles bootfiles
```

All downloads come directly from [GitHub Releases](https://github.com/shniranjan/lightboot/releases). No external sites involved.

---

## Linux Setup

### 1. Download and Make Executable

```bash
curl -L -o lightboot https://github.com/shniranjan/lightboot/releases/latest/download/lightboot-linux-amd64
chmod +x lightboot
```

### 2. Grant Network Capabilities (for DHCP Proxy)

The DHCP proxy binds to UDP port 67, which requires elevated privileges. Instead of running as root, grant the specific capability:

```bash
sudo setcap cap_net_bind_service=+ep ./lightboot
```

### 3. Create Directories

Create the directories LightBoot will use:

```bash
mkdir -p iso cache data profiles bootfiles
```

If you downloaded iPXE firmware files, place them in `bootfiles/`:

- `undionly.kpxe` — BIOS firmware
- `ipxe.efi` — UEFI x64 firmware
- `snponly.efi` — UEFI SNP driver

### 4. First Run

```bash
./lightboot
```

On first run, LightBoot:
1. Generates a random 64-character hex API token
2. Saves it to `data/.api_token`
3. Prints the token to the console — **save this!**
4. Starts all services (HTTP on :8080, TFTP on :69, DHCP proxy on :67)

```
LightBoot v0.1.0 — PXE Network Boot Manager

Generated new API token: a1b2c3d4e5f6...
Token saved to: data/.api_token

Database: data/lightboot.db
Profiles loaded: 6
HTTP server listening on http://0.0.0.0:8080
API token: a1b2c3d4e5f6...
Server started. Press Ctrl+C to stop.
```

---

## Systemd Service

Create `/etc/systemd/system/lightboot.service`:

```ini
[Unit]
Description=LightBoot PXE Boot Manager
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=lightboot
Group=lightboot
WorkingDirectory=/opt/lightboot
ExecStart=/usr/local/bin/lightboot
Restart=on-failure
RestartSec=5

# Allow binding to privileged ports
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

# Security hardening
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/opt/lightboot

[Install]
WantedBy=multi-user.target
```

Then enable and start:

```bash
sudo useradd -r -s /bin/false lightboot
sudo mkdir -p /opt/lightboot/{iso,cache,data,profiles,bootfiles}
sudo cp lightboot /usr/local/bin/
sudo cp config.yaml /opt/lightboot/
sudo cp -r bootfiles/* /opt/lightboot/bootfiles/
sudo chown -R lightboot:lightboot /opt/lightboot

sudo systemctl daemon-reload
sudo systemctl enable lightboot
sudo systemctl start lightboot
```

Check status:

```bash
sudo systemctl status lightboot
sudo journalctl -u lightboot -f
```

---

## Docker

### Docker Run

> **Note**: The DHCP proxy requires `--network host` or `--cap-add NET_BIND_SERVICE`.

```bash
docker run -d \
  --name lightboot \
  --network host \
  --cap-add NET_BIND_SERVICE \
  -v $(pwd)/data:/data \
  -v $(pwd)/iso:/iso \
  -v $(pwd)/cache:/cache \
  ghcr.io/shniranjan/lightboot:latest
```

### Docker Compose

```yaml
version: "3.8"
services:
  lightboot:
    image: ghcr.io/shniranjan/lightboot:latest
    container_name: lightboot
    network_mode: host
    cap_add:
      - NET_BIND_SERVICE
    volumes:
      - ./data:/data
      - ./iso:/iso
      - ./cache:/cache
      - ./profiles:/profiles
    restart: unless-stopped
```

```bash
docker compose up -d
```

---

## Build from Source

### Prerequisites

- Go 1.22+
- Node.js 18+ (for Web UI)
- Python 3 + venv (for documentation)

### Build

```bash
git clone https://github.com/shniranjan/lightboot.git
cd lightboot

# Build the Web UI
make web

# Build the documentation
make docs

# Build the binary
make build

# Run
./bin/lightboot
```

---

## Verifying the Installation

### 1. Check the Web UI

Open `http://localhost:8080` in your browser. You should see the login page.

### 2. Verify API Health

```bash
curl http://localhost:8080/api/health
# {"status":"ok"}
```

### 3. Check Services

```bash
# TFTP (should respond to a read request)
echo -ne "\x00\x01undionly.kpxe\x00octet\x00" | nc -u -w1 localhost 69 | xxd | head

# Boot menu (public endpoint)
curl http://localhost:8080/api/boot/ipxe
```

### 4. Upload an ISO

```bash
curl -X POST http://localhost:8080/api/isos/upload \
  -H "Authorization: Bearer YOUR_API_TOKEN" \
  -F "file=@/path/to/ubuntu-24.04-server-amd64.iso"
```

---

## Upgrading

1. Download the new binary
2. Stop the running instance
3. Replace the binary
4. Start again — the database and config are backward-compatible

```bash
sudo systemctl stop lightboot
sudo cp lightboot /usr/local/bin/
sudo systemctl start lightboot
```

---

## Uninstalling

```bash
sudo systemctl stop lightboot
sudo systemctl disable lightboot
sudo rm /usr/local/bin/lightboot
sudo rm /etc/systemd/system/lightboot.service
sudo rm -rf /opt/lightboot
sudo userdel lightboot
```

> ⚠️ This deletes your ISO cache, database, and configuration. Back up anything you need first.
