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
		&models.DeviceAccessApproval{},
		&models.Device{},
		&models.RemoteSession{},
		&models.SessionRecording{},
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

func TestDeviceAccessApprovalEventsCommitWithApprovalRows(t *testing.T) {
	db := newEventOutboxTestDB(t)
	_, _, bus := newEventOutboxTestRedis(t)
	outbox := NewEventOutbox(db, bus)
	seedOutboxTestDevice(t, db, "D1234567", 7)
	approvals := NewDeviceAccessApprovalService(db).WithOutbox(outbox)
	ctx := context.Background()

	row, _, err := approvals.CreateWithEvent(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D1234567",
		OwnerID:  7,
		ClientID: "controller-1",
		ClientIP: "203.0.113.7",
		Source:   models.ApprovalSourceDeviceCode,
		TTL:      time.Minute,
	}, func(row *models.DeviceAccessApproval) Event {
		return Event{
			Type:     EventDeviceAccessApprovalRequested,
			UserID:   row.OwnerID,
			DeviceID: row.DeviceID,
			Data: map[string]interface{}{
				"approval_id": row.ID,
				"trigger":     ApprovalTriggerAlways,
			},
		}
	})
	if err != nil {
		t.Fatalf("CreateWithEvent: %v", err)
	}
	if row.Status != models.ApprovalPending {
		t.Fatalf("created approval = %+v, want pending", row)
	}

	var outboxRows []models.EventOutbox
	if err := db.Order("id").Find(&outboxRows).Error; err != nil {
		t.Fatalf("load requested outbox row: %v", err)
	}
	if len(outboxRows) != 1 || outboxRows[0].Status != eventOutboxPending {
		t.Fatalf("outbox rows = %+v, want one pending requested event", outboxRows)
	}
	var requested Event
	if err := json.Unmarshal([]byte(outboxRows[0].Payload), &requested); err != nil {
		t.Fatalf("decode requested event: %v", err)
	}
	if requested.Type != EventDeviceAccessApprovalRequested || requested.UserID != 7 ||
		requested.DeviceID != "D1234567" ||
		requested.Data["approval_id"] != float64(row.ID) {
		t.Fatalf("requested event = %+v", requested)
	}

	decided, changed, err := approvals.DecideWithEvent(
		ctx, "D1234567", row.ID, true, 7, "on my way",
		func(row *models.DeviceAccessApproval) Event {
			return Event{
				Type:     EventDeviceAccessApprovalDecided,
				UserID:   row.OwnerID,
				DeviceID: row.DeviceID,
				Data: map[string]interface{}{
					"approval_id": row.ID,
					"status":      row.Status,
					"ip":          "198.51.100.7",
				},
			}
		},
	)
	if err != nil {
		t.Fatalf("DecideWithEvent: %v", err)
	}
	if !changed || decided == nil || decided.Status != models.ApprovalApproved {
		t.Fatalf("decided approval = %+v, changed=%v", decided, changed)
	}

	outboxRows = nil
	if err := db.Order("id").Find(&outboxRows).Error; err != nil {
		t.Fatalf("load decided outbox rows: %v", err)
	}
	if len(outboxRows) != 2 {
		t.Fatalf("outbox rows = %d, want requested+decided", len(outboxRows))
	}
	eventsByType := make(map[string]Event, len(outboxRows))
	for _, outboxRow := range outboxRows {
		var event Event
		if err := json.Unmarshal([]byte(outboxRow.Payload), &event); err != nil {
			t.Fatalf("decode outbox event %s: %v", outboxRow.ID, err)
		}
		eventsByType[event.Type] = event
	}
	decidedEvent := eventsByType[EventDeviceAccessApprovalDecided]
	if decidedEvent.Type != EventDeviceAccessApprovalDecided || decidedEvent.UserID != 7 ||
		decidedEvent.Data["status"] != models.ApprovalApproved ||
		decidedEvent.Data["ip"] != "198.51.100.7" {
		t.Fatalf("decided event = %+v", decidedEvent)
	}

	_, changed, err = approvals.DecideWithEvent(
		ctx, "D1234567", row.ID, false, 7, "flip",
		func(row *models.DeviceAccessApproval) Event {
			return Event{
				Type:     EventDeviceAccessApprovalDecided,
				UserID:   row.OwnerID,
				DeviceID: row.DeviceID,
			}
		},
	)
	if err != nil {
		t.Fatalf("second DecideWithEvent: %v", err)
	}
	if changed {
		t.Fatal("deciding an approval twice must not emit or change anything")
	}
	var total int64
	if err := db.Model(&models.EventOutbox{}).Count(&total).Error; err != nil {
		t.Fatalf("count outbox rows: %v", err)
	}
	if total != 2 {
		t.Fatalf("outbox rows after no-op decision = %d, want 2", total)
	}
}

