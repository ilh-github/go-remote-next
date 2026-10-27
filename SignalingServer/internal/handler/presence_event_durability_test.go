package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"quickdesk/signaling/internal/config"
	"quickdesk/signaling/internal/database"
	"quickdesk/signaling/internal/models"
	"quickdesk/signaling/internal/repository"
	"quickdesk/signaling/internal/service"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// The offline half of `device.online.changed` was made durable in the outbox
// (PresenceWatcher.WithOutbox). The online half was not: the WS handshake and
// the heartbeat route still publish straight to the bus, so a Redis failure
// between "the device came online" and "the owner's client was told" loses the
// transition with no row to retry. The owner's device list keeps showing
// offline until the next transition, and the offline alert evaluator - which
// reads presence, not events - has already moved on.
//
// Both halves have to behave the same way: a device whose online state the
// owner is told about is the same promise whichever direction it moved.
func TestHostPublishOnlineEventIsDurableInOutbox(t *testing.T) {
	db, rdb, bus, outbox := newPresenceEventTestStack(t)
	owner := uint(7301)
	deviceID := "900000001"
	seedOwnedDevice(t, db, owner, deviceID)

	h := newHostHandlerForPresenceEvent(db, rdb, bus).WithPresenceEventOutbox(outbox)

	h.publishPresenceEvent(context.Background(), deviceID, owner, false)

	rows := assertPresenceOutboxRows(t, db, 1)
	if rows[0].DeviceID != deviceID {
		t.Fatalf("outbox device_id = %q, want %q", rows[0].DeviceID, deviceID)
	}
	if rows[0].UserID != owner {
		t.Fatalf("outbox user_id = %d, want %d", rows[0].UserID, owner)
	}
	var evt service.Event
	if err := json.Unmarshal([]byte(rows[0].Payload), &evt); err != nil {
		t.Fatalf("decode outbox payload: %v", err)
	}
	if online, _ := evt.Data["online"].(bool); !online {
		t.Fatal("the durable online event carries online=false")
	}

	// The row must actually reach the bus when the worker drains it, or the
	// durability is only a write with nothing behind it.
	if _, err := service.NewEventOutbox(db, bus).Drain(context.Background(), time.Now().UTC(), 10); err != nil {
		t.Fatalf("Drain(): %v", err)
	}
	assertPresenceEventDelivered(t, db, rows[0].ID)
}

