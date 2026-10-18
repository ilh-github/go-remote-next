package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type captureSubscriber struct {
	events chan Event
}

func (s *captureSubscriber) Name() string { return "test-capture" }

func (s *captureSubscriber) HandleEvent(_ context.Context, evt Event) {
	s.events <- evt
}

func TestEventBusPublishesRecordingLifecycleEvent(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	bus := NewEventBus(client)
	subscriber := &captureSubscriber{events: make(chan Event, 1)}
	bus.Subscribe(subscriber)

	bus.Publish(context.Background(), Event{
		Type:     EventRecordingCreated,
		UserID:   42,
		DeviceID: "123456789",
		Data: map[string]interface{}{
			"recording_id": 1,
			"session_id":   "session-1",
		},
	})

	select {
	case evt := <-subscriber.events:
		if evt.Type != EventRecordingCreated {
			t.Fatalf("type = %q, want %q", evt.Type, EventRecordingCreated)
		}
		if evt.UserID != 42 || evt.DeviceID != "123456789" {
			t.Fatalf("unexpected event routing: %+v", evt)
		}
		if evt.ServerRev == 0 {
			t.Fatal("expected server_rev to be assigned")
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive event")
	}

	stream, err := client.XRange(context.Background(), "qd:events:user:42", "-", "+").Result()
	if err != nil {
		t.Fatalf("read user event stream: %v", err)
	}
	if len(stream) != 1 {
		t.Fatalf("stream entries = %d, want 1", len(stream))
	}
	if stream[0].Values["e"] == nil {
		t.Fatalf("stream entry missing payload: %+v", stream[0])
	}
}

func TestEventBusUsesConfiguredReplayRetention(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	bus := NewEventBusWithRetention(client, EventBusRetention{
		MaxLen: 2,
		TTL:    10 * time.Minute,
	})
	for index := 0; index < 3; index++ {
		bus.Publish(context.Background(), Event{
			Type:     EventRecordingCreated,
			UserID:   42,
			DeviceID: "123456789",
			Data: map[string]interface{}{
				"recording_id": index,
			},
		})
	}

	streamKey := "qd:events:user:42"
	stream, err := client.XRange(context.Background(), streamKey, "-", "+").Result()
	if err != nil {
		t.Fatalf("read configured user event stream: %v", err)
	}
	if len(stream) != 2 {
		t.Fatalf("stream entries = %d, want 2 after configured trim", len(stream))
	}
	ttl, err := client.TTL(context.Background(), streamKey).Result()
	if err != nil {
		t.Fatalf("read configured stream TTL: %v", err)
	}
	if ttl != 10*time.Minute {
		t.Fatalf("stream TTL = %s, want 10m", ttl)
	}

	redisServer.FastForward(11 * time.Minute)
	exists, err := client.Exists(context.Background(), streamKey).Result()
	if err != nil {
		t.Fatalf("check expired stream: %v", err)
	}
	if exists != 0 {
		t.Fatalf("stream still exists after configured TTL, exists=%d", exists)
	}
}

func TestEventBusRetryUsesConfiguredReplayRetention(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	bus := NewEventBusWithRetention(client, EventBusRetention{
		MaxLen: 1,
		TTL:    20 * time.Minute,
	})
	streamKey := "qd:events:user:77"
	_, err := client.XAdd(context.Background(), &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]interface{}{
			"e": `{"id":"evt-old","type":"device.online.changed","user_id":77,"server_rev":8}`,
		},
	}).Result()
	if err != nil {
		t.Fatalf("seed user event stream: %v", err)
	}

	retryEvent := Event{
		ID:        "evt-retry",
		Type:      EventDeviceOnlineChanged,
		UserID:    77,
		ServerRev: 9,
	}
	payload, err := json.Marshal(retryEvent)
	if err != nil {
		t.Fatalf("marshal retry event: %v", err)
	}
	if err := client.LPush(context.Background(), eventRetryKey, string(payload)).Err(); err != nil {
		t.Fatalf("seed retry list: %v", err)
	}
	bus.drainRetry(context.Background())

	stream, err := client.XRange(context.Background(), streamKey, "-", "+").Result()
	if err != nil {
		t.Fatalf("read retried user event stream: %v", err)
	}
	if len(stream) != 1 {
		t.Fatalf("retried stream entries = %d, want configured maxlen 1", len(stream))
	}
	if !strings.Contains(stream[0].Values["e"].(string), `"id":"evt-retry"`) {
		t.Fatalf("retried stream did not keep the newest event: %+v", stream[0])
	}
	ttl, err := client.TTL(context.Background(), streamKey).Result()
	if err != nil {
		t.Fatalf("read retried stream TTL: %v", err)
	}
	if ttl != 20*time.Minute {
		t.Fatalf("retried stream TTL = %s, want 20m", ttl)
	}
}

