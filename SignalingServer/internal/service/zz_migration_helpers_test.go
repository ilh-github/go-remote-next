package service

import (
	"testing"

	"quickdesk/signaling/internal/models"
)

// The scratch Postgres database is shared by every Postgres-backed test in this
// package and nothing resets it between runs, so any helper that builds a schema
// must survive tables a previous run already created.
//
// `migrateTestSchema` exists for exactly this: gorm.io/driver/postgres's
// Migrator.GetRows injects pgx.QueryExecModeSimpleProtocol as a query argument
// when PrepareStmt is off, and on an existing table AutoMigrate's schema
// comparison runs `SELECT * FROM "t" LIMIT 1` with that argument attached --
// pgx rejects it with "insufficient arguments". A bare `db.AutoMigrate` is
// therefore only safe the first time a table is seen.
//
// The symptom this pins: the package is green on a fresh database and red on a
// second run, or red as soon as some other test created `users` first. Two of
// the package's own tests failed that way, while passing in isolation, because
// the shared database already held the tables they tried to migrate.
func TestScratchSchemaSurvivesTablesAnEarlierRunCreated(t *testing.T) {
	db := openTestPostgres(t)

	// The tables that only GORM defines. Migrating them twice is the second-run
	// path that breaks without PrepareStmt, and it must stay safe.
	migrateTestSchema(t, db, &models.AuditLog{})
	migrateTestSchema(t, db, &models.AuditLog{})

	// The tables the SQL file defines, replayed after other tests already made
	// them. This is the pair that failed: `session_recordings` and
	// `device_access_logs` migrated through AutoMigrate on a database where
	// `users` already existed.
	applyTestMigration(t, db)
	applyTestMigration(t, db)
}

// A table the SQL file defines must never reach AutoMigrate: GORM names a
// unique index uni_<table>_<column> while the file writes plain `UNIQUE` and
// lets Postgres name it, so AutoMigrate tries to drop a constraint that was
// never there. Pinning that failure keeps the two helpers honest about which
// tables belong to which.
func TestAutoMigrateOnASQLDefinedTableFails(t *testing.T) {
	db := openTestPostgres(t)
	applyTestMigration(t, db)

	err := db.AutoMigrate(&models.User{}).Error
	if err == nil {
		t.Fatal("AutoMigrate on a table migrations/001_init.sql created succeeded; the index-naming mismatch the helpers avoid may be gone")
	}
}

// A bare AutoMigrate is what fails, so state that directly: on a table that
// already exists it errors. This documents why the helpers above route through
// migrateTestSchema instead, and it goes red if the underlying driver behaviour
// that motivated the helper ever changes.
func TestBareAutoMigrateFailsOnAnExistingTable(t *testing.T) {
	db := openTestPostgres(t)
	applyTestMigration(t, db)

	err := db.AutoMigrate(&models.User{}).Error
	if err == nil {
		t.Fatal("bare AutoMigrate on an existing table succeeded; the PrepareStmt workaround in migrateTestSchema may no longer be needed")
	}
}
