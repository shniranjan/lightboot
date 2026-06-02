package model

import "regexp"

// Profile is a YAML-defined boot profile with detection rules.
type Profile struct {
	// Display metadata
	Name    string `yaml:"name"    json:"name"`
	Distro  string `yaml:"distro"  json:"distro"`
	Version string `yaml:"version" json:"version"`
	Arch    string `yaml:"arch"    json:"arch"`

	// Detection rules
	Detect DetectRules `yaml:"detect" json:"detect"`

	// Boot parameters
	Kernel    string     `yaml:"kernel"    json:"kernel"`
	Initrd    string     `yaml:"initrd"    json:"initrd"`
	Cmdline   string     `yaml:"cmdline"   json:"cmdline"`
	BootModes []BootMode `yaml:"boot_modes" json:"boot_modes"`
}

// DetectRules defines how to identify an ISO as matching this profile.
type DetectRules struct {
	Files    []string        `yaml:"files"    json:"files"`
	Contents []ContentRule   `yaml:"contents" json:"contents"`
}

// ContentRule checks that a file inside the ISO contains text matching a regex.
type ContentRule struct {
	Path  string `yaml:"path"  json:"path"`
	Regex string `yaml:"regex" json:"regex"`
}

// ToBootProfile converts a Profile into a BootProfile suitable for the ISO record.
func (p *Profile) ToBootProfile() *BootProfile {
	bp := &BootProfile{
		Name:      p.Name,
		Distro:    p.Distro,
		Version:   p.Version,
		Arch:      p.Arch,
		Kernel:    p.Kernel,
		Initrd:    p.Initrd,
		Cmdline:   p.Cmdline,
		BootModes: p.BootModes,
	}
	if bp.BootModes == nil {
		bp.BootModes = []BootMode{BootModeBIOS, BootModeUEFI}
	}
	return bp
}

// Validate checks required fields and compiles content regexes.
func (p *Profile) Validate() error {
	if len(p.Detect.Files) == 0 && len(p.Detect.Contents) == 0 {
		// A profile must have at least one detection rule
	}
	// Precompile regexes so we fail early
	for _, cr := range p.Detect.Contents {
		if _, err := regexp.Compile(cr.Regex); err != nil {
			return err
		}
	}
	return nil
}

// GetCompiledRegexes returns precompiled regex patterns for content rules.
func (d *DetectRules) GetCompiledRegexes() (map[string]*regexp.Regexp, error) {
	m := make(map[string]*regexp.Regexp, len(d.Contents))
	for _, cr := range d.Contents {
		re, err := regexp.Compile(cr.Regex)
		if err != nil {
			return nil, err
		}
		m[cr.Path] = re
	}
	return m, nil
}
