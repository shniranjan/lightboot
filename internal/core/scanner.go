package core

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/shniranjan/lightboot/internal/event"
	"github.com/shniranjan/lightboot/internal/model"
)

// Scanner watches the ISO directory for new/changed/removed files and emits
// events on the event bus. It also runs a periodic full scan as a fallback.
type Scanner struct {
	isoDir   string
	repo     *ISORepository
	bus      *event.EventBus
	log      *Logger
	detector *Detector
	cache    *CacheManager
	watcher  *fsnotify.Watcher
	stopCh   chan struct{}
}

// NewScanner creates a new ISO scanner.
func NewScanner(isoDir string, repo *ISORepository, bus *event.EventBus, log *Logger, detector *Detector, cache *CacheManager) *Scanner {
	return &Scanner{
		isoDir:   isoDir,
		repo:     repo,
		bus:      bus,
		log:      log,
		detector: detector,
		cache:    cache,
		stopCh:   make(chan struct{}),
	}
}

// Start begins watching the ISO directory and launches the periodic scan goroutine.
func (s *Scanner) Start(interval time.Duration) error {
	if err := os.MkdirAll(s.isoDir, 0755); err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	s.watcher = watcher

	if err := watcher.Add(s.isoDir); err != nil {
		return err
	}

	s.log.Info("scanner", "Watching directory: %s", s.isoDir)

	go s.watchLoop()
	go s.periodicScan(interval)
	go s.fullScan()

	return nil
}

// Stop shuts down the scanner.
func (s *Scanner) Stop() {
	close(s.stopCh)
	if s.watcher != nil {
		s.watcher.Close()
	}
}

func (s *Scanner) watchLoop() {
	debounceMap := make(map[string]*time.Timer)

	for {
		select {
		case <-s.stopCh:
			return
		case ev, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			path := ev.Name
			s.log.Debug("scanner", "FS event: %s %s", ev.Op, path)
			if !strings.HasSuffix(strings.ToLower(path), ".iso") {
				continue
			}
			if timer, exists := debounceMap[path]; exists {
				timer.Stop()
			}
			eventOp := ev.Op
			debounceMap[path] = time.AfterFunc(2*time.Second, func() {
				delete(debounceMap, path)
				s.handleFileEvent(path, eventOp)
			})
		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			s.log.Error("scanner", "Watcher error: %v", err)
		}
	}
}

func (s *Scanner) handleFileEvent(path string, op fsnotify.Op) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		s.log.Info("scanner", "ISO removed: %s", path)
		s.bus.Publish(event.ISORemoved, path)
		return
	}
	if err != nil {
		s.log.Error("scanner", "Stat failed for %s: %v", path, err)
		return
	}
	if info.IsDir() {
		return
	}
	if op&fsnotify.Create != 0 || op&fsnotify.Write != 0 {
		s.log.Info("scanner", "New ISO detected: %s", path)
		s.bus.Publish(event.ISOAdded, path)
	}
}

func (s *Scanner) periodicScan(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.log.Debug("scanner", "Periodic scan triggered")
			s.fullScan()
		}
	}
}

func (s *Scanner) fullScan() {
	s.log.Debug("scanner", "Full scan started for %s", s.isoDir)
	foundPaths := make(map[string]os.FileInfo)

	_ = filepath.Walk(s.isoDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".iso") {
			return nil
		}
		foundPaths[path] = info
		return nil
	})

	for path := range foundPaths {
		existing, err := s.repo.GetISOBySourcePath(path)
		if err != nil || existing == nil {
			s.log.Info("scanner", "Found new ISO (full scan): %s", path)
			s.bus.Publish(event.ISOAdded, path)
		}
	}

	allISOs, err := s.repo.GetAllISOs()
	if err != nil {
		return
	}
	for _, iso := range allISOs {
		if _, exists := foundPaths[iso.SourcePath]; !exists {
			s.log.Info("scanner", "ISO no longer on disk: %s", iso.SourcePath)
			s.bus.Publish(event.ISORemoved, iso.SourcePath)
		}
	}
	s.log.Debug("scanner", "Full scan complete")
}

