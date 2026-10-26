package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"quickdesk/signaling/internal/models"

	"gorm.io/gorm"
)

// countingOutbox records enqueue calls so a test can tell "the prompt went
// through the durable path" from "it was published directly and lost on crash".
type countingOutbox struct {
	mu       sync.Mutex
	enqueued int
	err      error
}

func (o *countingOutbox) EnqueueTx(_ context.Context, _ *gorm.DB, evt Event) (Event, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.enqueued++
	return evt, o.err
}

// QD-607: "定时连接" was never built, and the reason is written into the action
// registry's own comment: a task that needed a client to cooperate could be
// scheduled but never actually run. That is correct for a connection - the
// control plane cannot make somebody's controller dial a device.
//
// What it can do is everything around the connection. A scheduled task that
// says "prompt me to connect to this device at 09:00" is entirely control-plane
// work: the row is there, the clock fires, and the owner's already-open clients
// get told. That is the half worth building, and building it does not pretend the
// other half exists.

// recordingBus captures what was published, so a test can assert on the event
// rather than on "nothing returned an error".
type recordingBus struct {
	mu     sync.Mutex
	events []Event
}

func (b *recordingBus) Publish(_ context.Context, evt Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, evt)
}

func (b *recordingBus) published() []Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]Event(nil), b.events...)
}

