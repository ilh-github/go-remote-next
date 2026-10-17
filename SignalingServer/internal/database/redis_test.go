package database

import (
	"context"
	"testing"
	"time"

	"quickdesk/signaling/internal/config"

	"github.com/redis/go-redis/v9"
)

func TestEmbeddedRedisExpiresKeysAndPublishes(t *testing.T) {
	runtime, err := InitRedisRuntime(&config.Config{
		Redis: config.RedisConfig{Driver: "embedded"},
	})
	if err != nil {
		t.Fatalf("InitRedisRuntime() error = %v", err)
	}
	t.Cleanup(func() { _ = runtime.Close() })

	ctx := context.Background()
	if err := runtime.Client.Set(ctx, "short-lived", "value", 250*time.Millisecond).Err(); err != nil {
		t.Fatalf("SET short-lived: %v", err)
	}
	if got, err := runtime.Client.Get(ctx, "short-lived").Result(); err != nil || got != "value" {
		t.Fatalf("GET short-lived = %q, %v; want value, nil", got, err)
	}

	sub := runtime.Client.Subscribe(ctx, "minimal-events")
	t.Cleanup(func() { _ = sub.Close() })
	if _, err := sub.Receive(ctx); err != nil {
		t.Fatalf("subscribe handshake: %v", err)
	}
	if err := runtime.Client.Publish(ctx, "minimal-events", "payload").Err(); err != nil {
		t.Fatalf("PUBLISH: %v", err)
	}
	select {
	case msg := <-sub.Channel():
		if msg.Payload != "payload" {
			t.Fatalf("published payload = %q, want payload", msg.Payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for embedded Redis pub/sub message")
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		_, err = runtime.Client.Get(ctx, "short-lived").Result()
		if err == redis.Nil {
			break
		}
		if err != nil {
			t.Fatalf("GET short-lived while waiting for expiry: %v", err)
		}
		if time.Now().After(deadline) {
			t.Fatal("embedded Redis key did not expire")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
