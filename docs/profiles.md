# Profiles

Profiles are YAML files that teach LightBoot how to recognize and boot different Linux distributions. Each profile defines the files to look for, content patterns to match, and the kernel/initrd paths to extract.

---

## How Profiles Work

When a new ISO is detected, LightBoot runs every enabled profile against the ISO's filesystem. Each profile has two types of detection rules:

1. **`files`** — a list of file paths that must exist in the ISO
2. **`contents`** — a file path + regex pattern that must match inside a file

**The first profile where ALL rules match wins.** Profiles are tested in order — built-in profiles first, then custom profiles alphabetically.

---

## Profile YAML Schema

```yaml
# Profile identification
name: "Ubuntu 24.04 Server"
description: "Ubuntu 24.04 Noble Numbat (Server Edition)"
priority: 100

# Detection rules (ALL must match for the profile to apply)
detect:
  files:
    - "/casper/vmlinuz"
    - "/casper/initrd"
    - "/.disk/info"
  contents:
    - path: "/.disk/info"
      pattern: "Ubuntu.*24\\.04"

# Boot configuration
boot:
  kernel: "/casper/vmlinuz"
  initrd: "/casper/initrd"
  cmdline: "root=/dev/ram0 ramdisk_size=1500000 ip=dhcp url=http://{{.HTTP}}/cache/{{.ID}}/filesystem.squashfs autoinstall ds=nocloud-net;s=http://{{.HTTP}}/cloud-config/"
  extra_files:
    - "/casper/filesystem.squashfs"
```

### Field Reference

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Human-readable name (shown in boot menu) |
| `description` | string | No | Longer description of the profile |
| `priority` | int | No | Lower = tested first. Built-in profiles use 100-600 |
| `detect.files` | []string | No | File paths that must exist in the ISO root |
| `detect.contents` | []object | No | Files + regex patterns to match |
| `detect.contents[].path` | string | Yes | Path to the file to read from the ISO |
| `detect.contents[].pattern` | string | Yes | Go regex pattern (RE2) |
| `boot.kernel` | string | Yes | Path to the Linux kernel inside the ISO |
| `boot.initrd` | string | Yes | Path to the initrd/initramfs inside the ISO |
| `boot.cmdline` | string | No | Kernel command line (supports Go templates) |
| `boot.extra_files` | []string | No | Additional files to extract into the cache |

### Template Variables

Inside `boot.cmdline`, you can use Go template syntax:

| Variable | Description | Example |
|----------|-------------|--------|
| `{{.HTTP}}` | HTTP server address | `192.168.1.10:8080` |
| `{{.ID}}` | ISO database ID | `1` |
| `{{.CACHE}}` | Cache directory base URL | `http://192.168.1.10:8080/cache/1` |

---

## Built-in Profiles

LightBoot ships with these profiles embedded in the binary:

| Profile | Priority | Detection |
|---------|---------|-----------|
| **Ubuntu 24.04 Desktop** | 100 | `/casper/vmlinuz` + `/.disk/info` |
| **Ubuntu 24.04 Server** | 100 | `/casper/vmlinuz` + `/.disk/info` |
| **Debian 12** | 200 | `/install.amd/vmlinuz` + `/md5sum.txt` |
| **Fedora 40** | 300 | `/images/pxeboot/vmlinuz` + `/Fedora-Legal-README.txt` |
| **Arch Linux** | 400 | `/arch/boot/x86_64/vmlinuz-linux` |
| **Alpine 3.20** | 500 | `/boot/vmlinuz-lts` + `/etc/alpine-release` |

---

## Custom Profiles

Place custom YAML files in the `profiles/` directory. They are loaded at startup and merged with built-in profiles.

### Example: Kali Linux

```yaml
# profiles/kali.yml
name: "Kali Linux"
description: "Kali Linux (Rolling Release)"
priority: 250
detect:
  files:
    - "/install.amd/vmlinuz"
    - "/install.amd/initrd.gz"
    - "/.disk/info"
boot:
  kernel: "/install.amd/vmlinuz"
  initrd: "/install.amd/initrd.gz"
  cmdline: "root=/dev/ram0 ramdisk_size=1500000 ip=dhcp url=http://{{.HTTP}}/cache/{{.ID}}/filesystem.squashfs"
```

### Example: Pop_OS!

```yaml
# profiles/popos.yml
name: "Pop!_OS"
description: "Pop!_OS 22.04+"
priority: 150
detect:
  files:
    - "/casper/vmlinuz.efi"
    - "/casper/initrd.gz"
    - "/.disk/info"
  contents:
    - path: "/.disk/info"
      pattern: "(?i)Pop.*22\\.04|Pop.*24\\.04"
boot:
  kernel: "/casper/vmlinuz.efi"
  initrd: "/casper/initrd.gz"
  cmdline: "root=/dev/ram0 ramdisk_size=1500000 ip=dhcp url=http://{{.HTTP}}/cache/{{.ID}}/filesystem.squashfs"
```

### Testing Profiles

To test a new profile:

1. Place the `.yml` file in `profiles/`
2. Restart LightBoot
3. Check logs: `curl http://localhost:8080/api/logs/recent -H "Authorization: Bearer TOKEN"`
4. If the profile loaded correctly, you'll see: `Profile loaded: Kali Linux`
5. Upload an ISO matching the profile
6. Watch the logs for detection results

---

## Profile Priority System

Profiles are tested in order of `priority` (lower numbers first):

```
Priority 100-199: Ubuntu family
Priority 200-299: Debian family
Priority 300-399: Fedora/RHEL family
Priority 400-499: Arch family
Priority 500-599: Alpine family
Priority 600+:    Custom / catch-all
```

Custom profiles with lower priorities can override built-in profiles. This is useful when you need to handle a specific variant of a distribution differently.

---

## Advanced Detection

### Multiple Content Rules

All `contents` rules must match (AND logic):

```yaml
detect:
  contents:
    - path: "/.disk/info"
      pattern: "Ubuntu"
    - path: "/md5sum.txt"
      pattern: "(?i)server"
```

### File Paths Are Case-Sensitive

ISO9660 filesystems are case-sensitive. Use the exact case shown by `xorriso -indev file.iso -ls`.

### Escaping Regex

YAML can interfere with regex escaping. Use quotes and double-backslashes:

```yaml
pattern: "Ubuntu.*24\\.04"  # Matches "Ubuntu 24.04"
```

Or use single quotes for literal strings:

```yaml
pattern: 'Ubuntu.*24\.04'
```

---

## Contributing Profiles

If you create a profile for a distribution not yet supported, please [contribute it](contributing.md)! The community benefits from every new profile.

See `CONTRIBUTING.md` for the pull request process.
