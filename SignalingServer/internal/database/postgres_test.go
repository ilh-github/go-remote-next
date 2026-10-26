package database

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/models"
)

// The server opens its Postgres handle in one place and then migrates every
// model through it, so the handle has to survive migrating a table that is
// already there. gorm.io/driver/postgres's Migrator.GetRows injects
// pgx.QueryExecModeSimpleProtocol as a query argument when PrepareStmt is off,
// and it does that for `SELECT * FROM "t" LIMIT 1`, a statement with no
// placeholder. pgx's sanitizer rejects the query with "insufficient
// arguments", so a table that exists cannot be migrated.
//
// That failure is invisible in production: cmd/signaling migrates one model at
// a time and only logs a warning per failure, so an existing installation
// silently stops receiving schema updates and a fresh one never finishes. It
// reached the conformance smoke as "relation devices does not exist".
func TestOpenPostgreSQLMigratesAnExistingTable(t *testing.T) {
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the Postgres tests")
	}
	db, err := OpenPostgreSQL(&config.Config{
		Database: databaseConfigFromDSN(t, dsn),
	})
	if err != nil {
		t.Fatalf("OpenPostgreSQL: %v", err)
	}
	if db == nil {
		t.Fatal("OpenPostgreSQL returned nil")
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	if err := db.Migrator().DropTable(&models.Device{}); err != nil {
		t.Fatalf("drop: %v", err)
	}
	// A table that does not exist yet goes straight to CREATE TABLE, which is
	// why a fresh database migrates fine. The second call is the one that has to
	// compare schemas, and that is the path that fails.
	if err := AutoMigrate(db, &models.Device{}); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := AutoMigrate(db, &models.Device{}); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

// databaseConfigFromDSN splits the key=value DSN the CI hands us into the fields
// OpenPostgreSQL builds its own DSN from, so the test drives the real
// configuration path rather than a hand-written connection string.
func databaseConfigFromDSN(t *testing.T, dsn string) config.DatabaseConfig {
	t.Helper()
	cfg := config.DatabaseConfig{Driver: "postgres"}
	for _, field := range strings.Fields(dsn) {
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		switch key {
		case "host":
			cfg.Host = value
		case "port":
			port, err := strconv.Atoi(value)
			if err != nil {
				t.Fatalf("parse port %q: %v", value, err)
			}
			cfg.Port = port
		case "user":
			cfg.User = value
		case "password":
			cfg.Password = value
		case "dbname":
			cfg.DBName = value
		}
	}
	return cfg
}