func TestEventBusRecoversProcessingFromDeadRetryWorker(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	retryEvent := Event{
		ID:        "evt-recovered",
		Type:      EventDeviceOnlineChanged,
		UserID:    88,
		ServerRev: 12,
	}
	payload, err := json.Marshal(retryEvent)
	if err != nil {
		t.Fatalf("marshal retry event: %v", err)
	}
	const registryKey = "qd:events:retry:workers"
	const processingKey = "qd:events:retry:processing:dead-worker"
	if err := client.SAdd(context.Background(), registryKey, "dead-worker").Err(); err != nil {
		t.Fatalf("seed dead retry worker registry: %v", err)
	}
	if err := client.LPush(context.Background(), processingKey, string(payload)).Err(); err != nil {
		t.Fatalf("seed dead retry worker processing list: %v", err)
	}

	bus := NewEventBus(client)
	bus.recoverStaleRetryWorkers(context.Background())

	pending, err := client.LLen(context.Background(), eventRetryKey).Result()
	if err != nil {
		t.Fatalf("read recovered retry queue: %v", err)
	}
	if pending != 1 {
		t.Fatalf("recovered retry queue length = %d, want 1", pending)
	}
	processing, err := client.LLen(context.Background(), processingKey).Result()
	if err != nil {
		t.Fatalf("read dead retry processing list: %v", err)
	}
	if processing != 0 {
		t.Fatalf("dead retry processing list length = %d, want 0", processing)
	}
	registered, err := client.SIsMember(context.Background(), registryKey, "dead-worker").Result()
	if err != nil {
		t.Fatalf("check dead retry worker registry: %v", err)
	}
	if registered {
		t.Fatal("dead retry worker remained registered")
	}

	bus.drainRetry(context.Background())
	stream, err := client.XRange(context.Background(), "qd:events:user:88", "-", "+").Result()
	if err != nil {
		t.Fatalf("read recovered user stream: %v", err)
	}
	if len(stream) != 1 {
		t.Fatalf("recovered user stream entries = %d, want 1", len(stream))
	}
}

func TestEventBusRetryDrainRecoversItsOwnProcessing(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	bus := NewEventBus(client)
	retryEvent := Event{
		ID:        "evt-own-processing",
		Type:      EventDeviceOnlineChanged,
		UserID:    89,
		ServerRev: 13,
	}
	payload, err := json.Marshal(retryEvent)
	if err != nil {
		t.Fatalf("marshal retry event: %v", err)
	}
	processingKey := retryProcessingKey(bus.retryWorker)
	if err := client.LPush(context.Background(), processingKey, string(payload)).Err(); err != nil {
		t.Fatalf("seed own retry processing list: %v", err)
	}

	bus.drainRetry(context.Background())

	stream, err := client.XRange(context.Background(), "qd:events:user:89", "-", "+").Result()
	if err != nil {
		t.Fatalf("read own recovered user stream: %v", err)
	}
	if len(stream) != 1 {
		t.Fatalf("own recovered user stream entries = %d, want 1", len(stream))
	}
	processing, err := client.LLen(context.Background(), processingKey).Result()
	if err != nil {
		t.Fatalf("read own retry processing list: %v", err)
	}
	if processing != 0 {
		t.Fatalf("own retry processing list length = %d, want 0", processing)
	}
}

func TestEventBusDoesNotStealProcessingFromLiveRetryWorker(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	const registryKey = "qd:events:retry:workers"
	const workerID = "live-worker"
	const processingKey = "qd:events:retry:processing:live-worker"
	if err := client.SAdd(context.Background(), registryKey, workerID).Err(); err != nil {
		t.Fatalf("seed live retry worker registry: %v", err)
	}
	if err := client.Set(
		context.Background(),
		retryHeartbeatKey(workerID),
		time.Now().UTC().Format(time.RFC3339Nano),
		eventRetryHeartbeatTTL,
	).Err(); err != nil {
		t.Fatalf("seed live retry worker heartbeat: %v", err)
	}
	if err := client.LPush(
		context.Background(),
		processingKey,
		`{"id":"evt-live","type":"device.online.changed","user_id":90}`,
	).Err(); err != nil {
		t.Fatalf("seed live retry worker processing list: %v", err)
	}

	bus := NewEventBus(client)
	bus.recoverStaleRetryWorkers(context.Background())

	processing, err := client.LLen(context.Background(), processingKey).Result()
	if err != nil {
		t.Fatalf("read live retry processing list: %v", err)
	}
	if processing != 1 {
		t.Fatalf("live retry processing list length = %d, want 1", processing)
	}
	pending, err := client.LLen(context.Background(), eventRetryKey).Result()
	if err != nil {
		t.Fatalf("read pending retry queue: %v", err)
	}
	if pending != 0 {
		t.Fatalf("pending retry queue length = %d, want 0", pending)
	}
}
