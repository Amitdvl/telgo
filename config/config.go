package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	TelegramAppID   int
	TelegramAppHash string
	AnthropicAPIKey string
	SessionDir      string
}

func Load() (*Config, error) {
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

	sessionDir := os.Getenv("TELGO_SESSION_DIR")
	if sessionDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		sessionDir = filepath.Join(home, ".telgo")
	}

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
