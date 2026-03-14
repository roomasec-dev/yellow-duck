package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"rm_ai_agent/internal/app"
	"rm_ai_agent/internal/config"
)

func main() {
	defaultConfigPath := resolveDefaultConfigPath()
	configPath := flag.String("config", defaultConfigPath, "path to TOML config")
	flag.Parse()
	log.Printf("using config: %s", *configPath)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("load config failed: %v", err)
		os.Exit(1)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Printf("build app failed: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := application.Close(); err != nil {
			log.Printf("close app failed: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := application.Run(ctx); err != nil {
		log.Printf("app stopped with error: %v", err)
		os.Exit(1)
	}
}

func resolveDefaultConfigPath() string {
	preferred := filepath.Clean("configs/config.local.toml")
	if _, err := os.Stat(preferred); err == nil {
		return preferred
	}
	return filepath.Clean("configs/config.example.toml")
}
