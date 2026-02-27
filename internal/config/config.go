package config

import (
	"errors"
	"os"
	"strings"
)

type Config struct {
	OwnerUserID      string
	OwnerSharedSecret string
	MCPBaseURL       string
	ListenAddr       string
}

func Load() (*Config, error) {
	cfg := &Config{
		OwnerUserID:       strings.TrimSpace(os.Getenv("OWNER_USER_ID")),
		OwnerSharedSecret: strings.TrimSpace(os.Getenv("OWNER_SHARED_SECRET")),
		MCPBaseURL:        strings.TrimRight(strings.TrimSpace(os.Getenv("MCP_BASE_URL")), "/"),
		ListenAddr:        strings.TrimSpace(os.Getenv("LISTEN_ADDR")),
	}
	if cfg.MCPBaseURL == "" {
		cfg.MCPBaseURL = "http://127.0.0.1:18060"
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8081"
	}
	if cfg.OwnerUserID == "" {
		return nil, errors.New("OWNER_USER_ID is required")
	}
	if cfg.OwnerSharedSecret == "" {
		return nil, errors.New("OWNER_SHARED_SECRET is required")
	}
	return cfg, nil
}