// A deployment that never configured an outbox must keep publishing directly.
// The durable path is an upgrade, not a new requirement to boot.
func TestHostPublishOnlineEventFallsBackToDirectPublish(t *testing.T) {
	db, rdb, bus, _ := newPresenceEventTestStack(t)
	owner := uint(7302)
	deviceID := "900000002"
	seedOwnedDevice(t, db, owner, deviceID)

	seen := make(chan service.Event, 4)
	bus.Subscribe(&presenceEventCapture{events: seen})

	h := newHostHandlerForPresenceEvent(db, rdb, bus)
	h.publishPresenceEvent(context.Background(), deviceID, owner, false)

	select {
	case evt := <-seen:
		if evt.Type != service.EventDeviceOnlineChanged {
			t.Fatalf("event type = %q, want %q", evt.Type, service.EventDeviceOnlineChanged)
		}
		if evt.UserID != owner || evt.DeviceID != deviceID {
			t.Fatalf("event = user %d device %q, want user %d device %q",
				evt.UserID, evt.DeviceID, owner, deviceID)
		}
		if online, _ := evt.Data["online"].(bool); !online {
			t.Fatal("the direct publish carries online=false")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no direct online event was observed without an outbox")
	}
	if rows := assertPresenceOutboxRows(t, db, 0); len(rows) != 0 {
		t.Fatalf("outbox rows = %d, want 0 on the best-effort path", len(rows))
	}
}

// The outbox write has to be able to fail, and the caller must not report a
// state transition it could not make durable. Otherwise this is a decoration.
func TestHostPublishOnlineEventSurfacesOutboxFailure(t *testing.T) {
	db, rdb, bus, _ := newPresenceEventTestStack(t)
	owner := uint(7303)
	seedOwnedDevice(t, db, owner, "900000003")

	h := newHostHandlerForPresenceEvent(db, rdb, bus)
	h.WithPresenceEventOutbox(failingPresenceOutbox{})

	if err := h.publishPresenceEventChecked(context.Background(), "900000003", owner, false); err == nil {
		t.Fatal("a failed outbox write was reported as success")
	}
}

var errPresenceOutboxFailed = errors.New("presence outbox write failed")

// The route, not the helper: the wiring is the part that rots. A durable
// publisher nobody called is the same as no publisher, and only driving the
// real heartbeat handler proves the outbox is reached.
func TestHeartbeatRouteMakesOnlineTransitionDurable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, rdb, bus, outbox := newPresenceEventTestStack(t)
	owner := uint(7304)
	deviceID := "900000004"
	seedOwnedDevice(t, db, owner, deviceID)

	h := newHostHandlerForPresenceEvent(db, rdb, bus).WithPresenceEventOutbox(outbox)

	// Presence is online only when both signals exist. The handler refreshes
	// the heartbeat key itself; the WS key is what makes the composite true.
	presence := service.NewPresenceService(rdb, "heartbeat-route-test")
	if err := presence.MarkWSConnected(context.Background(), deviceID); err != nil {
		t.Fatalf("MarkWSConnected(): %v", err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("auth.device_id", deviceID)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/devices/"+deviceID+"/heartbeat", nil)

	h.Heartbeat(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", recorder.Code, recorder.Body.String())
	}
	rows := assertPresenceOutboxRows(t, db, 1)
	if rows[0].DeviceID != deviceID || rows[0].UserID != owner {
		t.Fatalf("outbox row = device %q user %d, want device %q user %d",
			rows[0].DeviceID, rows[0].UserID, deviceID, owner)
	}
}

type failingPresenceOutbox struct{}

func (failingPresenceOutbox) EnqueueTx(_ context.Context, _ *gorm.DB, _ service.Event) (service.Event, error) {
	return service.Event{}, errPresenceOutboxFailed
}

func newPresenceEventTestStack(t *testing.T) (*gorm.DB, *redis.Client, *service.EventBus, *service.EventOutbox) {
	t.Helper()
	db, err := database.InitSQLite(&config.Config{
		Database: config.DatabaseConfig{
			Driver: "sqlite",
			Path:   filepath.Join(t.TempDir(), "presence-event.db"),
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
	if err := db.AutoMigrate(&models.User{}, &models.Device{}, &models.EventOutbox{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	bus := service.NewEventBus(client)
	return db, client, bus, service.NewEventOutbox(db, bus)
}

func newHostHandlerForPresenceEvent(db *gorm.DB, rdb *redis.Client, bus *service.EventBus) *HostHandler {
	return NewHostHandler(
		db,
		service.NewDeviceService(repository.NewDeviceRepository(db), service.NewDeviceSecretService()),
		service.NewTokenService(rdb),
		service.NewPresenceService(rdb, "presence-event-test"),
		service.NewSettingsService(db),
		service.NewRateLimitService(rdb),
		service.NewDeviceAccessService(db),
		service.NewDeviceAccessPolicyService(db),
		service.NewDeviceAccessGrantService(db),
		service.NewDeviceAccessApprovalService(db),
		bus,
		nil,
	)
}

func seedOwnedDevice(t *testing.T, db *gorm.DB, owner uint, deviceID string) {
	t.Helper()
	if err := db.Create(&models.User{
		ID:       owner,
		Username: "presence-owner-" + deviceID,
		Password: "test-hash",
	}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&models.Device{
		DeviceID:   deviceID,
		DeviceUUID: "uuid-" + deviceID,
		UserID:     &owner,
		AccessCode: "135790",
	}).Error; err != nil {
		t.Fatalf("seed device: %v", err)
	}
}

func assertPresenceOutboxRows(t *testing.T, db *gorm.DB, want int) []models.EventOutbox {
	t.Helper()
	var rows []models.EventOutbox
	if err := db.Where("event_type = ?", string(service.EventDeviceOnlineChanged)).Find(&rows).Error; err != nil {
		t.Fatalf("query event_outbox: %v", err)
	}
	if len(rows) != want {
		t.Fatalf("outbox rows = %d, want %d", len(rows), want)
	}
	return rows
}

func assertPresenceEventDelivered(t *testing.T, db *gorm.DB, id string) {
	t.Helper()
	var row models.EventOutbox
	if err := db.Where("id = ?", id).First(&row).Error; err != nil {
		t.Fatalf("reload outbox row: %v", err)
	}
	if row.Status != "delivered" {
		t.Fatalf("outbox status = %q, want delivered", row.Status)
	}
}

type presenceEventCapture struct {
	events chan service.Event
}

func (c *presenceEventCapture) Name() string { return "presence-event-capture" }

func (c *presenceEventCapture) HandleEvent(_ context.Context, evt service.Event) {
	select {
	case c.events <- evt:
	default:
	}
}