func TestDeviceAccessApprovalOutboxFailureRollsBackCreateAndDecision(t *testing.T) {
	db := newEventOutboxTestDB(t)
	seedOutboxTestDevice(t, db, "D7654321", 7)
	ctx := context.Background()
	failing := NewDeviceAccessApprovalService(db).WithOutbox(
		&failingScheduledTaskOutbox{err: errors.New("outbox unavailable")},
	)

	_, _, err := failing.CreateWithEvent(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D7654321",
		OwnerID:  7,
		ClientID: "controller-1",
		ClientIP: "203.0.113.7",
		TTL:      time.Minute,
	}, func(row *models.DeviceAccessApproval) Event {
		return Event{
			Type:     EventDeviceAccessApprovalRequested,
			UserID:   row.OwnerID,
			DeviceID: row.DeviceID,
		}
	})
	if err == nil {
		t.Fatal("CreateWithEvent succeeded despite outbox failure")
	}
	var approvals int64
	if err := db.Model(&models.DeviceAccessApproval{}).Count(&approvals).Error; err != nil {
		t.Fatalf("count approvals after create rollback: %v", err)
	}
	if approvals != 0 {
		t.Fatalf("approvals after create rollback = %d, want 0", approvals)
	}

	created, _, err := NewDeviceAccessApprovalService(db).Create(ctx, CreateDeviceAccessApprovalInput{
		DeviceID: "D7654321",
		OwnerID:  7,
		ClientID: "controller-1",
		ClientIP: "203.0.113.7",
		TTL:      time.Minute,
	})
	if err != nil {
		t.Fatalf("seed pending approval: %v", err)
	}
	_, _, err = failing.DecideWithEvent(
		ctx, "D7654321", created.ID, true, 7, "approve",
		func(row *models.DeviceAccessApproval) Event {
			return Event{
				Type:     EventDeviceAccessApprovalDecided,
				UserID:   row.OwnerID,
				DeviceID: row.DeviceID,
			}
		},
	)
	if err == nil {
		t.Fatal("DecideWithEvent succeeded despite outbox failure")
	}
	var stored models.DeviceAccessApproval
	if err := db.First(&stored, created.ID).Error; err != nil {
		t.Fatalf("load approval after decision rollback: %v", err)
	}
	if stored.Status != models.ApprovalPending || stored.DecidedAt != nil {
		t.Fatalf("decision was not rolled back: %+v", stored)
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

func TestRemoteSessionLifecycleEventsCommitWithSessionRows(t *testing.T) {
	db := newEventOutboxTestDB(t)
	_, _, bus := newEventOutboxTestRedis(t)
	outbox := NewEventOutbox(db, bus)
	seedOutboxTestDevice(t, db, "device-7", 7)
	sessions := NewRemoteSessionService(db).WithOutbox(outbox)
	ctx := context.Background()

	opened, err := sessions.Open(ctx, OpenRemoteSessionInput{
		SessionID:  "session-1",
		DeviceID:   "device-7",
		ClientID:   "client-1",
		ClientIP:   "203.0.113.7",
		InstanceID: "instance-a",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !opened.Active || opened.UserID != 7 {
		t.Fatalf("opened session = %+v", opened)
	}

	closed, changed, err := sessions.Close(ctx, "session-1", models.RemoteSessionEndDisconnected)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !changed || closed.EndedAt == nil || closed.EndReason != models.RemoteSessionEndDisconnected {
		t.Fatalf("closed session = %+v, changed=%v", closed, changed)
	}

	var rows []models.EventOutbox
	if err := db.Order("created_at, id").Find(&rows).Error; err != nil {
		t.Fatalf("load outbox: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("outbox rows = %d, want started and ended", len(rows))
	}
	types := map[string]bool{}
	for i := range rows {
		var event Event
		if err := json.Unmarshal([]byte(rows[i].Payload), &event); err != nil {
			t.Fatalf("decode event: %v", err)
		}
		types[event.Type] = true
		if event.UserID != 7 || event.DeviceID != "device-7" {
			t.Fatalf("event routing = %+v", event)
		}
	}
	if !types[EventRemoteSessionStarted] || !types[EventRemoteSessionEnded] {
		t.Fatalf("event types = %+v", types)
	}
}

func TestRemoteSessionOrphanSweepEnqueuesEndedEvent(t *testing.T) {
	db := newEventOutboxTestDB(t)
	_, _, bus := newEventOutboxTestRedis(t)
	outbox := NewEventOutbox(db, bus)
	seedOutboxTestDevice(t, db, "device-7", 7)
	if err := db.Create(&models.RemoteSession{
		SessionID:  "session-orphan",
		UserID:     7,
		DeviceID:   "device-7",
		ClientID:   "client-1",
		ClientIP:   "203.0.113.7",
		InstanceID: "dead-instance",
		StartedAt:  time.Now().UTC().Add(-time.Minute),
	}).Error; err != nil {
		t.Fatalf("create session: %v", err)
	}
	sessions := NewRemoteSessionService(db).WithOutbox(outbox)

	closed, err := sessions.CloseOrphanedSessions(
		context.Background(),
		[]string{"live-instance"},
		models.RemoteSessionEndServerRestart,
	)
	if err != nil {
		t.Fatalf("CloseOrphanedSessions: %v", err)
	}
	if closed != 1 {
		t.Fatalf("closed = %d, want 1", closed)
	}
	var row models.EventOutbox
	if err := db.First(&row).Error; err != nil {
		t.Fatalf("load outbox: %v", err)
	}
	var event Event
	if err := json.Unmarshal([]byte(row.Payload), &event); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if event.Type != EventRemoteSessionEnded || event.Data["end_reason"] != models.RemoteSessionEndServerRestart {
		t.Fatalf("sweep event = %+v", event)
	}
}

func TestRemoteSessionOutboxFailureRollsBackOpen(t *testing.T) {
	db := newEventOutboxTestDB(t)
	seedOutboxTestDevice(t, db, "device-7", 7)
	sessions := NewRemoteSessionService(db).WithOutbox(
		&failingScheduledTaskOutbox{err: errors.New("outbox unavailable")},
	)

	_, err := sessions.Open(context.Background(), OpenRemoteSessionInput{
		SessionID:  "session-1",
		DeviceID:   "device-7",
		ClientID:   "client-1",
		ClientIP:   "203.0.113.7",
		InstanceID: "instance-a",
	})
	if err == nil {
		t.Fatal("Open succeeded despite outbox failure")
	}
	var count int64
	if err := db.Model(&models.RemoteSession{}).Count(&count).Error; err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if count != 0 {
		t.Fatalf("sessions = %d, want rollback to 0", count)
	}
}

func TestRecordingLifecycleEventsCommitWithRecordingRows(t *testing.T) {
	db := newEventOutboxTestDB(t)
	_, _, bus := newEventOutboxTestRedis(t)
	outbox := NewEventOutbox(db, bus)
	recordings := NewSessionRecordingService(db).WithOutbox(outbox)
	ctx := context.Background()

	created, err := recordings.CreateWithEvent(ctx, 7, CreateSessionRecordingInput{
		DeviceID:  "device-7",
		SessionID: "session-1",
		Status:    "recording",
		Format:    "webm",
	}, func(row *models.SessionRecording) Event {
		return Event{
			Type:     EventRecordingCreated,
			UserID:   row.UserID,
			DeviceID: row.DeviceID,
			Data:     map[string]interface{}{"recording_id": row.ID},
		}
	})
	if err != nil {
		t.Fatalf("CreateWithEvent: %v", err)
	}

	status := "completed"
	updated, err := recordings.UpdateWithEvent(ctx, 7, created.ID, UpdateSessionRecordingInput{
		Status: &status,
	}, func(row *models.SessionRecording) Event {
		return Event{
			Type:     EventRecordingUpdated,
			UserID:   row.UserID,
			DeviceID: row.DeviceID,
			Data:     map[string]interface{}{"recording_id": row.ID, "status": row.Status},
		}
	})
	if err != nil {
		t.Fatalf("UpdateWithEvent: %v", err)
	}
	if updated.Status != "completed" {
		t.Fatalf("updated status = %q", updated.Status)
	}

	err = recordings.DeleteWithEvent(ctx, 7, created.ID, func(row *models.SessionRecording) Event {
		return Event{
			Type:     EventRecordingDeleted,
			UserID:   row.UserID,
			DeviceID: row.DeviceID,
			Data:     map[string]interface{}{"recording_id": row.ID},
		}
	})
	if err != nil {
		t.Fatalf("DeleteWithEvent: %v", err)
	}

	var rows []models.EventOutbox
	if err := db.Order("created_at, id").Find(&rows).Error; err != nil {
		t.Fatalf("load outbox: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("outbox rows = %d, want created/updated/deleted", len(rows))
	}
	types := map[string]bool{}
	for i := range rows {
		var event Event
		if err := json.Unmarshal([]byte(rows[i].Payload), &event); err != nil {
			t.Fatalf("decode event: %v", err)
		}
		types[event.Type] = true
		if event.UserID != 7 || event.DeviceID != "device-7" {
			t.Fatalf("event routing = %+v", event)
		}
	}
	for _, typ := range []string{EventRecordingCreated, EventRecordingUpdated, EventRecordingDeleted} {
		if !types[typ] {
			t.Fatalf("missing %s in event types %+v", typ, types)
		}
	}
}

func TestRecordingOutboxFailureRollsBackCreate(t *testing.T) {
	db := newEventOutboxTestDB(t)
	recordings := NewSessionRecordingService(db).WithOutbox(
		&failingScheduledTaskOutbox{err: errors.New("outbox unavailable")},
	)
	_, err := recordings.CreateWithEvent(context.Background(), 7, CreateSessionRecordingInput{
		DeviceID: "device-7",
		Status:   "recording",
	}, func(row *models.SessionRecording) Event {
		return Event{
			Type:     EventRecordingCreated,
			UserID:   row.UserID,
			DeviceID: row.DeviceID,
		}
	})
	if err == nil {
		t.Fatal("CreateWithEvent succeeded despite outbox failure")
	}
	var count int64
	if err := db.Model(&models.SessionRecording{}).Count(&count).Error; err != nil {
		t.Fatalf("count recordings: %v", err)
	}
	if count != 0 {
		t.Fatalf("recordings = %d, want rollback to 0", count)
	}
}

func seedOutboxTestDevice(t *testing.T, db *gorm.DB, deviceID string, userID uint) {
	t.Helper()
	if err := db.Create(&models.Device{
		DeviceID:   deviceID,
		DeviceUUID: "uuid-" + deviceID,
		UserID:     &userID,
	}).Error; err != nil {
		t.Fatalf("create device: %v", err)
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
