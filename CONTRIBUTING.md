# Contributing to LightBoot

Thanks for your interest in contributing! LightBoot is a PXE/iPXE network boot
manager that turns any Linux machine into a netboot appliance. We welcome
contributions of all kinds: bug reports, feature requests, documentation
improvements, new ISO profiles, and code changes.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Project Structure](#project-structure)
- [Adding New ISO Profiles](#adding-new-iso-profiles)
- [Pull Request Process](#pull-request-process)
- [Style Guide](#style-guide)
- [Reporting Bugs](#reporting-bugs)
- [Feature Requests](#feature-requests)

## Code of Conduct

This project adheres to the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md).
By participating, you are expected to uphold this code.

## Getting Started

### Prerequisites

- **Go 1.24+** (for the backend server)
- **Node.js 20+** (for the Vue.js frontend)
- **xorriso** (optional, for ISO extraction fallback)
- **Docker** (optional, for containerized builds)

### Quick Start

```bash
git clone https://github.com/shniranjan/lightboot.git
cd lightboot

# Build everything from source
make web      # Build Vue.js frontend
make build    # Build Go binary

# Run LightBoot
./bin/lightboot
```

Open `http://localhost:8080` in your browser.

### Using Docker

```bash
make docker-up    # Build and start with docker-compose
make docker-logs  # Follow logs
make docker-down  # Stop all services
```

## Development Workflow

1. **Fork** the repository and create a feature branch off `main`.
2. **Make changes** following the style guide below.
3. **Build and test** your changes:
   ```bash
   make web    # Build frontend
   make build  # Compile Go binary
   ./bin/lightboot  # Run locally
   ```
4. **Submit a pull request** with a clear description of what changed and why.

### Running Tests

```bash
go test ./...          # Run all tests
go test -v -race ./... # Verbose with race detector
```

### Development Mode (hot-reload frontend)

```bash
# Terminal 1: Start the backend
./bin/lightboot

# Terminal 2: Start Vite dev server with API proxy
cd web && npm install && npx vite
```

Access the dev UI at `http://localhost:5173`.

## Project Structure

```
lightboot/
├── cmd/lightboot/main.go     # CLI entrypoint
├── internal/
│   ├── core/                 # Core application logic
│   │   ├── auth.go           # API authentication
│   │   ├── cache.go          # ISO extraction cache
│   │   ├── config.go         # Configuration loading
│   │   ├── database.go       # SQLite database
│   │   ├── detector.go       # ISO profile detection
│   │   ├── dhcp.go           # ProxyDHCP server
│   │   ├── isoreader.go      # ISO 9660 filesystem reader
│   │   ├── logger.go         # Structured logging
│   │   ├── menu.go           # iPXE menu generator
│   │   ├── profiles.go       # Profile loader
│   │   ├── profiles/         # Built-in YAML profiles
│   │   ├── repository.go     # ISO database repository
│   │   ├── router.go         # HTTP route definitions
│   │   ├── scanner.go        # ISO directory scanner
│   │   └── tftp.go           # TFTP server
│   ├── event/                # In-process event bus
│   └── model/                # Data models
├── profiles/                 # User profile overrides (runtime)
├── bootfiles/                # iPXE boot files
├── web/src/                  # Vue.js SPA source
│   ├── views/                # Page components
│   ├── App.vue               # Root layout
│   └── api.js                # HTTP client
├── docs/                     # MkDocs documentation source
├── docker/                   # Docker build files
└── Makefile
```

## Adding New ISO Profiles

Profiles are how LightBoot detects and boots different Linux distributions.
Adding a new profile is one of the most impactful contributions.

### Profile YAML Structure

```yaml
name: "Distribution Name"
version_regex: "-(\\d+\\.\\d+)"
architectures:
  - x86_64
boot:
  kernel: "/path/to/vmlinuz"
  initrd: "/path/to/initrd"
  append: "root=live:http://${server}/cache/${id}/filesystem.squashfs"
detect:
  files:
    - "/path/to/distro-identifier-file"
  contents:
    - file: "/path/to/release-file"
      regex: "DISTRIB_ID=SomeDistro"
```

### Profile Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Human-readable distribution name |
| `version_regex` | No | Regex to extract version from ISO filename |
| `architectures` | Yes | Supported CPU architectures |
| `boot.kernel` | Yes | Path to kernel binary inside ISO |
| `boot.initrd` | Yes | Path to initrd/initramfs inside ISO |
| `boot.append` | No | Additional kernel command-line parameters |
| `detect.files` | Yes | Files that must exist in ISO to match this profile |
| `detect.contents` | No | Files whose content must match a regex |

### How to Test a Profile

1. Add your profile YAML to `internal/core/profiles/`.
2. Build and run LightBoot:
   ```bash
   make build && ./bin/lightboot
   ```
3. Place a matching ISO in the `iso/` directory.
4. Watch the SSE log stream at `http://localhost:8080` to verify detection.

## Pull Request Process

1. Ensure your PR targets the `main` branch.
2. Update or add documentation if your change affects user-facing behavior.
3. Add a meaningful description — explain **what** changed and **why**.
4. If you add a new Go package, run `go mod tidy`.
5. The PR will be reviewed by a maintainer. We aim to respond within 5 business
days.

### PR Checklist

- [ ] Code compiles (`make build`)
- [ ] Frontend builds if applicable (`make web`)
- [ ] Tests pass (`go test ./...`)
- [ ] No new warnings from `go vet`
- [ ] Documentation updated if needed
- [ ] New profiles have been tested with a real ISO

## Style Guide

### Go

- Follow standard Go conventions (`gofmt`, `go vet`).
- Use idiomatic error handling — never panic in library code.
- Keep functions small and focused.
- Write godoc comments for exported types and functions.

### Vue.js

- Use the Composition API (`<script setup>` preferred for new code).
- Keep components under 200 lines; extract reusable pieces.
- Use the existing API client (`web/src/api.js`) for all HTTP calls.
- Follow the existing component patterns (see Dashboard.vue, ISOLibrary.vue).

### General

- Commit messages should be clear and descriptive.
- Reference issue numbers when applicable (e.g., `Fixes #42`).
- Keep pull requests focused — one feature or fix per PR.

## Reporting Bugs

If you find a bug, please open an issue with:

- **LightBoot version** (`lightboot --version`)
- **Operating system** (Linux distribution, kernel version)
- **Steps to reproduce** the issue
- **Expected vs actual behavior**
- **Log output** from the SSE stream or console

## Feature Requests

We welcome feature requests! Before opening an issue:

1. Check if it's already been requested.
2. Describe the use case — who needs this and why.
3. Suggest an implementation approach if you have one.

## License

By contributing, you agree that your contributions will be licensed under the
same [MIT License](LICENSE) that covers the project.