// ProcessISOAdded handles an ISOAdded event by computing SHA256, running the
// detector, creating/updating the DB record, and auto-caching.
func (s *Scanner) ProcessISOAdded(path string) {
	info, err := os.Stat(path)
	if err != nil {
		s.log.Error("scanner", "Cannot stat new ISO %s: %v", path, err)
		return
	}

	result, err := s.detector.Detect(path)
	if err != nil {
		s.log.Error("scanner", "Detection failed for %s: %v", path, err)
		return
	}

	if result.SHA256 != "" {
		existing, err := s.repo.GetISOBySHA256(result.SHA256)
		if err == nil && existing != nil {
			s.log.Debug("scanner", "ISO already known by SHA256: %s", path)
			return
		}
	}

	existing, err := s.repo.GetISOBySourcePath(path)
	if err != nil {
		s.log.Error("scanner", "DB lookup failed: %v", err)
		return
	}

	iso := &model.ISO{
		Name:       filepath.Base(path),
		SourcePath: path,
		Size:       info.Size(),
		SHA256:     result.SHA256,
		Status:     model.StatusUnknown,
	}

	if result.Detected && result.Profile != nil {
		iso.Distro = result.Profile.Distro
		iso.Version = result.Profile.Version
		iso.Arch = result.Profile.Arch
		iso.BootModes = result.Profile.BootModes
		iso.BootProfile = result.Profile.ToBootProfileJSON()
		iso.Status = model.StatusDetected
		s.log.Info("scanner", "ISO detected as %q: %s", result.ProfileName, path)
	} else {
		s.log.Warn("scanner", "ISO not recognized: %s", path)
	}

	if existing != nil {
		iso.ID = existing.ID
		iso.CachedPath = existing.CachedPath
		if err := s.repo.UpdateISO(iso); err != nil {
			s.log.Error("scanner", "Failed to update ISO: %v", err)
			return
		}
		s.log.Info("scanner", "ISO updated: id=%d distro=%s", iso.ID, iso.Distro)
	} else {
		if err := s.repo.InsertISO(iso); err != nil {
			s.log.Error("scanner", "Failed to insert ISO: %v", err)
			return
		}
		s.log.Info("scanner", "ISO created: id=%d distro=%s status=%s", iso.ID, iso.Distro, iso.Status)
	}

	// --- Stage 3: Auto-cache kernel + initrd for detected ISOs ---
	if iso.Status == model.StatusDetected && s.cache != nil {
		kernelURL, initrdURL, err := s.cache.Ensure(iso)
		if err != nil {
			s.log.Error("scanner", "Cache failed for ISO %d: %v", iso.ID, err)
			iso.Status = model.StatusError
		} else {
			iso.Status = model.StatusReady
			iso.CachedPath = kernelURL // cache base URL; kernelURL is /cache/<sha256>/vmlinuz
		}

		if err := s.repo.UpdateISO(iso); err != nil {
			s.log.Error("scanner", "Failed to update cached ISO %d: %v", iso.ID, err)
			return
		}

		if iso.Status == model.StatusReady {
			s.log.Info("scanner", "ISO cached: id=%d kernel=%s initrd=%s", iso.ID, kernelURL, initrdURL)
		} else {
			s.log.Error("scanner", "ISO caching failed: id=%d", iso.ID)
		}
	}
}

// ProcessISORemoved handles an ISORemoved event by removing the DB record
// and purging the cache.
func (s *Scanner) ProcessISORemoved(path string) {
	iso, err := s.repo.GetISOBySourcePath(path)
	if err != nil || iso == nil {
		return
	}

	// Stage 3: Purge cached files
	if s.cache != nil && iso.SHA256 != "" {
		if err := s.cache.Purge(iso.SHA256); err != nil {
			s.log.Error("scanner", "Cache purge failed for ISO %d: %v", iso.ID, err)
		}
	}

	if err := s.repo.DeleteISO(iso.ID); err != nil {
		s.log.Error("scanner", "Failed to delete ISO %d: %v", iso.ID, err)
		return
	}
	s.log.Info("scanner", "ISO deleted: id=%d", iso.ID)
}
