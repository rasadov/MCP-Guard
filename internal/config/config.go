package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr         string
	DatabaseURL  string
	JWTSecret    string
	JWTExpiry    time.Duration
	AuthDevMode  bool
	Slack        SlackConfig
	SlackMCPPath string
	SeedOnStart  bool
}

type SlackConfig struct {
	BotToken string
}

func Load() Config {
	jwtHours := getEnvInt("JWT_EXPIRY_HOURS", 24)
	return Config{
		Addr:         getEnv("ADDR", ":8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://mcpguard:mcpguard@localhost:5432/mcpguard?sslmode=disable"),
		JWTSecret:    getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		JWTExpiry:    time.Duration(jwtHours) * time.Hour,
		AuthDevMode:  getEnvBool("AUTH_DEV_MODE", true),
		Slack:        SlackConfig{BotToken: os.Getenv("SLACK_BOT_TOKEN")},
		SlackMCPPath: getEnv("SLACK_MCP_PATH", "slack-mcp-server"),
		SeedOnStart:  getEnvBool("SEED_ON_START", true),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
