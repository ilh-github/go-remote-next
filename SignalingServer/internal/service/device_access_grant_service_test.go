package service

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newGrantTestDB opens the throwaway database and clears the grants table.
func newGrantTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("QD_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set QD_TEST_DATABASE_URL to a throwaway Postgres to run the grant tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.DeviceAccessGrant{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec("TRUNCATE device_access_grants").Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return db
}

func TestDeviceAccessGrantUsableAt(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	revoked := now.Add(-time.Minute)
	cases := []struct {
		name  string
		grant models.DeviceAccessGrant
		want  bool
	}{
		{"fresh", models.DeviceAccessGrant{ExpiresAt: now.Add(time.Hour)}, true},
		{"expired", models.DeviceAccessGrant{ExpiresAt: now.Add(-time.Second)}, false},
		{"expires exactly now is over", models.DeviceAccessGrant{ExpiresAt: now}, false},
		{"revoked", models.DeviceAccessGrant{ExpiresAt: now.Add(time.Hour), RevokedAt: &revoked}, false},
		{"used up", models.DeviceAccessGrant{ExpiresAt: now.Add(time.Hour), MaxUses: 1, Uses: 1}, false},
		{"one use left", models.DeviceAccessGrant{ExpiresAt: now.Add(time.Hour), MaxUses: 2, Uses: 1}, true},
		{"unlimited uses", models.DeviceAccessGrant{ExpiresAt: now.Add(time.Hour), MaxUses: 0, Uses: 99}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.grant.UsableAt(now); got != tc.want {
				t.Errorf("UsableAt = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCreateDeviceAccessGrantValidation(t *testing.T) {
	db := newGrantTestDB(t)
	svc := NewDeviceAccessGrantService(db)
	ctx := context.Background()

	if _, err := svc.Create(ctx, "dev-1", 1, CreateDeviceAccessGrantInput{ExpiresInMinutes: 0}); !errors.Is(err, ErrInvalidAccessGrant) {
		t.Errorf("zero lifetime err = %v, want ErrInvalidAccessGrant", err)
	}
	if _, err := svc.Create(ctx, "dev-1", 1, CreateDeviceAccessGrantInput{ExpiresInMinutes: 30, MaxUses: -1}); !errors.Is(err, ErrInvalidAccessGrant) {
		t.Errorf("negative max_uses err = %v, want ErrInvalidAccessGrant", err)
	}

	// A generated code is the right length and stays inside the unambiguous
	// alphabet, so nobody has to guess between O and 0.
	row, err := svc.Create(ctx, "dev-1", 1, CreateDeviceAccessGrantInput{ExpiresInMinutes: 30})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(row.Code) != 8 {
		t.Errorf("generated code %q has length %d, want 8", row.Code, len(row.Code))
	}
	for _, ch := range row.Code {
		if !strings.ContainsRune(grantCodeAlphabet, ch) {
			t.Errorf("generated code %q contains %q, outside the alphabet", row.Code, ch)
		}
	}

	// A caller-supplied code is kept as-is (trimmed).
	custom, err := svc.Create(ctx, "dev-1", 1, CreateDeviceAccessGrantInput{ExpiresInMinutes: 30, Code: "  GUEST123  "})
	if err != nil {
		t.Fatalf("create with custom code: %v", err)
	}
	if custom.Code != "GUEST123" {
		t.Errorf("custom code = %q, want GUEST123", custom.Code)
	}
}

func TestAccessGrantLifecycle(t *testing.T) {
	db := newGrantTestDB(t)
	svc := NewDeviceAccessGrantService(db)
	ctx := context.Background()

	row, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{
		Label:            "guest",
		ExpiresInMinutes: 60,
		MaxUses:          1,
		Code:             "ONCEONLY",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if matched, err := svc.Match(ctx, "dev-1", "ONCEONLY"); err != nil || matched == nil || matched.ID != row.ID {
		t.Fatalf("match before use = (%v, %v), want the grant", matched, err)
	}
	// A different device must not see this grant, and a wrong code matches nothing.
	if matched, _ := svc.Match(ctx, "dev-2", "ONCEONLY"); matched != nil {
		t.Error("grant leaked to another device")
	}
	if matched, _ := svc.Match(ctx, "dev-1", "WRONGCODE"); matched != nil {
		t.Error("wrong code matched a grant")
	}

	ok, err := svc.Consume(ctx, row.ID)
	if err != nil || !ok {
		t.Fatalf("first consume = (%v, %v), want true", ok, err)
	}
	// Single-use grant is now spent: no match, no second consume.
	if matched, _ := svc.Match(ctx, "dev-1", "ONCEONLY"); matched != nil {
		t.Error("a spent single-use grant still matched")
	}
	if ok, _ := svc.Consume(ctx, row.ID); ok {
		t.Error("second consume succeeded on a single-use grant")
	}

	var reloaded models.DeviceAccessGrant
	if err := db.First(&reloaded, row.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Uses != 1 || reloaded.LastUsedAt == nil {
		t.Errorf("after use: uses=%d last_used_at=%v, want 1 and set", reloaded.Uses, reloaded.LastUsedAt)
	}
}

func TestAccessGrantExpiredAndRevoked(t *testing.T) {
	db := newGrantTestDB(t)
	svc := NewDeviceAccessGrantService(db)
	ctx := context.Background()

	expired, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{ExpiresInMinutes: 30, Code: "EXPIRED1"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := db.Model(&models.DeviceAccessGrant{}).Where("id = ?", expired.ID).
		Update("expires_at", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("expire: %v", err)
	}
	if matched, _ := svc.Match(ctx, "dev-1", "EXPIRED1"); matched != nil {
		t.Error("an expired grant matched")
	}
	if ok, _ := svc.Consume(ctx, expired.ID); ok {
		t.Error("an expired grant was consumed")
	}

	live, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{ExpiresInMinutes: 30, Code: "REVOKEME"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.Revoke(ctx, "dev-1", live.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if matched, _ := svc.Match(ctx, "dev-1", "REVOKEME"); matched != nil {
		t.Error("a revoked grant matched")
	}
	if ok, _ := svc.Consume(ctx, live.ID); ok {
		t.Error("a revoked grant was consumed")
	}
	// Revoking through the wrong device is a no-op, not a cross-device revoke.
	other, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{ExpiresInMinutes: 30, Code: "OTHERDEV"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.Revoke(ctx, "dev-2", other.ID); err != nil {
		t.Fatalf("revoke from another device: %v", err)
	}
	if matched, _ := svc.Match(ctx, "dev-1", "OTHERDEV"); matched == nil {
		t.Error("revoking through another device should not revoke the grant")
	}
}

// TestAccessGrantConsumeIsAtomic covers the race that matters for a single-use
// link: two controllers redeeming the last slot must not both win.
func TestAccessGrantConsumeIsAtomic(t *testing.T) {
	db := newGrantTestDB(t)
	svc := NewDeviceAccessGrantService(db)
	ctx := context.Background()

	row, err := svc.Create(ctx, "dev-1", 7, CreateDeviceAccessGrantInput{
		ExpiresInMinutes: 30, MaxUses: 1, Code: "RACEONCE",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	const workers = 8
	var mu sync.Mutex
	wins := 0
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := svc.Consume(ctx, row.ID)
			if err != nil {
				t.Errorf("consume: %v", err)
				return
			}
			if ok {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Errorf("%d concurrent consumes won, want exactly 1", wins)
	}

	var reloaded models.DeviceAccessGrant
	if err := db.First(&reloaded, row.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Uses != 1 {
		t.Errorf("uses = %d, want 1", reloaded.Uses)
	}
}
