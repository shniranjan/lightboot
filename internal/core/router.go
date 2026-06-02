package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	root "github.com/shniranjan/lightboot"
	"github.com/shniranjan/lightboot/internal/event"
	"github.com/shniranjan/lightboot/web"
)

// RouterDeps holds all dependencies needed by the HTTP router.
type RouterDeps struct {
	RateLimiter   *RateLimiter
	Config        *Config
	EventBus      *event.EventBus
	Repository    *ISORepository
	SSEHandler    *SSEHandler
	LogBuffer     *LogRingBuffer
	MenuGenerator *MenuGenerator
	CacheDir      string
	Scanner       *Scanner
}

// NewRouter builds the HTTP handler with all routes.
func NewRouter(deps *RouterDeps) http.Handler {
	mux := http.NewServeMux()

	// --- Public boot endpoints (no auth required) ---

	if deps.MenuGenerator != nil {
		serverAddr := deps.Config.HTTPListenAddr()
		if strings.HasPrefix(serverAddr, "0.0.0.0") {
			serverAddr = strings.Replace(serverAddr, "0.0.0.0", "localhost", 1)
		}
		baseURL := "http://" + serverAddr

		mux.HandleFunc("GET /api/boot/ipxe", deps.MenuGenerator.ServeIPXEHandler(serverAddr))
		mux.HandleFunc("GET /api/boot/chain", func(w http.ResponseWriter, r *http.Request) {
			script := deps.MenuGenerator.GenerateChainIPXEScript(serverAddr)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Write([]byte(script))
		})
		mux.HandleFunc("GET /api/boot/menu", deps.MenuGenerator.ServeBootMenuJSONHandler(baseURL))
	}

	// Cache file serving
	if deps.CacheDir != "" {
		cacheFS := http.FileServer(http.Dir(deps.CacheDir))
		mux.Handle("/cache/", http.StripPrefix("/cache/", cacheFS))
	}

	// Public health endpoint
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// SSE log stream (public: EventSource cannot send Authorization headers)
	if deps.SSEHandler != nil {
		mux.HandleFunc("GET /api/logs/stream", deps.SSEHandler.ServeHTTP)
	}

	// --- Authenticated API routes ---
	apiMux := http.NewServeMux()

	if deps.Repository != nil {
		apiMux.HandleFunc("GET /api/isos", deps.handleListISOs())
		apiMux.HandleFunc("DELETE /api/isos/{id}", deps.handleDeleteISO())
		apiMux.HandleFunc("POST /api/isos/upload", deps.handleUploadISO())
	}

	apiMux.HandleFunc("POST /api/scan", deps.handleTriggerScan())
	apiMux.HandleFunc("GET /api/config", deps.handleGetConfig())
	apiMux.HandleFunc("POST /api/config/regenerate-token", deps.handleRegenerateToken())

	apiMux.HandleFunc("GET /api/logs/recent", deps.handleRecentLogs())

	// Wrap API routes with auth
	authedAPI := AuthMiddleware(apiMux, deps.Config, deps.RateLimiter)
	mux.Handle("/api/", authedAPI)

	// --- Documentation site (public) ---
	docsFS, docsErr := fs.Sub(root.DocsFS, "site")
	if docsErr == nil {
		mux.Handle("/docs/", http.StripPrefix("/docs", http.FileServer(http.FS(docsFS))))
	}

	// --- SPA fallback ---
	webFS, err := fs.Sub(web.Dist, "dist")
	if err == nil {
		spa := spaHandler{content: http.FS(webFS), fsContent: webFS}
		mux.Handle("/", spa)
	}

	return mux
}

// --- Handler methods ---

func (d *RouterDeps) handleListISOs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isos, err := d.Repository.GetAllISOs()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to query ISOs"})
			return
		}
		type isoBrief struct {
			ID           int64  `json:"id"`
			Name         string `json:"name"`
			Size         int64  `json:"size"`
			Status       string `json:"status"`
			Distro       string `json:"distro"`
			Version      string `json:"version"`
			Architecture string `json:"architecture"`
			SHA256       string `json:"sha256"`
		}
		results := make([]isoBrief, 0, len(isos))
		for _, iso := range isos {
			results = append(results, isoBrief{
				ID:           iso.ID,
				Name:         iso.Name,
				Size:         iso.Size,
				Status:       string(iso.Status),
				Distro:       iso.Distro,
				Version:      iso.Version,
				Architecture: iso.Arch,
				SHA256:       iso.SHA256,
			})
		}
		writeJSON(w, http.StatusOK, results)
	}
}

