package repository

import (
	"context"
	"path/filepath"
	"testing"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"
)

func TestUpdateLastLoginWritesPortableTimestamp(t *testing.T) {
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "admin-login.db"),
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
	if err := db.AutoMigrate(&models.AdminUser{}); err != nil {
		t.Fatalf("AutoMigrate(AdminUser): %v", err)
	}
	admin := models.AdminUser{
		Username: "minimal-admin",
		Password: "hash",
		Role:     "super_admin",
		Status:   true,
	}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("create admin: %v", err)
	}

	repo := NewAdminUserRepository(db)
	if err := repo.UpdateLastLogin(context.Background(), admin.ID); err != nil {
		t.Fatalf("UpdateLastLogin(): %v", err)
	}
	var got models.AdminUser
	if err := db.First(&got, admin.ID).Error; err != nil {
		t.Fatalf("reload admin: %v", err)
	}
	if got.LastLogin.IsZero() {
		t.Fatal("last_login is zero after UpdateLastLogin")
	}
}
