package service

import (
	"testing"

	"quickdesk/signaling/internal/models"
)

// gorm.io/driver/postgres's Migrator.GetRows injects
// pgx.QueryExecModeSimpleProtocol as a query argument when PrepareStmt is off.
// On a table that already exists, AutoMigrate's schema comparison runs
// `SELECT * FROM "t" LIMIT 1` with that argument attached, and pgx's sanitizer
// rejects it with "insufficient arguments" because the statement has no
// placeholder for it to fill.
//
// This is the failure that kept the SignalingServer verify job red: CI sets
// QD_TEST_DATABASE_URL and migrates the same scratch database over and over,
// so every run reaches the "table exists" branch, while a developer without
// the variable set has all of these tests skipped.
func TestMigrateTestSchemaSurvivesAnExistingTable(t *testing.T) {
	db := openTestPostgres(t)

	// Drop and recreate once so the first migration sees a missing table, then
	// migrate the same table again to reach the schema-comparison path.
	if err := db.Migrator().DropTable(&models.AuditLog{}); err != nil {
		t.Fatalf("drop: %v", err)
	}
	migrateTestSchema(t, db, &models.AuditLog{})
	migrateTestSchema(t, db, &models.AuditLog{})
}
