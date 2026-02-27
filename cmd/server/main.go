package main

import (
	"log"
	"net/http"
	"time"

	"github.com/lihuss/xiaohongshu-ai-pet-operator/internal/config"
	"github.com/lihuss/xiaohongshu-ai-pet-operator/internal/owner"
	"github.com/lihuss/xiaohongshu-ai-pet-operator/internal/security"
	"github.com/lihuss/xiaohongshu-ai-pet-operator/internal/server"
	"github.com/lihuss/xiaohongshu-ai-pet-operator/internal/xhs"
)

func main() {
	cfg, err := config.Load("config/user.config.json")
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	nonceStore := security.NewNonceStore(10 * time.Minute)
	guard := owner.NewGuard(cfg.OwnerUserID)
	client := xhs.NewClient(cfg.MCPBaseURL, 20*time.Second)
	h := server.NewHandler(cfg, guard, nonceStore, client)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           h.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("xhs ai pet operator listening on %s", cfg.ListenAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
