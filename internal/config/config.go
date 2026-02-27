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
	OwnerSharedSecret string
	MCPBaseURL        string
	ListenAddr        string
	Account           AccountConfig
}

type AccountConfig struct {
	Phone       string
	Password    string
	CountryCode string
	LoginMethod string
}

type fileConfig struct {
	Account struct {
		Phone       string `json:"phone"`
		Password    string `json:"password"`
		CountryCode string `json:"country_code"`
		LoginMethod string `json:"login_method"`
	} `json:"account"`
	Owner struct {
		UserID       string `json:"user_id"`
		SharedSecret string `json:"shared_secret"`
	} `json:"owner"`
	Operator struct {
		ListenAddr string `json:"listen_addr"`
	} `json:"operator"`
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
		OwnerUserID:       strings.TrimSpace(fc.Owner.UserID),
		OwnerSharedSecret: strings.TrimSpace(fc.Owner.SharedSecret),
		MCPBaseURL:        strings.TrimRight(strings.TrimSpace(fc.MCP.BaseURL), "/"),
		ListenAddr:        strings.TrimSpace(fc.Operator.ListenAddr),
		Account: AccountConfig{
			Phone:       strings.TrimSpace(fc.Account.Phone),
			Password:    strings.TrimSpace(fc.Account.Password),
			CountryCode: strings.TrimSpace(fc.Account.CountryCode),
			LoginMethod: strings.TrimSpace(fc.Account.LoginMethod),
		},
	}

	if cfg.MCPBaseURL == "" {
		cfg.MCPBaseURL = "http://127.0.0.1:18060"
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8081"
	}
	if cfg.OwnerUserID == "" {
		return nil, errors.New("owner.user_id is required")
	}
	if cfg.OwnerSharedSecret == "" {
		return nil, errors.New("owner.shared_secret is required")
	}
	return cfg, nil
}

// osOpen is separated for testability.
var osOpen = func(path string) (io.ReadCloser, error) {
	return os.Open(path)
}
