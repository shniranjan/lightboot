package core

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/shniranjan/lightboot/internal/model"
)

// CacheManager handles extracting kernel+initrd from ISOs into a flat cache
// directory and serving extracted files via HTTP.
type CacheManager struct {
	cacheDir string
	reader   *ISOReader
	log      *Logger
}

// NewCacheManager creates a new CacheManager.
func NewCacheManager(cacheDir string, reader *ISOReader, log *Logger) *CacheManager {
	return &CacheManager{
		cacheDir: cacheDir,
		reader:   reader,
		log:      log,
	}
}

// Ensure makes the ISO boot-ready by extracting kernel and initrd into the
// cache if not already present. Returns the relative URL paths.
// Paths returned are like "/cache/<sha256>/vmlinuz" and "/cache/<sha256>/initrd".
func (cm *CacheManager) Ensure(iso *model.ISO) (kernelURL, initrdURL string, err error) {
	if iso.Status != model.StatusDetected {
		return "", "", fmt.Errorf("cannot cache ISO with status %q", iso.Status)
	}

	// Parse boot profile to know what files to extract
	profile, err := model.ParseBootProfile(iso.BootProfile)
	if err != nil || profile == nil {
		return "", "", fmt.Errorf("no boot profile for ISO %d", iso.ID)
	}

	// Cache directory: cache/<sha256>/
	cacheSubDir := filepath.Join(cm.cacheDir, iso.SHA256)

	// Create directory
	if err := os.MkdirAll(cacheSubDir, 0755); err != nil {
		return "", "", fmt.Errorf("create cache dir: %w", err)
	}

	// Kernel local path
	kernelFile := filepath.Join(cacheSubDir, "vmlinuz")
	kernelURL = fmt.Sprintf("/cache/%s/vmlinuz", iso.SHA256)

	// Initrd local path
	initrdFile := filepath.Join(cacheSubDir, "initrd")
	initrdURL = fmt.Sprintf("/cache/%s/initrd", iso.SHA256)

	// Extract kernel if not cached
	if _, err := os.Stat(kernelFile); os.IsNotExist(err) {
		cm.log.Info("cache", "Extracting kernel %s from ISO %s", profile.Kernel, iso.SourcePath)
		data, err := cm.reader.ReadFile(iso.SourcePath, profile.Kernel)
		if err != nil {
			return "", "", fmt.Errorf("extract kernel %s: %w", profile.Kernel, err)
		}
		if err := os.WriteFile(kernelFile, data, 0644); err != nil {
			return "", "", fmt.Errorf("write kernel: %w", err)
		}
		cm.log.Info("cache", "Kernel cached: %s (%d bytes)", kernelURL, len(data))
	}

	// Extract initrd if not cached
	if _, err := os.Stat(initrdFile); os.IsNotExist(err) {
		cm.log.Info("cache", "Extracting initrd %s from ISO %s", profile.Initrd, iso.SourcePath)
		data, err := cm.reader.ReadFile(iso.SourcePath, profile.Initrd)
		if err != nil {
			return "", "", fmt.Errorf("extract initrd %s: %w", profile.Initrd, err)
		}
		if err := os.WriteFile(initrdFile, data, 0644); err != nil {
			return "", "", fmt.Errorf("write initrd: %w", err)
		}
		cm.log.Info("cache", "Initrd cached: %s (%d bytes)", initrdURL, len(data))
	}

	return kernelURL, initrdURL, nil
}

// Purge removes the cached directory for an ISO by its SHA256 hash.
func (cm *CacheManager) Purge(sha256 string) error {
	cacheSubDir := filepath.Join(cm.cacheDir, sha256)
	if _, err := os.Stat(cacheSubDir); os.IsNotExist(err) {
		return nil
	}

	cm.log.Info("cache", "Purging cache: %s", cacheSubDir)
	return os.RemoveAll(cacheSubDir)
}

// computeISO256 is a helper (unused here — detector uses its own).
func computeISO256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
