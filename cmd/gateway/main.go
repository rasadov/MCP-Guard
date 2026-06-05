package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/rasadov/mcp-guard/internal/config"
	"github.com/rasadov/mcp-guard/internal/database"
	mcpgw "github.com/rasadov/mcp-guard/internal/mcp"
	"github.com/rasadov/mcp-guard/internal/seed"
	"github.com/rasadov/mcp-guard/internal/server"
	webassets "github.com/rasadov/mcp-guard/web"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

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

	downstream, err := mcpgw.ConnectSlack(ctx, cfg)
	if err != nil {
		slog.Warn("slack downstream unavailable", "error", err)
		downstream = &mcpgw.Downstream{}
	}
	defer downstream.Close()

	srv := server.New(cfg, db, webassets.Dist, downstream)
	if err := srv.Run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
