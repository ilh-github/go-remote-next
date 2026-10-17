package database

import (
	"fmt"

	"quickdesk/signaling/internal/config"

	"gorm.io/gorm"
)

// OpenDatabase selects the configured database backend.
func OpenDatabase(cfg *config.Config) (*gorm.DB, error) {
	driver := cfg.Database.Driver
	if driver == "" {
		driver = "postgres"
	}
	switch driver {
	case "postgres", "postgresql":
		return OpenPostgreSQL(cfg)
	case "sqlite":
		return InitSQLite(cfg)
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q", cfg.Database.Driver)
	}
}
