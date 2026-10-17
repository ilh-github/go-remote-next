package database

import (
	"path/filepath"
	"testing"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/models"
)

func TestInitSQLiteCreatesPersistentWALDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "quickdesk.db")
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   path,
		},
	}

	db, err := InitSQLite(cfg)
	if err != nil {
		t.Fatalf("InitSQLite() error = %v", err)
	}
	if got := db.Dialector.Name(); got != "sqlite" {
		t.Fatalf("dialector = %q, want sqlite", got)
	}

	var journalMode string
	if err := db.Raw("PRAGMA journal_mode").Scan(&journalMode).Error; err != nil {
		t.Fatalf("read journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}

	if err := db.AutoMigrate(&models.User{}); err != nil {
		t.Fatalf("AutoMigrate(User): %v", err)
	}
	user := models.User{Username: "sqlite-persist", Password: "test-hash"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close first SQLite connection: %v", err)
	}

	reopened, err := InitSQLite(cfg)
	if err != nil {
		t.Fatalf("reopen InitSQLite() error = %v", err)
	}
	reopenedSQL, err := reopened.DB()
	if err != nil {
		t.Fatalf("reopened.DB(): %v", err)
	}
	t.Cleanup(func() { _ = reopenedSQL.Close() })

	var got models.User
	if err := reopened.Where("username = ?", user.Username).First(&got).Error; err != nil {
		t.Fatalf("read persisted user: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("persisted user id = %d, want %d", got.ID, user.ID)
	}
}

func TestOpenDatabaseSelectsSQLiteDriver(t *testing.T) {
	path := filepath.Join(t.TempDir(), "selected.db")
	db, err := OpenDatabase(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   path,
		},
	})
	if err != nil {
		t.Fatalf("OpenDatabase() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if got := db.Dialector.Name(); got != "sqlite" {
		t.Fatalf("dialector = %q, want sqlite", got)
	}
}

func TestSQLiteMigratesDevicesAndFavoritesWithDistinctIndexes(t *testing.T) {
	db, err := InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "indexes.db"),
		},
	})
	if err != nil {
		t.Fatalf("InitSQLite() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := db.AutoMigrate(
		&models.User{},
		&models.UserDevice{},
		&models.UserFavorite{},
	); err != nil {
		t.Fatalf("AutoMigrate devices and favorites: %v", err)
	}

	var indexes []string
	if err := db.Raw(
		"SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name IN ('user_devices', 'user_favorites')",
	).Scan(&indexes).Error; err != nil {
		t.Fatalf("read SQLite indexes: %v", err)
	}
	seen := make(map[string]bool, len(indexes))
	for _, name := range indexes {
		seen[name] = true
	}
	for _, want := range []string{"idx_user_device", "idx_user_favorite_user_device"} {
		if !seen[want] {
			t.Fatalf("index %q missing; got %v", want, indexes)
		}
	}
}
