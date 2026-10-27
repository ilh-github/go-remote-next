package service

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// sessionRevokeProbeSchema is a private schema for this test. The SQL file uses
// CREATE TABLE IF NOT EXISTS without a schema qualifier, so replaying it in
// `public` would build every table in the shared scratch database - and the next
// test to migrate there would fail on tables that already exist. A schema of its
// own keeps the replay equivalent (same DDL, same types, same indexes) while
// leaving `public` exactly as this test found it.
const sessionRevokeProbeSchema = "session_revoke_probe"

// The SQL file calls itself the schema source of truth for Postgres, and
// AutoMigrate is what a fresh deployment actually gets. A table that exists in
// one and not the other is a deployment that works until the day it does not, so
// the two are checked against each other for the table this change adds.
func TestSessionRevokeCommandSchemaMatchesTheSQLMigration(t *testing.T) {
	db := openTestPostgres(t)

	if err := db.Exec("CREATE SCHEMA IF NOT EXISTS " + sessionRevokeProbeSchema).Error; err != nil {
		t.Fatalf("create probe schema: %v", err)
	}
	t.Cleanup(func() {
		// search_path is per-connection, and the pooled handle is shared with
		// every other test in this package - so it is put back, not just the
		// schema dropped.
		_ = db.Exec("SET search_path TO public").Error
		_ = db.Exec("DROP SCHEMA IF EXISTS " + sessionRevokeProbeSchema + " CASCADE").Error
	})
	if err := db.Exec("SET search_path TO " + sessionRevokeProbeSchema).Error; err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	// The statement that creates this table, taken from the migration file rather
	// than retyped here, so the test cannot drift from what a fresh install runs.
	// The whole file is not replayed: it needs extensions that live at database
	// level, and the point of this test is one table's columns, not the rest of
	// the schema.
	ddl := sessionRevokeCommandDDL(t) + ";\n"
	if err := db.Exec(ddl).Error; err != nil {
		t.Fatalf("create the table from the migration DDL: %v", err)
	}

	// The model's columns must all exist after the SQL replay, or a deployment
	// that migrates from the file would be missing a column the code writes.
	//
	// PrepareStmt rather than a bare AutoMigrate: the postgres driver's migrator
	// injects a query argument into a statement that has no placeholder for it
	// unless that flag is on, which fails on a table the SQL file already
	// created. See postgres_test_helper_test.go.
	if err := db.Session(&gorm.Session{PrepareStmt: true}).
		AutoMigrate(&models.SessionRevokeCommand{}); err != nil {
		t.Fatalf("AutoMigrate on top of the SQL schema: %v", err)
	}

	now := time.Now().UTC()
	row := models.SessionRevokeCommand{
		SessionID:  "mig-probe-session",
		InstanceID: "mig-probe-instance",
		Reason:     models.SessionRevokeReasonOwner,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("insert into the migrated table: %v", err)
	}
	if row.ID == 0 {
		t.Fatal("the insert produced no id")
	}

	store := NewSessionRevokeCommandStore(db)
	ctx := context.Background()
	if err := store.Claim(ctx, row.ID, time.Minute); err != nil {
		t.Fatalf("Claim(): %v", err)
	}
	var reloaded models.SessionRevokeCommand
	if err := db.Where("session_id = ?", "mig-probe-session").First(&reloaded).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Attempts != 1 || reloaded.ClaimExpiresAt == nil {
		t.Fatalf("claim columns did not persist: %+v", reloaded)
	}
	if err := store.MarkDelivered(ctx, row.ID); err != nil {
		t.Fatalf("MarkDelivered(): %v", err)
	}
	if err := db.Where("session_id = ?", "mig-probe-session").First(&reloaded).Error; err != nil {
		t.Fatalf("reload after delivery: %v", err)
	}
	if !reloaded.Delivered || reloaded.DeliveredAt == nil {
		t.Fatalf("delivery columns did not persist: %+v", reloaded)
	}
}

// sessionRevokeCommandDDL extracts the CREATE TABLE and its indexes from the
// migration file, so the assertion is about the DDL that actually ships.
// The statements are matched by name and taken verbatim, so a change to the
// migration file shows up here rather than being silently retyped.
func sessionRevokeCommandDDL(t *testing.T) string {
	t.Helper()
	migration, err := os.ReadFile("../../migrations/001_init.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	lines := strings.Split(string(migration), "\n")
	var kept []string
	// A statement is accumulated whole before it is judged: an index names its
	// table on the line *after* the CREATE, so matching the opening line alone
	// would drop every index for this table.
	var statement []string
	flush := func() {
		block := strings.Join(statement, "\n")
		if strings.Contains(block, "session_revoke_commands") {
			kept = append(kept, block)
		}
		statement = nil
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		upper := strings.ToUpper(trimmed)
		startsStatement := strings.HasPrefix(upper, "CREATE TABLE") ||
			strings.HasPrefix(upper, "CREATE UNIQUE INDEX") ||
			strings.HasPrefix(upper, "CREATE INDEX")
		if startsStatement {
			flush()
		}
		statement = append(statement, line)
		if strings.HasSuffix(trimmed, ";") {
			flush()
		}
	}
	flush()
	if len(kept) == 0 {
		t.Fatal("the migration file no longer creates session_revoke_commands")
	}
	return strings.Join(kept, "\n")
}