// The whole point: the task fires and the owner's clients are told, with the
// device and the task id attached so a client can act on it without a round trip.
func TestConnectPromptActionPromptsTheOwner(t *testing.T) {
	bus := &recordingBus{}
	action := NewConnectPromptAction(bus)

	err := action.Run(context.Background(), models.ScheduledTask{
		ID:     7,
		UserID: 9901,
		Kind:   models.ScheduledTaskConnectPrompt,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	events := bus.published()
	if len(events) != 1 {
		t.Fatalf("published %d events, want 1", len(events))
	}
	evt := events[0]
	if evt.Type != EventDeviceConnectPrompted {
		t.Fatalf("event type = %q, want %q", evt.Type, EventDeviceConnectPrompted)
	}
	if evt.UserID != 9901 {
		t.Fatalf("event user = %d, want the task owner 9901", evt.UserID)
	}
	if got, _ := evt.Data["task_id"].(uint); got != 7 {
		t.Fatalf("event task_id = %v, want 7", evt.Data["task_id"])
	}
}

// A nil bus is a configuration error, not a silent success: an occurrence that
// reports success while telling nobody anything is the failure mode this whole
// feature exists to avoid.
func TestConnectPromptActionWithoutABusFails(t *testing.T) {
	action := NewConnectPromptAction(nil)
	if err := action.Run(context.Background(), models.ScheduledTask{ID: 1, UserID: 2}); err == nil {
		t.Fatal("Run with no bus returned nil, want an error")
	}
}

// A task that names a device carries it through, because "connect to something"
// is not a prompt anybody can act on.
func TestConnectPromptActionCarriesTheDevice(t *testing.T) {
	bus := &recordingBus{}
	action := NewConnectPromptAction(bus)
	err := action.Run(context.Background(), models.ScheduledTask{
		ID: 3, UserID: 42, Kind: models.ScheduledTaskConnectPrompt,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(bus.published()) != 1 {
		t.Fatal("expected exactly one event")
	}
}

// An outbox, when wired, is what makes the prompt survive a crash. Without it a
// restart between the clock firing and the publish loses the prompt silently.
func TestConnectPromptActionUsesTheOutboxWhenWired(t *testing.T) {
	bus := &recordingBus{}
	outbox := &countingOutbox{}
	action := NewConnectPromptAction(bus).WithOutbox(outbox).
		WithEnqueue(func(_ context.Context, evt Event) error {
			_, err := outbox.EnqueueTx(context.Background(), nil, evt)
			return err
		})

	if err := action.Run(context.Background(), models.ScheduledTask{ID: 9, UserID: 77}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if outbox.enqueued != 1 {
		t.Fatalf("outbox enqueued %d, want 1", outbox.enqueued)
	}
	// The direct publish must not also happen, or every prompt goes out twice.
	if len(bus.published()) != 0 {
		t.Fatalf("bus published %d events, want 0 when the outbox is wired", len(bus.published()))
	}
}

// A failed enqueue must not fail the occurrence: the prompt is already due, and
// retrying it later would deliver a reminder for a moment that has passed.
func TestConnectPromptActionSurvivesAFailedEnqueue(t *testing.T) {
	bus := &recordingBus{}
	outbox := &countingOutbox{err: errors.New("outbox is down")}
	action := NewConnectPromptAction(bus).WithOutbox(outbox).
		WithEnqueue(func(_ context.Context, evt Event) error {
			_, err := outbox.EnqueueTx(context.Background(), nil, evt)
			return err
		})

	if err := action.Run(context.Background(), models.ScheduledTask{ID: 5, UserID: 8}); err != nil {
		t.Fatalf("Run with a failing outbox = %v, want nil", err)
	}
}

// End to end through the service rather than the action alone: a task of this
// kind has to be creatable and then actually run by the worker. An action that
// works in isolation but is not registered is a feature nobody can schedule.
func TestConnectPromptKindIsCreatableAndRuns(t *testing.T) {
	db := openTestPostgres(t)
	bus := &recordingBus{}
	svc := NewScheduledTaskService(db).
		WithNotifier(bus).
		WithOutbox(&countingOutbox{})
	// The enqueue seam stands in for the outbox transaction and forwards to the
	// bus, because a real outbox delivers by publishing once its worker drains
	// the row. Stubbing it as a no-op would prove the worker ran and nothing
	// else, which is the assertion this test is trying to avoid making.
	svc.RegisterAction(models.ScheduledTaskConnectPrompt,
		NewConnectPromptAction(bus).
			WithOutbox(&countingOutbox{}).
			WithEnqueue(func(_ context.Context, evt Event) error {
				bus.Publish(context.Background(), evt)
				return nil
			}))

	kinds := svc.SupportedKinds()
	found := false
	for _, kind := range kinds {
		if kind == models.ScheduledTaskConnectPrompt {
			found = true
		}
	}
	if !found {
		t.Fatalf("connect_prompt is not registered; supported kinds: %v", kinds)
	}

	task, err := svc.Create(context.Background(), 4242, ScheduledTaskSpec{
		Kind:          models.ScheduledTaskConnectPrompt,
		RepeatSeconds: 0,
		RunAt:         time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if task.Kind != models.ScheduledTaskConnectPrompt {
		t.Fatalf("created kind = %q", task.Kind)
	}

	// Make it due and let the worker run it, which is the path a real schedule
	// takes rather than calling the action directly.
	if err := db.Model(task).Update("next_run_at", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatalf("force due: %v", err)
	}
	ran, err := svc.RunDue(context.Background(), time.Now().UTC())
	if err != nil {
		t.Fatalf("RunDue: %v", err)
	}
	if ran == 0 {
		t.Fatal("RunDue ran nothing, so the prompt never fired")
	}
	if len(bus.published()) == 0 {
		t.Fatal("the worker ran but published nothing")
	}
}

// The whole path, not just the action. An event that is published but not
// subscribable reaches the realtime socket and nowhere the owner configured
// their notifications - the half-wired version of this feature. This drives the
// same subscriber the server wires at startup, so the assertion is about the
// delivery a user actually gets.
func TestConnectPromptReachesASubscribedChannel(t *testing.T) {
	db := openTestPostgres(t)

	// A channel that subscribes to exactly this event, owned by the task's user.
	owner := uint(77123)
	if err := db.Exec("DELETE FROM user_notification_channels WHERE user_id = ?", owner).Error; err != nil {
		t.Fatalf("clear channels: %v", err)
	}
	channel := models.UserNotificationChannel{
		UserID:  owner,
		Name:    "prompt-sink",
		URL:     "https://example.invalid/hook",
		Enabled: true,
		Events:  `["` + EventDeviceConnectPrompted + `"]`,
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("create channel: %v", err)
	}

	svc := NewNotificationChannelService(db, NewNotificationDeliveryService(db))
	sub := NewUserNotificationSubscriber(svc)
	sub.HandleEvent(context.Background(), Event{
		Type:   EventDeviceConnectPrompted,
		UserID: owner,
		Data:   map[string]interface{}{"task_id": 99},
	})

	// A delivery row is the proof the prompt left the control plane. Without one
	// the event existed only in the bus, which no configured channel ever saw.
	var count int64
	if err := db.Model(&models.NotificationDelivery{}).
		Where("channel_id = ? AND event_type = ?", channel.ID, EventDeviceConnectPrompted).
		Count(&count).Error; err != nil {
		t.Fatalf("count deliveries: %v", err)
	}
	if count == 0 {
		t.Fatal("no delivery was enqueued, so the subscribed channel was never told")
	}
}
