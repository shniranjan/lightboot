package core

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/shniranjan/lightboot/internal/model"
)

//go:embed all:profiles
var builtinProfilesFS embed.FS

// ProfileLoader loads built-in and user-defined profiles.
type ProfileLoader struct {
	builtinDir string // embedded directory name ("profiles")
	userDir    string // optional user profiles directory on disk
	log        *Logger
}

// NewProfileLoader creates a new ProfileLoader.
func NewProfileLoader(userDir string, log *Logger) *ProfileLoader {
	return &ProfileLoader{
		builtinDir: "profiles",
		userDir:    userDir,
		log:        log,
	}
}

// LoadAll returns all profiles: built-in first, then user overrides.
// User profiles with the same Name override built-in ones.
func (l *ProfileLoader) LoadAll() ([]*model.Profile, error) {
	var all []*model.Profile

	// Load built-in profiles
	builtins, err := l.loadFromEmbed()
	if err != nil {
		return nil, fmt.Errorf("failed to load built-in profiles: %w", err)
	}
	all = append(all, builtins...)

	// Load user profiles (override built-ins by name)
	if l.userDir != "" {
		userProfiles, err := l.loadFromDirectory(l.userDir)
		if err != nil {
			l.log.Warn("profiles", "Failed to load user profiles from %s: %v", l.userDir, err)
		} else {
			// Override: remove any built-in with the same name
			for _, up := range userProfiles {
				for i, bp := range all {
					if bp.Name == up.Name {
						all = append(all[:i], all[i+1:]...)
						break
					}
				}
				all = append(all, up)
			}
		}
	}

	// Validate all profiles
	var valid []*model.Profile
	for _, p := range all {
		if err := p.Validate(); err != nil {
			l.log.Warn("profiles", "Skipping invalid profile '%s': %v", p.Name, err)
			continue
		}
		valid = append(valid, p)
	}

	l.log.Info("profiles", "Loaded %d profiles", len(valid))
	return valid, nil
}

// loadFromEmbed reads YAML files from the embedded profiles directory.
func (l *ProfileLoader) loadFromEmbed() ([]*model.Profile, error) {
	entries, err := builtinProfilesFS.ReadDir(l.builtinDir)
	if err != nil {
		// No embedded profiles — not an error
		return nil, nil
	}

	var profiles []*model.Profile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		data, err := builtinProfilesFS.ReadFile(filepath.Join(l.builtinDir, entry.Name()))
		if err != nil {
			l.log.Warn("profiles", "Failed to read built-in profile %s: %v", entry.Name(), err)
			continue
		}

		var p model.Profile
		if err := yaml.Unmarshal(data, &p); err != nil {
			l.log.Warn("profiles", "Failed to parse built-in profile %s: %v", entry.Name(), err)
			continue
		}

		profiles = append(profiles, &p)
	}

	return profiles, nil
}

// loadFromDirectory reads YAML profile files from a directory on disk.
func (l *ProfileLoader) loadFromDirectory(dir string) ([]*model.Profile, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil // directory doesn't exist yet — fine
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", dir, err)
	}

	var profiles []*model.Profile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(fullPath)
		if err != nil {
			l.log.Warn("profiles", "Failed to read user profile %s: %v", entry.Name(), err)
			continue
		}

		var p model.Profile
		if err := yaml.Unmarshal(data, &p); err != nil {
			l.log.Warn("profiles", "Failed to parse user profile %s: %v", entry.Name(), err)
			continue
		}

		profiles = append(profiles, &p)
	}

	return profiles, nil
}
