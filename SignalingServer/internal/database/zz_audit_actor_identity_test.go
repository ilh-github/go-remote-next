package database

import (
	"context"
	"path/filepath"
	"testing"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/models"
)

// The partial unique index behind `LogActorOnce` is Postgres DDL. SQLite cannot
// express a partial index at all, so minimal mode has to degrade rather than
// fail: the ensure call must report "not in place" and leave the database usable,
// because a self-hosted install that refuses to boot over a missing audit index
// is worse than one whose audit trail can duplicate a row under a race.
func TestEnsureAuditActorIdentityOnSQLiteReportsNotInPlace(t *testing.T) {
	db, err := InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "ensure-index.db"),
		},
	})
	if err != nil {
		t.Fatalf("InitSQLite(): %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if inPlace := EnsureAuditActorIdentity(context.Background(), db); inPlace {
		t.Fatal("EnsureAuditActorIdentity reported the index in place on SQLite; the Postgres DDL cannot have applied")
	}

	// And the handle still works, which is the point of degrading.
	if err := db.Exec("CREATE TABLE IF NOT EXISTS probe (id INTEGER PRIMARY KEY)").Error; err != nil {
		t.Fatalf("the database is unusable after the failed ensure: %v", err)
	}
}

// The Postgres form cannot be created on SQLite, so minimal mode gets its own
// statement. This proves that one actually enforces the guarantee rather than
// merely existing: a second row for the same actor identity is refused, and two
// admin rows for the same action are not.
func TestSQLiteAuditActorIdentityEnforcesOneRowPerTransfer(t *testing.T) {
	db, err := InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "audit-identity.db"),
		},
	})
	if err != nil {
		t.Fatalf("InitSQLite(): %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&models.AuditLog{}); err != nil {
		t.Fatalf("AutoMigrate(): %v", err)
	}
	if !EnsureAuditActorIdentitySQLite(context.Background(), db) {
		t.Fatal("the SQLite form of the index could not be created")
	}

	seed := func(actorUserID, adminID uint, resourceID string) error {
		return db.Create(&models.AuditLog{
			ActorUserID:  actorUserID,
			AdminID:      adminID,
			Action:       "file.transfer",
			ResourceType: "file_transfer",
			ResourceID:   resourceID,
		}).Error
	}

	if err := seed(4242, 0, "tr-1"); err != nil {
		t.Fatalf("first report: %v", err)
	}
	// The whole point: the same transfer id from the same actor is refused.
	if err := seed(4242, 0, "tr-1"); err == nil {
		t.Fatal("a second row for the same transfer id was accepted on SQLite")
	}
	// A different actor is a different row, which is the other half of the
	// guarantee - the index must not collapse distinct actors.
	if err := seed(4243, 0, "tr-1"); err != nil {
		t.Fatalf("a different actor was refused: %v", err)
	}
	// Admin rows carry actor_user_id = 0, so the partial index does not cover them
	// at all - which is the point: an admin legitimately repeats the same action
	// on the same resource, and a unique constraint there would drop rows.
	if err := seed(0, 11, "tr-admin"); err != nil {
		t.Fatalf("first admin row: %v", err)
	}
	if err := seed(0, 12, "tr-admin"); err != nil {
		t.Fatalf("a second admin's row was refused: %v", err)
	}

	// A row that names no resource has no stable identity to be unique on, and
	// several genuinely distinct events carry an empty resource_id. The retention
	// tests seed exactly that shape, which is why the index excludes it.
	if err := seed(4242, 0, ""); err != nil {
		t.Fatalf("first resource-less row: %v", err)
	}
	if err := seed(4242, 0, ""); err != nil {
		t.Fatalf("a second resource-less row was refused: %v", err)
	}
}
