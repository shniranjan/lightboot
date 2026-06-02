package model

import (
	"encoding/json"
	"time"
)

// BootMode represents a supported boot type.
type BootMode string

const (
	BootModeBIOS   BootMode = "bios"
	BootModeUEFI   BootMode = "uefi"
	BootModeLegacy BootMode = "legacy"
	BootModeARM64  BootMode = "arm64"
)

// ISOStatus represents the current state of an ISO in the system.
type ISOStatus string

const (
	StatusNew      ISOStatus = "new"      // discovered, not yet processed
	StatusDetected ISOStatus = "detected" // profile matched but not extracted
	StatusReady    ISOStatus = "ready"    // extracted and cached, bootable
	StatusError    ISOStatus = "error"    // processing failed
	StatusUnknown  ISOStatus = "unknown"  // no profile matched
)

// ISO represents a discovered ISO file stored in the database.
type ISO struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	SourcePath  string    `json:"source_path"`
	Size        int64     `json:"size"`
	SHA256      string    `json:"sha256"`
	Arch        string    `json:"architecture"`
	BootModes   []BootMode `json:"boot_modes"`
	Distro      string    `json:"distro"`
	Version     string    `json:"version"`
	BootProfile string    `json:"boot_profile"` // JSON-encoded BootProfile
	CachedPath  string    `json:"cached_path"`
	Status      ISOStatus `json:"status"`
	LastScanned time.Time `json:"last_scanned"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BootProfile describes how to boot an ISO.
type BootProfile struct {
	Name    string     `json:"name"    yaml:"name"`
	Distro  string     `json:"distro"  yaml:"distro"`
	Version string     `json:"version" yaml:"version"`
	Arch    string     `json:"arch"    yaml:"arch"`
	Kernel  string     `json:"kernel"  yaml:"kernel"`
	Initrd  string     `json:"initrd"  yaml:"initrd"`
	Cmdline string     `json:"cmdline" yaml:"cmdline"`
	BootModes []BootMode `json:"boot_modes" yaml:"boot_modes"`
}

// MenuItem represents a single entry in the iPXE boot menu.
type MenuItem struct {
	ID        int64  `json:"id"`
	Label     string `json:"label"`
	KernelURL string `json:"kernel_url"`
	InitrdURL string `json:"initrd_url"`
	Cmdline   string `json:"cmdline"`
}

// ToBootProfileJSON marshals a BootProfile to its JSON representation.
func (bp *BootProfile) ToBootProfileJSON() string {
	data, _ := json.Marshal(bp)
	return string(data)
}

// ParseBootProfile unmarshals a JSON boot profile string.
func ParseBootProfile(raw string) (*BootProfile, error) {
	var bp BootProfile
	if err := json.Unmarshal([]byte(raw), &bp); err != nil {
		return nil, err
	}
	return &bp, nil
}
