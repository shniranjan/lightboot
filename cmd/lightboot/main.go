package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/shniranjan/lightboot/internal/core"
	"github.com/shniranjan/lightboot/internal/event"
)

const (
	version = "0.1.0"
	banner  = `
  _      _       _     _   ______                      _   
 | |    (_)     | |   | |  | ___ \                    | |  
 | |     _  __ _| |__ | |__| |_/ /  ___    ___   _ __ | |_ 
 | |    | |/ _  | '_ \|  _  | ___ \ / _ \  / _ \ | '_ \| __|
 | |____| | (_| | | | | | | | |_/ /| (_) || (_) || | | | |_ 
 \_____/|_|\__, |_| |_\_| |_\____/  \___/  \___/ |_| |_\__|
            __/ |                                           
           |___/                                            
`
)

func main() {
	// Parse flags
	var (
		showVersion = flag.Bool("version", false, "Print version and exit")
		showHelp    = flag.Bool("help", false, "Print help and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("LightBoot version %s\n", version)
		os.Exit(0)
	}

	if *showHelp {
		fmt.Printf("LightBoot - PXE Network Boot Manager\n")
		fmt.Printf("Version: %s\n\n", version)
		fmt.Printf("Usage: lightboot [flags] [command]\n\n")
		fmt.Printf("Flags:\n")
		fmt.Printf("  --version    Print version and exit\n")
		fmt.Printf("  --help       Print this help message\n\n")
		fmt.Printf("Commands:\n")
		fmt.Printf("  help         Print manual URL and documentation reference\n\n")
		fmt.Printf("Documentation: http://localhost:8080/docs (once running)\n")
		os.Exit(0)
	}

	// Check for subcommands
	if len(flag.Args()) > 0 {
		switch flag.Arg(0) {
		case "help":
			fmt.Println("LightBoot full documentation available at: http://localhost:8080/docs")
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\nRun 'lightboot --help' for usage.\n", flag.Arg(0))
			os.Exit(1)
		}
	}

	// Print banner
	fmt.Print(banner)
	fmt.Printf("LightBoot v%s — PXE Network Boot Manager\n", version)
	fmt.Print("\n")

	// Load configuration
	cfg, err := core.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// --- Stage 1: Initialize database ---
	dbPath := filepath.Join(cfg.DataDir, "lightboot.db")
	db, err := core.OpenDatabase(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	fmt.Printf("Database: %s\n", dbPath)

	// --- Stage 1: Initialize event bus ---
	bus := event.NewEventBus()
	defer bus.Close()

	// --- Stage 1: Initialize log ring buffer ---
	logBuffer := core.NewLogRingBuffer(1000)

	// --- Stage 1: Initialize logger ---
	logLevel := core.LogInfo
	switch cfg.LogLevel {
	case "debug":
		logLevel = core.LogDebug
	case "warn":
		logLevel = core.LogWarn
	case "error":
		logLevel = core.LogError
	}
	logger := core.NewLogger(logBuffer, bus, logLevel)
	if cfg.LogFile != "" {
		if err := logger.SetLogFile(cfg.LogFile); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not open log file %s: %v\n", cfg.LogFile, err)
		} else {
			fmt.Printf("Logging to file: %s\n", cfg.LogFile)
		}
	}
	defer logger.Close()

	// --- Stage 1: Initialize repository ---
	repo := core.NewISORepository(db)

	// --- Stage 2: Load profiles ---
	profileLoader := core.NewProfileLoader(cfg.ProfilesDir, logger)
	profiles, err := profileLoader.LoadAll()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading profiles: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Profiles loaded: %d\n", len(profiles))

	// --- Stage 2: ISO reader + detector ---
	isoReader := core.NewISOReader(logger)
	detector := core.NewDetector(profiles, isoReader, logger)

	// --- Stage 3: Initialize cache manager ---
	cacheManager := core.NewCacheManager(cfg.CacheDir, isoReader, logger)

	// --- Stage 2/3: Initialize scanner with detector + cache ---
	scanner := core.NewScanner(cfg.ISODir, repo, bus, logger, detector, cacheManager)
	if err := scanner.Start(time.Duration(cfg.ScannerInterval) * time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting scanner: %v\n", err)
		os.Exit(1)
	}
	defer scanner.Stop()

	// --- Stage 1: Subscribe to scanner events ---
	isoAddedCh := bus.Subscribe(event.ISOAdded)
	isoRemovedCh := bus.Subscribe(event.ISORemoved)

	go func() {
		for msg := range isoAddedCh {
			if path, ok := msg.(string); ok {
				scanner.ProcessISOAdded(path)
			}
		}
	}()

	go func() {
		for msg := range isoRemovedCh {
			if path, ok := msg.(string); ok {
				scanner.ProcessISORemoved(path)
			}
		}
	}()

	// --- Stage 1: Initialize SSE handler ---
	sseHandler := core.NewSSEHandler(bus, logBuffer)

	// --- Stage 3: Initialize menu generator ---
	menuGen := core.NewMenuGenerator(repo, cacheManager, logger, cfg.BootFilesDir)

	// --- Stage 4: Start TFTP server ---
	tftpSrv := core.NewTFTPServer(cfg.TFTPListenAddr(), cfg.BootFilesDir, logger)
	if err := tftpSrv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting TFTP server: %v\n", err)
		os.Exit(1)
	}
	defer tftpSrv.Stop()

	// --- Stage 4: Start DHCP Proxy server (if configured) ---
	if cfg.DHCPProxyEnabled {
		serverIP := core.GetLocalIP(cfg.DHCPListenAddr())
		tftpServerAddr := fmt.Sprintf("%s:%d", serverIP, cfg.TFTPPort)
		httpServerAddr := fmt.Sprintf("%s:%d", serverIP, cfg.HTTPPort)

		fmt.Printf("DHCP Proxy: listening on %s (TFTP: %s, HTTP: %s)\n",
			cfg.DHCPListenAddr(), tftpServerAddr, httpServerAddr)

		dhcpSrv := core.NewDHCPProxyServer(cfg.DHCPListenAddr(), httpServerAddr, tftpServerAddr, logger)
		if err := dhcpSrv.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting DHCP proxy: %v\n", err)
			logger.Warn("main", "DHCP proxy failed to start: %v", err)
		} else {
			defer dhcpSrv.Stop()
		}
	}

	// Print startup info
	fmt.Printf("HTTP server listening on http://%s\n", cfg.HTTPListenAddr())
	fmt.Printf("API token: %s\n", cfg.GetAPIToken())
	fmt.Printf("ISO directory: %s\n", cfg.ISODir)
	fmt.Printf("Cache directory: %s\n", cfg.CacheDir)
	fmt.Println()

	// --- Create rate limiter for auth ---
	authRateLimiter := core.NewRateLimiter(10, 1*time.Minute, 5*time.Minute)
	
	// --- Build router with all dependencies ---
	deps := &core.RouterDeps{
		Config:        cfg,
		EventBus:      bus,
		Repository:    repo,
		SSEHandler:    sseHandler,
		LogBuffer:     logBuffer,
		MenuGenerator: menuGen,
		CacheDir:      cfg.CacheDir,
		Scanner:       scanner,
		RateLimiter:   authRateLimiter,
	}

	// Start HTTP server with security middleware
	router := core.NewRouter(deps)
	secureHandler := core.CSPMiddleware(router)

	srv := &http.Server{
		Addr:         cfg.HTTPListenAddr(),
		Handler:      secureHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run server in goroutine
	go func() {
		fmt.Printf("Server started. Press Ctrl+C to stop.\n")
		logger.Info("main", "LightBoot server started on %s", cfg.HTTPListenAddr())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	fmt.Printf("\nReceived %v, shutting down gracefully...\n", sig)
	logger.Info("main", "Shutting down: received %v", sig)

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error during shutdown: %v\n", err)
	}

	fmt.Println("LightBoot stopped.")
}
