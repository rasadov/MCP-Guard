package database

import (
	"fmt"
	"log/slog"

	"github.com/rasadov/mcp-guard/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	if err := models.AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	slog.Info("database connected and migrated")
	return db, nil
}
