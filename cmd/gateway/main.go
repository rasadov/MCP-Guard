package main

import (
	"log/slog"
	"os"

	"github.com/rasadov/mcp-guard/internal/config"
	"github.com/rasadov/mcp-guard/internal/database"
	"github.com/rasadov/mcp-guard/internal/seed"
	"github.com/rasadov/mcp-guard/internal/server"
	webassets "github.com/rasadov/mcp-guard/web"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}

	if cfg.SeedOnStart {
		if err := seed.Run(db); err != nil {
			slog.Error("seed failed", "error", err)
			os.Exit(1)
		}
	}

	srv := server.New(cfg, db, webassets.Dist)
	if err := srv.Run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
