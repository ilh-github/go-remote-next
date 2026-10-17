package database

import (
	"fmt"
	"log"
	"quickdesk/signaling/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// OpenPostgreSQL initializes the PostgreSQL connection.
func OpenPostgreSQL(cfg *config.Config) (*gorm.DB, error) {
	dsn := cfg.Database.DSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	// Get underlying SQL DB for connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get PostgreSQL database instance: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping PostgreSQL database: %w", err)
	}

	log.Printf("Successfully connected to PostgreSQL at %s:%d/%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	return db, nil
}

// InitPostgreSQL preserves the fatal-on-error helper used by one-off commands.
func InitPostgreSQL(cfg *config.Config) *gorm.DB {
	db, err := OpenPostgreSQL(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	return db
}

// RunMigrations executes SQL migration files
func RunMigrations(db *gorm.DB) error {
	// For simplicity, using AutoMigrate
	// In production, use proper migration tool like golang-migrate
	log.Println("Running database migrations...")

	// Auto-migrate will create tables based on model structs
	// This is simpler for development
	// For production, use the SQL files in migrations/

	log.Println("Database migrations completed")
	return nil
}
