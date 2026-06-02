package core

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/kdomanski/iso9660"
)

// ISOReader provides methods to read files from an ISO image.
type ISOReader struct {
	log *Logger
}

// NewISOReader creates a new ISOReader.
func NewISOReader(log *Logger) *ISOReader {
	return &ISOReader{log: log}
}

// ListFiles returns a list of file paths inside the ISO.
// It tries the native iso9660 library first, then falls back to xorriso.
func (r *ISOReader) ListFiles(isoPath string) ([]string, error) {
	files, err := r.listWithISO9660(isoPath)
	if err != nil {
		r.log.Warn("isoreader", "iso9660 library failed for %s: %v, falling back to xorriso", isoPath, err)
		return r.listWithXorriso(isoPath)
	}
	return files, nil
}

// ReadFile reads the contents of a specific file inside the ISO.
func (r *ISOReader) ReadFile(isoPath, internalPath string) ([]byte, error) {
	data, err := r.readWithISO9660(isoPath, internalPath)
	if err != nil {
		r.log.Warn("isoreader", "iso9660 read failed for %s:%s: %v, falling back to xorriso", isoPath, internalPath, err)
		return r.readWithXorriso(isoPath, internalPath)
	}
	return data, nil
}

// ---------------------------------------------------------------------------
// Native iso9660 implementation (tree-based API)
// ---------------------------------------------------------------------------

func (r *ISOReader) listWithISO9660(isoPath string) ([]string, error) {
	f, err := os.Open(isoPath)
	if err != nil {
		return nil, fmt.Errorf("open ISO: %w", err)
	}
	defer f.Close()

	img, err := iso9660.OpenImage(f)
	if err != nil {
		return nil, fmt.Errorf("parse ISO: %w", err)
	}

	root, err := img.RootDir()
	if err != nil {
		return nil, fmt.Errorf("get root dir: %w", err)
	}

	var files []string
	err = walkISOFile(root, "", func(path string, entry *iso9660.File) error {
		if !entry.IsDir() {
			files = append(files, normalizePath(path))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk ISO: %w", err)
	}

	return files, nil
}

func (r *ISOReader) readWithISO9660(isoPath, internalPath string) ([]byte, error) {
	f, err := os.Open(isoPath)
	if err != nil {
		return nil, fmt.Errorf("open ISO: %w", err)
	}
	defer f.Close()

	img, err := iso9660.OpenImage(f)
	if err != nil {
		return nil, fmt.Errorf("parse ISO: %w", err)
	}

	root, err := img.RootDir()
	if err != nil {
		return nil, fmt.Errorf("get root dir: %w", err)
	}

	// Navigate to the file using the tree
	entry, err := findISOFile(root, internalPath)
	if err != nil {
		return nil, fmt.Errorf("find %s in ISO: %w", internalPath, err)
	}

	// Read file contents
	return io.ReadAll(entry.Reader())
}

// walkISOFile recursively walks the ISO directory tree.
func walkISOFile(dir *iso9660.File, prefix string, visitor func(string, *iso9660.File) error) error {
	children, err := dir.GetChildren()
	if err != nil {
		return err
	}

	for _, child := range children {
		name := child.Name()
		fullPath := name
		if prefix != "" {
			fullPath = prefix + "/" + name
		}

		if err := visitor(fullPath, child); err != nil {
			return err
		}

		if child.IsDir() {
			if err := walkISOFile(child, fullPath, visitor); err != nil {
				return err
			}
		}
	}

	return nil
}

// findISOFile locates a file entry by path inside the ISO tree.
func findISOFile(dir *iso9660.File, filePath string) (*iso9660.File, error) {
	parts := strings.Split(strings.Trim(filePath, "/"), "/")
	current := dir

	for i, part := range parts {
		children, err := current.GetChildren()
		if err != nil {
			return nil, fmt.Errorf("read dir %s: %w", part, err)
		}

		found := false
		for _, child := range children {
			if strings.EqualFold(child.Name(), part) {
				if i == len(parts)-1 {
					return child, nil
				}
				if !child.IsDir() {
					return nil, fmt.Errorf("%s is not a directory", part)
				}
				current = child
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("file not found: %s", part)
		}
	}

	return nil, fmt.Errorf("empty path")
}

// ---------------------------------------------------------------------------
// xorriso fallback (invokes external command)
// ---------------------------------------------------------------------------

func (r *ISOReader) listWithXorriso(isoPath string) ([]string, error) {
	if _, err := exec.LookPath("xorriso"); err != nil {
		return nil, fmt.Errorf("xorriso not found in PATH; install xorriso for ISO reading support")
	}

	cmd := exec.Command("xorriso", "-indev", isoPath, "-lsl", "--")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("xorriso list failed: %w\nOutput: %s", err, string(output))
	}

	var files []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "d") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 8 {
			filePath := fields[len(fields)-1]
			files = append(files, normalizePath(filePath))
		}
	}

	return files, nil
}

func (r *ISOReader) readWithXorriso(isoPath, internalPath string) ([]byte, error) {
	if _, err := exec.LookPath("xorriso"); err != nil {
		return nil, fmt.Errorf("xorriso not found in PATH")
	}

	tmpDir, err := os.MkdirTemp("", "lightboot-xorriso-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command("xorriso", "-osirrox", "on", "-indev", isoPath,
		"-extract", internalPath, filepath.Join(tmpDir, "extracted"), "--")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("xorriso extract failed: %w\nOutput: %s", err, string(output))
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "extracted", filepath.Base(internalPath)))
	if err != nil {
		return nil, fmt.Errorf("read extracted file: %w", err)
	}

	return data, nil
}

// normalizePath ensures the path uses forward slashes and is lowercase for
// consistent matching.
func normalizePath(path string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.ToSlash(path), "/"))
}
