package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/shniranjan/lightboot/internal/model"
)

// Detector matches ISO files against loaded profiles and populates the
// ISO record with detected metadata.
type Detector struct {
	profiles []*model.Profile
	reader   *ISOReader
	log      *Logger
}

// NewDetector creates a new Detector.
func NewDetector(profiles []*model.Profile, reader *ISOReader, log *Logger) *Detector {
	return &Detector{
		profiles: profiles,
		reader:   reader,
		log:      log,
	}
}

// DetectResult holds the outcome of detecting an ISO.
type DetectResult struct {
	Profile      *model.BootProfile
	ProfileName  string
	Detected     bool
	SHA256       string
}

// Detect opens the ISO, computes its SHA256, lists files, and tries each
// profile until one matches.
func (d *Detector) Detect(isoPath string) (*DetectResult, error) {
	// Compute SHA256 of the ISO file
	sha256Hash, err := computeSHA256(isoPath)
	if err != nil {
		return nil, fmt.Errorf("compute SHA256: %w", err)
	}

	result := &DetectResult{
		SHA256: sha256Hash,
	}

	// List files inside the ISO
	files, err := d.reader.ListFiles(isoPath)
	if err != nil {
		d.log.Warn("detector", "Cannot read ISO %s: %v — falling back to xorriso", isoPath, err)
		return result, nil
	}

	// Build a set for quick file existence checks
	fileSet := make(map[string]bool, len(files))
	for _, f := range files {
		fileSet[f] = true
	}

	d.log.Debug("detector", "ISO %s has %d files", isoPath, len(files))

	// Try each profile
	for _, p := range d.profiles {
		if d.matchProfile(p, fileSet, isoPath) {
			bp := p.ToBootProfile()
			result.Profile = bp
			result.ProfileName = p.Name
			result.Detected = true
			d.log.Info("detector", "ISO %s matched profile %q", isoPath, p.Name)
			return result, nil
		}
	}

	d.log.Info("detector", "ISO %s did not match any profile", isoPath)
	return result, nil
}

// matchProfile checks whether an ISO matches a single profile.
func (d *Detector) matchProfile(p *model.Profile, fileSet map[string]bool, isoPath string) bool {
	// Check file existence rules
	for _, requiredFile := range p.Detect.Files {
		if !fileSet[requiredFile] {
			return false
		}
	}

	// Check content rules
	if len(p.Detect.Contents) > 0 {
		compiled, err := p.Detect.GetCompiledRegexes()
		if err != nil {
			d.log.Warn("detector", "Profile %q has invalid regex: %v", p.Name, err)
			return false
		}

		for _, cr := range p.Detect.Contents {
			re, ok := compiled[cr.Path]
			if !ok {
				continue
			}

			data, err := d.reader.ReadFile(isoPath, cr.Path)
			if err != nil {
				d.log.Debug("detector", "Cannot read %s in ISO: %v", cr.Path, err)
				return false
			}

			if !re.Match(data) {
				return false
			}
		}
	}

	return true
}

// computeSHA256 returns the hex-encoded SHA256 hash of a file.
func computeSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// Helper to compile regexes from content rules.
func compileContentRegexes(rules []model.ContentRule) (map[string]*regexp.Regexp, error) {
	m := make(map[string]*regexp.Regexp, len(rules))
	for _, cr := range rules {
		re, err := regexp.Compile(cr.Regex)
		if err != nil {
			return nil, fmt.Errorf("compile regex for %s: %w", cr.Path, err)
		}
		m[cr.Path] = re
	}
	return m, nil
}
