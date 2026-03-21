package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	TelegramAppID   int
	TelegramAppHash string
	AnthropicAPIKey string
	SessionDir      string
}

// DefaultDir returns the default telgo data directory (~/.telgo).
func DefaultDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".telgo")
}

// loadEnvFile reads KEY=VALUE pairs from path and sets them as environment
// variables, skipping any key that is already set. Silent no-op if missing.
func loadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k != "" && os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
}

func Load() (*Config, error) {
	// Auto-load ~/.telgo/.env so users only need to run 'telgo setup' once.
	// Explicit env vars take priority over the file.
	sessionDir := os.Getenv("TELGO_SESSION_DIR")
	if sessionDir == "" {
		sessionDir = DefaultDir()
	}
	loadEnvFile(filepath.Join(sessionDir, ".env"))

	appIDStr := os.Getenv("TELEGRAM_APP_ID")
	if appIDStr == "" {
		return nil, fmt.Errorf("TELEGRAM_APP_ID is required (get it from https://my.telegram.org)")
	}
	appID, err := strconv.Atoi(appIDStr)
	if err != nil {
		return nil, fmt.Errorf("TELEGRAM_APP_ID must be an integer: %w", err)
	}

	appHash := os.Getenv("TELEGRAM_APP_HASH")
	if appHash == "" {
		return nil, fmt.Errorf("TELEGRAM_APP_HASH is required (get it from https://my.telegram.org)")
	}

	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")

	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return nil, fmt.Errorf("cannot create session directory: %w", err)
	}

	return &Config{
		TelegramAppID:   appID,
		TelegramAppHash: appHash,
		AnthropicAPIKey: anthropicKey,
		SessionDir:      sessionDir,
	}, nil
}
