package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

// Config holds runtime configuration for the application.
type Config struct {
	ServerPort          string
	DatabaseDSN         string
	DataDir             string
	DefaultSubnet       string
	EnableAutoMigrate   bool
	APIBaseURL          string
	NebulaVersion       string
	NebulaDownloadBase  string
	NebulaProxyPrefix   string
	FrontendDir         string
	AdminUsername       string
	AdminPassword       string
	SessionSecret       string
	SessionSecureCookie bool
	StaticAccessToken   string
}

var (
	cfg  *Config
	once sync.Once
)

// Load returns a singleton configuration loaded from environment variables.
func Load() *Config {
	once.Do(func() {
		_ = godotenv.Load()

		dataDir := os.Getenv("NEBULA_DATA_DIR")
		if dataDir == "" {
			dataDir = "data"
		}
		// Ensure the data directory exists so file operations do not fail later.
		if err := os.MkdirAll(dataDir, 0o755); err != nil {
			log.Fatalf("failed to create data dir %s: %v", dataDir, err)
		}

		serverPort := fallback(os.Getenv("NEBULA_SERVER_PORT"), "8080")
		apiBase := os.Getenv("NEBULA_API_BASE")
		if apiBase == "" {
			if detected, err := detectAPIBase(serverPort); err == nil {
				apiBase = detected
			} else {
				apiBase = fmt.Sprintf("http://localhost:%s", serverPort)
			}
		}

		cfg = &Config{
			ServerPort:          serverPort,
			DatabaseDSN:         fallback(os.Getenv("NEBULA_MYSQL_DSN"), "root:123150.wangzai7@tcp(47.109.89.95:23006)/nebula_manager?charset=utf8mb4&parseTime=True&loc=Local"),
			DataDir:             filepath.Clean(dataDir),
			DefaultSubnet:       os.Getenv("NEBULA_DEFAULT_SUBNET"),
			EnableAutoMigrate:   os.Getenv("NEBULA_AUTO_MIGRATE") != "false",
			APIBaseURL:          apiBase,
			NebulaVersion:       fallback(os.Getenv("NEBULA_BINARY_VERSION"), "1.9.3"),
			NebulaDownloadBase:  fallback(os.Getenv("NEBULA_BINARY_BASE"), "https://github.com/slackhq/nebula/releases/download"),
			NebulaProxyPrefix:   os.Getenv("NEBULA_BINARY_PROXY_PREFIX"),
			FrontendDir:         fallback(os.Getenv("NEBULA_FRONTEND_DIR"), filepath.Join("frontend", "dist")),
			AdminUsername:       fallback(os.Getenv("NEBULA_ADMIN_USERNAME"), "admin"),
			AdminPassword:       fallback(os.Getenv("NEBULA_ADMIN_PASSWORD"), "admin123"),
			SessionSecret:       fallback(os.Getenv("NEBULA_SESSION_SECRET"), randomSecret()),
			SessionSecureCookie: boolFromEnv(os.Getenv("NEBULA_SESSION_SECURE")),
			StaticAccessToken:   os.Getenv("NEBULA_STATIC_TOKEN"),
		}
	})
	return cfg
}

func fallback(value, defaultVal string) string {
	if value == "" {
		return defaultVal
	}
	return value
}

func boolFromEnv(val string) bool {
	if val == "" {
		return false
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return false
	}
	return parsed
}

func randomSecret() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "nebula-session-secret"
	}
	return hex.EncodeToString(buf)
}

func detectAPIBase(port string) (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return "", fmt.Errorf("unexpected local address type")
	}
	ip := localAddr.IP.String()
	if ip == "" {
		return "", fmt.Errorf("empty ip detected")
	}
	return fmt.Sprintf("http://%s:%s", ip, port), nil
}
