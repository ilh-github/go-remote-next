package service

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func newEventOutboxTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(
		"file:"+filepath.ToSlash(filepath.Join(t.TempDir(), "outbox.db"))+"?cache=shared",
	), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.EventOutbox{},
		&models.ScheduledTask{},
		&models.DeviceAccessLog{},
		&models.UserDevice{},
		&models.DeviceAccessPolicy{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newEventOutboxTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client, *EventBus) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return server, client, NewEventBus(client)
}

func TestEventOutboxEnqueueRollsBackWithBusinessTransaction(t *testing.T) {
	db := newEventOutboxTestDB(t)
	_, _, bus := newEventOutboxTestRedis(t)
	outbox := NewEventOutbox(db, bus)
	sentinel := errors.New("business rollback")

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&models.ScheduledTask{
			UserID:    7,
			Kind:      "probe",
			NextRunAt: time.Now().UTC(),
			Enabled:   true,
		}).Error; err != nil {
			return err
		}
		if _, err := outbox.EnqueueTx(context.Background(), tx, Event{
			Type:   EventScheduledTaskFailed,
			UserID: 7,
		}); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("transaction error = %v, want sentinel", err)
	}

	var tasks int64
	if err := db.Model(&models.ScheduledTask{}).Count(&tasks).Error; err != nil {
		t.Fatalf("count tasks: %v", err)
	}
	var rows int64
	if err := db.Model(&models.EventOutbox{}).Count(&rows).Error; err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if tasks != 0 || rows != 0 {
		t.Fatalf("rollback left task=%d outbox=%d rows, want 0/0", tasks, rows)
	}
}

