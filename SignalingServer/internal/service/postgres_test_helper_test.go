package service

import (
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// openTestPostgres opens the scratch Postgres that QD_TEST_DATABASE_URL points
// at, skipping the test when the variable is unset.
//
// The pool is closed with the test. The scratch database is shared by every
// Postgres-backed test in this package, and Postgres has a small connection
// limit, so a handle left idle after its test finishes is a connection the
// tests still to run cannot have.
func openTestPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the Postgres-backed tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

// migrateTestSchema runs AutoMigrate through a session that has PrepareStmt on.
//
// gorm.io/driver/postgres's Migrator.GetRows injects
// pgx.QueryExecModeSimpleProtocol as a query argument whenever PrepareStmt is
// off, and it does so for `SELECT * FROM "t" LIMIT 1`, a statement with no
// placeholder for the argument to fill. pgx's sanitizer rejects that with
// "insufficient arguments", so migrating a table that already exists fails.
// This is what kept the SignalingServer verify job red: CI migrates the same
// scratch database run after run, so it always reaches the "table exists"
// branch, while a developer without the variable set has every one of these
// tests skipped.
//
// Turning PrepareStmt on for the whole handle would also fix it, but a prepared
// session refuses a multi-statement Exec, and several tests replay
// migrations/001_init.sql verbatim. So the flag belongs on the migration
// session alone and the caller keeps a plain handle for everything else.
func migrateTestSchema(t *testing.T, db *gorm.DB, models ...interface{}) {
	t.Helper()
	if err := db.Session(&gorm.Session{PrepareStmt: true}).AutoMigrate(models...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}

// applyTestMigration replays migrations/001_init.sql, which the file itself
// calls the source of truth for column types, indexes and defaults.
//
// Prefer this over migrateTestSchema for a table the SQL file already defines.
// GORM names a uniqueIndex uni_<table>_<column> while the SQL file leaves
// Postgres to name it, so AutoMigrate on a table the SQL file created spends
// every run trying to drop an index that was never there.
func applyTestMigration(t *testing.T, db *gorm.DB) {
	t.Helper()
	migration, err := os.ReadFile("../../migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if err := db.Exec(string(migration)).Error; err != nil {
		t.Fatalf("migrate: %v", err)
	}
}
