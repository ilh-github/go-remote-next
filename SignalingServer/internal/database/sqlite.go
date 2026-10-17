package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"quickdesk/signaling/internal/config"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitSQLite opens the single-file database used by the minimal self-hosted
// mode. A single pooled connection avoids cross-connection write contention
// while WAL keeps readers from blocking the writer.
func InitSQLite(cfg *config.Config) (*gorm.DB, error) {
	path := strings.TrimSpace(cfg.Database.Path)
	if path == "" {
		return nil, fmt.Errorf("DB_PATH is required when DB_DRIVER=sqlite")
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve SQLite path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o750); err != nil {
		return nil, fmt.Errorf("create SQLite directory: %w", err)
	}

	dsn := "file:" + filepath.ToSlash(absolutePath) +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get SQLite database instance: %w", err)
	}
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(1)
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping SQLite database: %w", err)
	}

	log.Printf("Successfully connected to SQLite at %s", absolutePath)
	return db, nil
}
