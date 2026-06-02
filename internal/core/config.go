package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds all runtime configuration.
type Config struct {
	// HTTP server settings
	HTTPPort    int    `yaml:"http_port"`
	HTTPAddress string `yaml:"http_address"`

	// TFTP settings
	TFTPPort    int    `yaml:"tftp_port"`
	TFTPAddress string `yaml:"tftp_address"`

	// DHCP Proxy settings (Stage 4)
	DHCPProxyEnabled bool   `yaml:"dhcp_proxy_enabled"`
	DHCPProxyPort    int    `yaml:"dhcp_proxy_port"`
	DHCPProxyAddress string `yaml:"dhcp_proxy_address"`

	// Directory paths
	ISODir       string `yaml:"iso_dir"`
	CacheDir     string `yaml:"cache_dir"`
	DataDir      string `yaml:"data_dir"`
	ProfilesDir  string `yaml:"profiles_dir"`
	BootFilesDir string `yaml:"bootfiles_dir"`

	// API token path
	APITokenPath string `yaml:"api_token_path"`

	// Maximum upload size in bytes (default 20 GB)
	MaxUploadSize int64 `yaml:"max_upload_size"`

	// ISO scanner interval in seconds
	ScannerInterval int `yaml:"scanner_interval"`

	// Log level: debug, info, warn, error
	LogLevel string `yaml:"log_level"`

	// Write logs to file as well
	LogFile string `yaml:"log_file"`

	// The API token (loaded from file or environment, not from YAML)
	apiToken string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		HTTPPort:         8080,
		HTTPAddress:      "0.0.0.0",
		TFTPPort:         69,
		TFTPAddress:      "0.0.0.0",
		DHCPProxyEnabled: true,
		DHCPProxyPort:    67,
		DHCPProxyAddress: "0.0.0.0",
		ISODir:           "./iso",
		CacheDir:         "./cache",
		DataDir:          "./data",
		ProfilesDir:      "./profiles",
		BootFilesDir:     "./bootfiles",
		APITokenPath:     "./data/.api_token",
		MaxUploadSize:    20 * 1024 * 1024 * 1024, // 20 GB
		ScannerInterval:  300,                      // 5 minutes
		LogLevel:         "info",
	}
}

// LoadConfig reads config.yaml, overrides with environment variables, loads
// or generates the API token, and returns the final Config.
func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	// 1. Try to load from config.yaml (if present)
	if data, err := os.ReadFile("config.yaml"); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config.yaml: %w", err)
		}
	}

	// 2. Environment variable overrides
	cfg.applyEnvOverrides()

	// 3. Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory %s: %w", cfg.DataDir, err)
	}

	// 4. Load or generate API token
	tokenPath := cfg.APITokenPath

	// Check environment first
	if envToken := os.Getenv("LIGHTBOOT_API_TOKEN"); envToken != "" {
		cfg.apiToken = envToken
	} else {
		// Try reading from file
		if data, err := os.ReadFile(tokenPath); err == nil {
			cfg.apiToken = strings.TrimSpace(string(data))
		} else {
			// Generate a new token
			token, err := generateAPIToken()
			if err != nil {
				return nil, fmt.Errorf("failed to generate API token: %w", err)
			}
			cfg.apiToken = token

			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(tokenPath), 0755); err != nil {
				return nil, fmt.Errorf("failed to create token directory: %w", err)
			}

			if err := os.WriteFile(tokenPath, []byte(token), 0600); err != nil {
				return nil, fmt.Errorf("failed to write API token to %s: %w", tokenPath, err)
			}

			fmt.Printf("Generated new API token: %s\n", token)
			fmt.Printf("Token saved to: %s\n", tokenPath)
		}
	}

	return cfg, nil
}

// GetAPIToken returns the current API token.
func (c *Config) GetAPIToken() string {
	return c.apiToken
}

// HTTPListenAddr returns the address:port string for the HTTP server.
func (c *Config) HTTPListenAddr() string {
	return net.JoinHostPort(c.HTTPAddress, strconv.Itoa(c.HTTPPort))
}

// TFTPListenAddr returns the address:port string for the TFTP server.
func (c *Config) TFTPListenAddr() string {
	return net.JoinHostPort(c.TFTPAddress, strconv.Itoa(c.TFTPPort))
}

// DHCPListenAddr returns the address:port string for the DHCP proxy.
func (c *Config) DHCPListenAddr() string {
	return net.JoinHostPort(c.DHCPProxyAddress, strconv.Itoa(c.DHCPProxyPort))
}

// applyEnvOverrides checks environment variables and applies them to the config.
func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("LIGHTBOOT_HTTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.HTTPPort = port
		}
	}
	if v := os.Getenv("LIGHTBOOT_HTTP_ADDRESS"); v != "" {
		c.HTTPAddress = v
	}
	if v := os.Getenv("LIGHTBOOT_TFTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.TFTPPort = port
		}
	}
	if v := os.Getenv("LIGHTBOOT_ISO_DIR"); v != "" {
		c.ISODir = v
	}
	if v := os.Getenv("LIGHTBOOT_CACHE_DIR"); v != "" {
		c.CacheDir = v
	}
	if v := os.Getenv("LIGHTBOOT_DATA_DIR"); v != "" {
		c.DataDir = v
	}
	if v := os.Getenv("LIGHTBOOT_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := os.Getenv("LIGHTBOOT_LOG_FILE"); v != "" {
		c.LogFile = v
	}
}

// generateAPIToken creates a 64-character hex-encoded random token.
func generateAPIToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