func TestEventOutboxDrainPublishesOnceAndMarksDelivered(t *testing.T) {
	db := newEventOutboxTestDB(t)
	_, client, bus := newEventOutboxTestRedis(t)
	outbox := NewEventOutbox(db, bus)
	capture := &captureSubscriber{events: make(chan Event, 2)}
	bus.Subscribe(capture)
	now := time.Now().UTC()

	event, err := outbox.EnqueueTx(context.Background(), db, Event{
		Type:     EventScheduledTaskFailed,
		UserID:   7,
		DeviceID: "device-7",
		Ts:       now.Add(-time.Second),
		Data:     map[string]interface{}{"task_id": 9},
	})
	if err != nil {
		t.Fatalf("EnqueueTx: %v", err)
	}

	claimed, err := outbox.Drain(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if claimed != 1 {
		t.Fatalf("claimed = %d, want 1", claimed)
	}

	select {
	case got := <-capture.events:
		if got.ID != event.ID || got.Type != EventScheduledTaskFailed || got.ServerRev == 0 {
			t.Fatalf("published event = %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("outbox event did not reach subscriber")
	}

	stream, err := client.XRange(context.Background(), "qd:events:user:7", "-", "+").Result()
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if len(stream) != 1 {
		t.Fatalf("stream entries = %d, want 1", len(stream))
	}

	var row models.EventOutbox
	if err := db.First(&row, "id = ?", event.ID).Error; err != nil {
		t.Fatalf("load outbox row: %v", err)
	}
	if row.Status != eventOutboxDelivered || row.Attempts != 1 || row.ClaimID != "" || row.DeliveredAt == nil {
		t.Fatalf("delivered row = %+v", row)
	}

	claimed, err = outbox.Drain(context.Background(), now.Add(time.Minute), 10)
	if err != nil {
		t.Fatalf("second Drain: %v", err)
	}
	if claimed != 0 {
		t.Fatalf("second drain claimed = %d, want 0", claimed)
	}
}

func TestEventOutboxDoesNotStealLiveClaim(t *testing.T) {
	db := newEventOutboxTestDB(t)
	_, _, bus := newEventOutboxTestRedis(t)
	outbox := NewEventOutbox(db, bus)
	now := time.Now().UTC()
	expires := now.Add(time.Minute)
	row := seedOutboxRow(t, db, now.Add(-time.Minute), &expires, "live-worker")

	claimed, err := outbox.Drain(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if claimed != 0 {
		t.Fatalf("claimed = %d, want 0 for a live claim", claimed)
	}
	var got models.EventOutbox
	if err := db.First(&got, "id = ?", row.ID).Error; err != nil {
		t.Fatalf("load row: %v", err)
	}
	if got.ClaimID != "live-worker" || got.Status != eventOutboxPending {
		t.Fatalf("live claim mutated: %+v", got)
	}
}

func TestEventOutboxRecoversExpiredClaim(t *testing.T) {
	db := newEventOutboxTestDB(t)
	_, _, bus := newEventOutboxTestRedis(t)
	outbox := NewEventOutbox(db, bus)
	now := time.Now().UTC()
	expired := now.Add(-time.Minute)
	row := seedOutboxRow(t, db, now.Add(-2*time.Minute), &expired, "dead-worker")

	claimed, err := outbox.Drain(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if claimed != 1 {
		t.Fatalf("claimed = %d, want 1", claimed)
	}
	var got models.EventOutbox
	if err := db.First(&got, "id = ?", row.ID).Error; err != nil {
		t.Fatalf("load row: %v", err)
	}
	if got.Status != eventOutboxDelivered || got.ClaimID != "" || got.Attempts != 2 {
		t.Fatalf("recovered row = %+v", got)
	}
}

func TestEventOutboxKeepsRowPendingWhenRedisPublishFails(t *testing.T) {
	db := newEventOutboxTestDB(t)
	redisServer, _, bus := newEventOutboxTestRedis(t)
	outbox := NewEventOutbox(db, bus)
	now := time.Now().UTC()
	event, err := outbox.EnqueueTx(context.Background(), db, Event{
		Type:   EventScheduledTaskFailed,
		UserID: 7,
		Ts:     now.Add(-time.Second),
	})
	if err != nil {
		t.Fatalf("EnqueueTx: %v", err)
	}
	redisServer.Close()

	claimed, err := outbox.Drain(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if claimed != 1 {
		t.Fatalf("claimed = %d, want 1", claimed)
	}
	var row models.EventOutbox
	if err := db.First(&row, "id = ?", event.ID).Error; err != nil {
		t.Fatalf("load row: %v", err)
	}
	if row.Status != eventOutboxPending || row.Attempts != 1 || row.ClaimID != "" {
		t.Fatalf("failed publish row = %+v", row)
	}
	if !row.NextAttemptAt.After(now) || row.LastError == "" {
		t.Fatalf("retry metadata missing: %+v", row)
	}
}

type outboxTestAction struct {
	err error
}

func (a *outboxTestAction) Run(context.Context, models.ScheduledTask) error {
	return a.err
}

type failingScheduledTaskOutbox struct {
	err error
}

func (o *failingScheduledTaskOutbox) EnqueueTx(context.Context, *gorm.DB, Event) (Event, error) {
	return Event{}, o.err
}

func TestScheduledTaskTerminalFailureCommitsOutboxWithOutcome(t *testing.T) {
	db := newEventOutboxTestDB(t)
	bus := NewEventBus(nil)
	capture := &captureSubscriber{events: make(chan Event, 1)}
	bus.Subscribe(capture)
	outbox := NewEventOutbox(db, bus)
	notifier := &capturingNotifier{}
	svc := NewScheduledTaskService(db).
		RegisterAction("probe", &outboxTestAction{err: errors.New("probe failed")}).
		WithNotifier(notifier).
		WithOutbox(outbox)
	now := time.Now().UTC()
	task := models.ScheduledTask{
		UserID:      7,
		Kind:        "probe",
		NextRunAt:   now,
		Enabled:     false,
		MaxAttempts: 1,
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	svc.runOne(context.Background(), task, now)

	var taskRow models.ScheduledTask
	if err := db.First(&taskRow, task.ID).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	if taskRow.LastStatus != ScheduledTaskStatusFailed || taskRow.LastError == "" {
		t.Fatalf("task outcome = %+v, want failed", taskRow)
	}
	var outboxRows []models.EventOutbox
	if err := db.Find(&outboxRows).Error; err != nil {
		t.Fatalf("load outbox: %v", err)
	}
	if len(outboxRows) != 1 || outboxRows[0].Status != eventOutboxPending {
		t.Fatalf("outbox rows = %+v, want one pending event", outboxRows)
	}
	if got := notifier.all(); len(got) != 0 {
		t.Fatalf("legacy notifier received %d events despite outbox", len(got))
	}

	claimed, err := outbox.Drain(context.Background(), now.Add(time.Second), 10)
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if claimed != 1 {
		t.Fatalf("claimed = %d, want 1", claimed)
	}
	select {
	case event := <-capture.events:
		if event.Type != EventScheduledTaskFailed || event.UserID != 7 {
			t.Fatalf("published event = %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("terminal failure event was not published")
	}
}

func TestScheduledTaskOutboxFailureRollsBackOutcome(t *testing.T) {
	db := newEventOutboxTestDB(t)
	notifier := &capturingNotifier{}
	svc := NewScheduledTaskService(db).
		RegisterAction("probe", &outboxTestAction{err: errors.New("probe failed")}).
		WithNotifier(notifier).
		WithOutbox(&failingScheduledTaskOutbox{err: errors.New("outbox unavailable")})
	now := time.Now().UTC()
	task := models.ScheduledTask{
		UserID:      7,
		Kind:        "probe",
		NextRunAt:   now,
		Enabled:     false,
		MaxAttempts: 1,
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}

	svc.runOne(context.Background(), task, now)

	var taskRow models.ScheduledTask
	if err := db.First(&taskRow, task.ID).Error; err != nil {
		t.Fatalf("load task: %v", err)
	}
	if taskRow.LastStatus != "" || taskRow.LastError != "" {
		t.Fatalf("outcome was not rolled back: %+v", taskRow)
	}
	if got := notifier.all(); len(got) != 0 {
		t.Fatalf("legacy notifier fired after rollback: %+v", got)
	}
}

func TestDeviceAccessEventCommitsLogAndOutboxTogether(t *testing.T) {
	db := newEventOutboxTestDB(t)
	_, _, bus := newEventOutboxTestRedis(t)
	outbox := NewEventOutbox(db, bus)
	access := NewDeviceAccessService(db).WithOutbox(outbox)
	if err := db.Create(&models.UserDevice{
		UserID:   7,
		DeviceID: "device-7",
		Status:   true,
	}).Error; err != nil {
		t.Fatalf("create binding: %v", err)
	}

	row, err := access.RecordWithEvent(context.Background(), RecordDeviceAccessInput{
		DeviceID:  "device-7",
		ClientID:  "client-1",
		SessionID: "session-1",
		IP:        "203.0.113.7",
		Result:    DeviceAccessGranted,
		Reason:    "access code accepted",
	}, EventDeviceAccessGranted)
	if err != nil {
		t.Fatalf("RecordWithEvent: %v", err)
	}
	if row.OwnerID != 7 || row.Result != DeviceAccessGranted {
		t.Fatalf("access row = %+v", row)
	}

	var outboxRows []models.EventOutbox
	if err := db.Find(&outboxRows).Error; err != nil {
		t.Fatalf("load outbox: %v", err)
	}
	if len(outboxRows) != 1 || outboxRows[0].Status != eventOutboxPending {
		t.Fatalf("outbox rows = %+v, want one pending event", outboxRows)
	}
	var event Event
	if err := json.Unmarshal([]byte(outboxRows[0].Payload), &event); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if event.Type != EventDeviceAccessGranted || event.UserID != 7 ||
		event.DeviceID != "device-7" || event.Data["log_id"] != float64(row.ID) {
		t.Fatalf("outbox event = %+v", event)
	}
}

func TestDeviceAccessEventOutboxFailureRollsBackLog(t *testing.T) {
	db := newEventOutboxTestDB(t)
	access := NewDeviceAccessService(db).WithOutbox(
		&failingScheduledTaskOutbox{err: errors.New("outbox unavailable")},
	)
	if err := db.Create(&models.UserDevice{
		UserID:   7,
		DeviceID: "device-7",
		Status:   true,
	}).Error; err != nil {
		t.Fatalf("create binding: %v", err)
	}

	_, err := access.RecordWithEvent(context.Background(), RecordDeviceAccessInput{
		DeviceID: "device-7",
		ClientID: "client-1",
		IP:       "203.0.113.7",
		Result:   DeviceAccessDeniedPolicy,
		Reason:   "policy denied",
	}, EventDeviceAccessDenied)
	if err == nil {
		t.Fatal("RecordWithEvent succeeded despite outbox failure")
	}

	var logs int64
	if err := db.Model(&models.DeviceAccessLog{}).Count(&logs).Error; err != nil {
		t.Fatalf("count access logs: %v", err)
	}
	if logs != 0 {
		t.Fatalf("access logs = %d, want rollback to 0", logs)
	}
}

func TestDevicePolicyEventCommitsPolicyAndOutboxTogether(t *testing.T) {
	db := newEventOutboxTestDB(t)
	_, _, bus := newEventOutboxTestRedis(t)
	outbox := NewEventOutbox(db, bus)
	policies := NewDeviceAccessPolicyService(db).WithOutbox(outbox)

	policy, err := policies.UpsertWithEvent(
		context.Background(),
		"device-7",
		7,
		UpsertDeviceAccessPolicyInput{
			MaxSessions: 2,
			AllowIPs:    []string{"203.0.113.0/24"},
			DenyIPs:     []string{"203.0.113.9"},
		},
		func(policy *models.DeviceAccessPolicy) Event {
			return Event{
				Type:     EventDevicePolicyChanged,
				UserID:   policy.OwnerID,
				DeviceID: policy.DeviceID,
				Data: map[string]interface{}{
					"deny_ips": policy.DenyIPs,
					"ip":       "198.51.100.7",
				},
			}
		},
	)
	if err != nil {
		t.Fatalf("UpsertWithEvent: %v", err)
	}
	if policy.OwnerID != 7 || policy.MaxSessions != 2 {
		t.Fatalf("policy = %+v", policy)
	}

	var outboxRows []models.EventOutbox
	if err := db.Find(&outboxRows).Error; err != nil {
		t.Fatalf("load outbox: %v", err)
	}
	if len(outboxRows) != 1 || outboxRows[0].Status != eventOutboxPending {
		t.Fatalf("outbox rows = %+v, want one pending event", outboxRows)
	}
	var event Event
	if err := json.Unmarshal([]byte(outboxRows[0].Payload), &event); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if event.Type != EventDevicePolicyChanged || event.UserID != 7 || event.DeviceID != "device-7" {
		t.Fatalf("outbox event = %+v", event)
	}
}

func TestDevicePolicyEventOutboxFailureRollsBackPolicy(t *testing.T) {
	db := newEventOutboxTestDB(t)
	policies := NewDeviceAccessPolicyService(db).WithOutbox(
		&failingScheduledTaskOutbox{err: errors.New("outbox unavailable")},
	)

	_, err := policies.UpsertWithEvent(
		context.Background(),
		"device-7",
		7,
		UpsertDeviceAccessPolicyInput{MaxSessions: 1},
		func(policy *models.DeviceAccessPolicy) Event {
			return Event{
				Type:     EventDevicePolicyChanged,
				UserID:   policy.OwnerID,
				DeviceID: policy.DeviceID,
			}
		},
	)
	if err == nil {
		t.Fatal("UpsertWithEvent succeeded despite outbox failure")
	}

	var count int64
	if err := db.Model(&models.DeviceAccessPolicy{}).Count(&count).Error; err != nil {
		t.Fatalf("count policies: %v", err)
	}
	if count != 0 {
		t.Fatalf("policies = %d, want rollback to 0", count)
	}
}

func seedOutboxRow(t *testing.T, db *gorm.DB, due time.Time, claimExpires *time.Time, claimID string) models.EventOutbox {
	t.Helper()
	event := prepareEvent(Event{
		Type:   EventScheduledTaskFailed,
		UserID: 7,
		Ts:     due,
	})
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	row := models.EventOutbox{
		ID:             event.ID,
		EventType:      event.Type,
		UserID:         event.UserID,
		Payload:        string(payload),
		Status:         eventOutboxPending,
		Attempts:       1,
		NextAttemptAt:  due,
		ClaimID:        claimID,
		ClaimExpiresAt: claimExpires,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed outbox row: %v", err)
	}
	return row
}
