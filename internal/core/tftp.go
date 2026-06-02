package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pin/tftp"
)

// TFTPServer serves boot files (undionly.kpxe, ipxe.efi, snponly.efi) to PXE
// clients. It maps BIOS clients to undionly.kpxe and UEFI clients to ipxe.efi
// (or snponly.efi for ARM64).
type TFTPServer struct {
	listenAddr   string
	bootFilesDir string
	log          *Logger
	srv          *tftp.Server
}

// NewTFTPServer creates a new TFTP server.
func NewTFTPServer(listenAddr, bootFilesDir string, log *Logger) *TFTPServer {
	return &TFTPServer{
		listenAddr:   listenAddr,
		bootFilesDir: bootFilesDir,
		log:          log,
	}
}

// Start begins listening for TFTP requests. Returns immediately; handles
// requests in a background goroutine.
func (t *TFTPServer) Start() error {
	// Ensure boot files directory exists
	if err := os.MkdirAll(t.bootFilesDir, 0755); err != nil {
		return fmt.Errorf("create boot files dir %s: %w", t.bootFilesDir, err)
	}

	// Create the TFTP server with a read-only handler pointing at boot files.
	// Signature: func(filename string, rf io.ReaderFrom) error
	t.srv = tftp.NewServer(t.handleRead, nil)

	ready := make(chan struct{})
	go func() {
		t.log.Info("tftp", "TFTP server listening on %s", t.listenAddr)
		t.log.Info("tftp", "Serving boot files from: %s", t.bootFilesDir)
		close(ready)
		if err := t.srv.ListenAndServe(t.listenAddr); err != nil {
			t.log.Error("tftp", "TFTP server error: %v", err)
		}
	}()

	// Wait for the goroutine to signal readiness
	<-ready

	return nil
}

// Stop shuts down the TFTP server.
func (t *TFTPServer) Stop() {
	if t.srv != nil {
		// The pin/tftp library can panic during Shutdown if ListenAndServe
		// hasn't fully initialized its internal listener. Handle gracefully.
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.log.Warn("tftp", "TFTP shutdown recovered from panic: %v", r)
				}
			}()
			t.srv.Shutdown()
		}()
		t.log.Info("tftp", "TFTP server stopped")
	}
}

// handleRead is called by the TFTP library for every RRQ (read request).
// It maps the requested filename to the appropriate boot file.
func (t *TFTPServer) handleRead(filename string, rf io.ReaderFrom) error {
	// Map common DHCP user-class / filename requests to the correct file
	physicalFile := t.mapFilename(filename)

	fullPath := filepath.Join(t.bootFilesDir, physicalFile)

	// Check if the file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		t.log.Error("tftp", "Requested file not found: %s (mapped from %s)", fullPath, filename)
		return fmt.Errorf("file not found: %s", filename)
	}

	t.log.Info("tftp", "Serving %s (%d bytes) for request %s", physicalFile, info.Size(), filename)

	file, err := os.Open(fullPath)
	if err != nil {
		t.log.Error("tftp", "Failed to open %s: %v", fullPath, err)
		return err
	}
	defer file.Close()

	// Transfer the file; the tftp library handles block sizes and retransmits
	_, err = rf.ReadFrom(file)
	return err
}

// mapFilename translates PXE DHCP user-class filenames to physical files.
//
// Common patterns:
//   - iPXE (all): requests "undionly.kpxe" via DHCP option 67
//   - BIOS PXE ROM: requests "undionly.kpxe"
//   - UEFI x64: requests "ipxe.efi"
//   - UEFI ARM64: requests "snponly.efi"
func (t *TFTPServer) mapFilename(filename string) string {
	base := filepath.Base(filename)

	switch base {
	case "undionly.kpxe", "undionly.kpxe.0":
		return "undionly.kpxe"
	case "ipxe.efi", "ipxe.efi.0":
		return "ipxe.efi"
	case "snponly.efi", "snponly.efi.0":
		return "snponly.efi"
	default:
		return base
	}
}
