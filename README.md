<h1 align="center">LightBoot</h1>
<p align="center"><strong>Zero‑dependency PXE network boot manager</strong></p>
<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go version">
  <img src="https://img.shields.io/badge/license-GPLv2-blue.svg" alt="License">
  <img src="https://img.shields.io/badge/docs-mkdocs-success.svg" alt="Docs">
</p>

---

**LightBoot** is a single binary that turns any Linux machine into a  
PXE boot server.  Drop ISO files in a folder and it automatically  
extracts kernels, builds iPXE menus, and boots BIOS / UEFI / ARM64  
clients – all without touching your existing DHCP server.

## How It Works

```
Client           Proxy DHCP        TFTP           HTTP            Cache
  │                  │               │              │               │
  ├─DHCPDISCOVER─────►              │              │               │
  ├─DHCPOFFER────────┤ bootfile URL │              │               │
  ├──────────TFTP GET───────────────► iPXE fw      │               │
  ├──────────HTTP GET /boot/ipxe────────────────────►   menu        │
  ├──────────HTTP GET kernel+initrd───────────────────────────────►  boot files
```

- **Proxy DHCP** – answers PXE clients without conflicting with your network's DHCP  
- **TFTP** – serves the right iPXE firmware for each client architecture  
- **HTTP Boot** – serves auto‑generated iPXE scripts and extracted kernel/initrd  
- **Auto‑Detection** – built‑in profiles recognise Ubuntu, Debian, Fedora, Arch, Alpine and more  

## Features

|                   |                                                                                                      |
|-------------------|------------------------------------------------------------------------------------------------------|
| 📦 **Single Binary** | No runtime, no dependencies – download and run                                                    |
| 🔍 **Auto Detection** | Profiles for 6+ Linux distributions, auto‑extracts and serves ISO content                        |
| 🌐 **Web UI**         | Vue.js SPA to upload ISOs, view logs and preview boot menus (SSE live logs)                      |
| 🔐 **API Token**      | Secure Bearer‑token authentication for the management API                                        |
| 🗄️ **ISO Caching**    | Extracts kernels and initrds once, serves them fast                                              |
| ⚡ **Real‑time**       | Server‑Sent Events for live log streaming                                                        |
| 🛠️ **Docker**          | Ready‑to‑use Dockerfile and docker‑compose for containerised deployment                         |

## Quick Start

### Binary

```bash
# Download from the releases page (or build yourself)
./lightboot

# The API token is printed on first start – save it!
# Open http://localhost:8080 and log in
# Drop ISO files into the iso/ directory
```

### Docker

```bash
docker compose up -d --build    # build & start
docker compose logs -f          # follow logs
docker compose down             # stop
```

## Development

```bash
# Frontend
make web                        # builds Vue.js frontend into web/dist/

# Go binary
make build                      # compiles to bin/lightboot

# Run with rebuild
make run                        # build + ./bin/lightboot

# Documentation
make docs                       # builds MkDocs site into site/

# Docker
make docker                     # build Docker image
```

### Prerequisites

- **Go** ≥ 1.25
- **Node.js** (for the Vue.js frontend)
- **MkDocs** with Material theme (optional, for docs)

## Documentation

Full documentation is available at **[the docs site](https://shniranjan.github.io/lightboot)** (or build it locally with `make docs`).

Browse the [docs/](docs/) directory for:
- Installation & configuration
- ISO management & boot modes
- Networking & architecture
- API reference
- Secure Boot setup
- Troubleshooting guide

## Contributing

Issues, pull requests and new distribution profiles are very welcome!  
Read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for community standards.

## License

LightBoot is licensed under the **GNU General Public License v2.0** – see [LICENSE](LICENSE) for the full text.

---

<p align="center"><sub>Made with ❤️ for the net‑boot community</sub></p>