package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestNotificationClaimWorksOnSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "outbox.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.NotificationDelivery{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	row := models.NotificationDelivery{
		ChannelID:     7,
		UserID:        11,
		EventType:     EventDeviceMetricAlert,
		DeviceID:      "123456789",
		Payload:       `{"event":"device.metric.alert"}`,
		Status:        DeliveryStatusPending,
		NextAttemptAt: time.Now().UTC().Add(-time.Second),
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed delivery: %v", err)
	}

	claimed, err := NewNotificationDeliveryService(db).claimDue(context.Background(), 1)
	if err != nil {
		t.Fatalf("claim due: %v", err)
	}
	if len(claimed) != 1 || claimed[0].ID != row.ID {
		t.Fatalf("claimed = %+v, want delivery %d", claimed, row.ID)
	}
	var stored models.NotificationDelivery
	if err := db.First(&stored, row.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.Status != DeliveryStatusDelivering {
		t.Fatalf("stored status = %q, want %q", stored.Status, DeliveryStatusDelivering)
	}
}
