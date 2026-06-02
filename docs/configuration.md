# Configuration

LightBoot is configured via a `config.yaml` file and environment variables. All settings have sensible defaults — you can start with an empty config file.

---

## Config File Location

LightBoot looks for `config.yaml` in the current working directory. If not found, all defaults are used.

### Generate a Default Config

```bash
# Create a config file with every option and its default
cat > config.yaml << 'EOF'
http_port: 8080
http_address: "0.0.0.0"
tftp_port: 69
tftp_address: "0.0.0.0"
dhcp_proxy_enabled: true
dhcp_proxy_port: 67
dhcp_proxy_address: "0.0.0.0"
iso_dir: "./iso"
cache_dir: "./cache"
data_dir: "./data"
profiles_dir: "./profiles"
bootfiles_dir: "./bootfiles"
api_token_path: "./data/.api_token"
max_upload_size: 21474836480
scanner_interval: 300
log_level: "info"
log_file: ""
EOF
```

---

## All Configuration Options

### HTTP Server

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `http_port` | int | `8080` | Port for the web UI and API |
| `http_address` | string | `0.0.0.0` | Bind address (`0.0.0.0` = all interfaces) |

**Environment variable overrides**:
- `LIGHTBOOT_HTTP_PORT=9090`
- `LIGHTBOOT_HTTP_ADDRESS=127.0.0.1`

### TFTP Server

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `tftp_port` | int | `69` | Port for TFTP file transfers |
| `tftp_address` | string | `0.0.0.0` | Bind address for TFTP |

**Environment variable override**:
- `LIGHTBOOT_TFTP_PORT=6969`

### DHCP Proxy

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `dhcp_proxy_enabled` | bool | `true` | Whether to start the proxy DHCP server |
| `dhcp_proxy_port` | int | `67` | UDP port for DHCP (requires privileges) |
| `dhcp_proxy_address` | string | `0.0.0.0` | Bind address for DHCP |

> **Important**: Port 67 requires `CAP_NET_BIND_SERVICE` or root. See [Installation](installation.md).

### Directory Paths

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `iso_dir` | string | `./iso` | Directory to watch for ISO files |
| `cache_dir` | string | `./cache` | Directory for extracted kernels and initrds |
| `data_dir` | string | `./data` | Directory for database and API token |
| `profiles_dir` | string | `./profiles` | Directory for custom YAML profiles |
| `bootfiles_dir` | string | `./bootfiles` | Directory for iPXE firmware files |

**Environment variable overrides**:
- `LIGHTBOOT_ISO_DIR=/srv/isos`
- `LIGHTBOOT_CACHE_DIR=/var/cache/lightboot`
- `LIGHTBOOT_DATA_DIR=/var/lib/lightboot`

### API Token

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `api_token_path` | string | `./data/.api_token` | File path for the API token |

**Environment variable override** (high priority):
- `LIGHTBOOT_API_TOKEN=your-token-here`

When `LIGHTBOOT_API_TOKEN` is set in the environment, it takes precedence over the file. This is useful for automated deployments where you want to supply a known token.

### Upload Limits

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `max_upload_size` | int64 | `21474836480` | Maximum ISO upload size in bytes (20 GB) |

### Scanner

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `scanner_interval` | int | `300` | Interval in seconds between ISO directory scans |

### Logging

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `log_level` | string | `info` | Log level: `debug`, `info`, `warn`, or `error` |
| `log_file` | string | `""` | Optional path for file-based logging |

**Environment variable overrides**:
- `LIGHTBOOT_LOG_LEVEL=debug`
- `LIGHTBOOT_LOG_FILE=/var/log/lightboot.log`

---

## Environment Variables Reference

All environment variable overrides use the `LIGHTBOOT_` prefix:

```bash
# Full list of env var overrides
LIGHTBOOT_HTTP_PORT=8080
LIGHTBOOT_HTTP_ADDRESS=0.0.0.0
LIGHTBOOT_TFTP_PORT=69
LIGHTBOOT_ISO_DIR=/path/to/isos
LIGHTBOOT_CACHE_DIR=/path/to/cache
LIGHTBOOT_DATA_DIR=/path/to/data
LIGHTBOOT_LOG_LEVEL=debug
LIGHTBOOT_LOG_FILE=/path/to/logfile.log
LIGHTBOOT_API_TOKEN=my-secret-token
```

**Precedence order** (highest to lowest):
1. Environment variable `LIGHTBOOT_API_TOKEN` (for the token only)
2. Environment variables for all other settings
3. `config.yaml` values
4. Built-in defaults

---

## Configuration Examples

### Minimal Setup (all defaults)

No config file needed. Just run `./lightboot`.

### Production Server

```yaml
# config.yaml — production deployment
http_port: 80
http_address: "192.168.1.10"
tftp_port: 69
tftp_address: "192.168.1.10"
dhcp_proxy_enabled: true
dhcp_proxy_port: 67
dhcp_proxy_address: "192.168.1.10"
iso_dir: "/srv/lightboot/isos"
cache_dir: "/var/cache/lightboot"
data_dir: "/var/lib/lightboot"
profiles_dir: "/etc/lightboot/profiles"
bootfiles_dir: "/usr/share/lightboot/bootfiles"
api_token_path: "/var/lib/lightboot/.api_token"
max_upload_size: 53687091200  # 50 GB
scanner_interval: 120         # 2 minutes
log_level: "info"
log_file: "/var/log/lightboot/lightboot.log"
```

### Development Setup

```yaml
# config.yaml — development with all debug options
http_port: 8080
http_address: "127.0.0.1"
log_level: "debug"
dhcp_proxy_enabled: false  # Disable DHCP for local dev
```

### Docker Environment

Use environment variables with Docker:

```bash
docker run -d \
  --name lightboot \
  --network host \
  --cap-add NET_BIND_SERVICE \
  -e LIGHTBOOT_API_TOKEN=my-secure-token \
  -e LIGHTBOOT_LOG_LEVEL=info \
  -v /srv/lightboot/data:/data \
  -v /srv/lightboot/isos:/iso \
  ghcr.io/shniranjan/lightboot:latest
```

---

## API Token Management

### View the Current Token

```bash
cat data/.api_token
```

### Regenerate via API

```bash
curl -X POST http://localhost:8080/api/config/regenerate-token \
  -H "Authorization: Bearer CURRENT_TOKEN"
```

This returns a new token and overwrites the file. **Save the new token immediately** — the old one is invalidated.

### Regenerate via File

```bash
# Stop LightBoot, delete the token, restart
sudo systemctl stop lightboot
rm data/.api_token
sudo systemctl start lightboot
# New token is printed to journal
sudo journalctl -u lightboot -n 5
```

---

## Database Location

LightBoot stores its SQLite database at `<data_dir>/lightboot.db`. This file contains:

- ISO metadata (name, size, SHA256, distro, version, boot profile)
- Processing status (pending, processing, ready, error)
- Cached file paths

The database is automatically created on first run and migrated on version upgrades. You can safely back up this file while LightBoot is running (SQLite WAL mode).
