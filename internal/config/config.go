package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type Config struct {
	OwnerUserID       string
	MCPBaseURL        string
}

type fileConfig struct {
	Owner struct {
		UserID string `json:"user_id"`
	} `json:"owner"`
	MCP struct {
		BaseURL string `json:"base_url"`
	} `json:"mcp"`
}

func Load(path string) (*Config, error) {
	f, err := osOpen(path)
	if err != nil {
		return nil, fmt.Errorf("open config file failed: %w", err)
	}
	defer f.Close()

	raw, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read config file failed: %w", err)
	}

	var fc fileConfig
	if err := json.Unmarshal(raw, &fc); err != nil {
		return nil, fmt.Errorf("parse config json failed: %w", err)
	}

	cfg := &Config{
		OwnerUserID: strings.TrimSpace(fc.Owner.UserID),
		MCPBaseURL:  strings.TrimRight(strings.TrimSpace(fc.MCP.BaseURL), "/"),
	}

	if cfg.MCPBaseURL == "" {
		cfg.MCPBaseURL = "http://127.0.0.1:18060"
	}
	if cfg.OwnerUserID == "" {
		return nil, errors.New("owner.user_id is required (must be the owner account user_id, not the pet account)")
	}
	return cfg, nil
}

// osOpen is separated for testability.
var osOpen = func(path string) (io.ReadCloser, error) {
	return os.Open(path)
}