func (d *RouterDeps) handleDeleteISO() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		var id int64
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil || id <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid ISO id"})
			return
		}
		if err := d.Repository.DeleteISO(id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete ISO"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func (d *RouterDeps) handleUploadISO() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		maxSize := d.Config.MaxUploadSize
		if maxSize <= 0 {
			maxSize = 20 * 1024 * 1024 * 1024
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxSize)

		if err := r.ParseMultipartForm(maxSize); err != nil {
			writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "file too large"})
			return
		}
		defer r.MultipartForm.RemoveAll()

		file, header, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file field"})
			return
		}
		defer file.Close()

		if !strings.HasSuffix(strings.ToLower(header.Filename), ".iso") {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "only .iso files are accepted"})
			return
		}

		// Verify ISO 9660 magic bytes: CD001 at offset 32769
		magicBuf := make([]byte, 65536)
		magicN, _ := file.Read(magicBuf)
		if magicN < 32774 || !bytes.Equal(magicBuf[32769:32774], []byte("CD001")) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file does not appear to be a valid ISO 9660 image"})
			return
		}
		// Seek back to start for the io.Copy below
		file.Seek(0, io.SeekStart)

		isoDir := d.Config.ISODir
		if err := os.MkdirAll(isoDir, 0755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create iso directory"})
			return
		}

		destPath := filepath.Join(isoDir, header.Filename)
		destFile, err := os.Create(destPath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create file"})
			return
		}
		defer destFile.Close()

		written, err := io.Copy(destFile, file)
		if err != nil {
			os.Remove(destPath)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "upload failed"})
			return
		}

		if d.EventBus != nil {
			d.EventBus.Publish(event.ISOAdded, destPath)
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "uploaded",
			"name":   header.Filename,
			"size":   written,
			"path":   destPath,
		})
	}
}

func (d *RouterDeps) handleTriggerScan() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.EventBus != nil {
			d.EventBus.Publish(event.ISOChanged, "manual-scan")
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "scan triggered"})
	}
}

func (d *RouterDeps) handleGetConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := map[string]interface{}{
			"version":              "0.2.0",
			"http_listen":          d.Config.HTTPListenAddr(),
			"tftp_listen":          d.Config.TFTPListenAddr(),
			"dhcp_listen":          d.Config.DHCPListenAddr(),
			"iso_dir":              d.Config.ISODir,
			"cache_dir":            d.Config.CacheDir,
			"data_dir":             d.Config.DataDir,
			"bootfiles_dir":        d.Config.BootFilesDir,
			"dhcp_proxy_enabled":   d.Config.DHCPProxyEnabled,
			"max_upload_size":      d.Config.MaxUploadSize,
			"scanner_interval":     d.Config.ScannerInterval,
			"log_level":            d.Config.LogLevel,
			"log_file":             d.Config.LogFile,
		}
		writeJSON(w, http.StatusOK, cfg)
	}
}

func (d *RouterDeps) handleRegenerateToken() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		newToken, err := generateAPIToken()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
			return
		}
		tokenPath := d.Config.APITokenPath
		if err := os.MkdirAll(filepath.Dir(tokenPath), 0755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to write token"})
			return
		}
		if err := os.WriteFile(tokenPath, []byte(newToken), 0600); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to write token"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"token": newToken})
	}
}

func (d *RouterDeps) handleRecentLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.LogBuffer == nil {
			writeJSON(w, http.StatusOK, []interface{}{})
			return
		}
		entries := d.LogBuffer.Recent(50)
		writeJSON(w, http.StatusOK, entries)
	}
}

// --- SPA handler ---

type spaHandler struct {
	content   http.FileSystem
	fsContent fs.FS
}

func (s spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "index.html"
	}

	f, err := s.content.Open(path)
	if err == nil {
		f.Close()
		http.FileServer(s.content).ServeHTTP(w, r)
		return
	}

	// Fallback to index.html for SPA routing
	indexData, err := fs.ReadFile(s.fsContent, "index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexData)
}

// --- Helper ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
