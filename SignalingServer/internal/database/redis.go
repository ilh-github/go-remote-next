package database

import (
	"context"
	"fmt"
	"log"
	"quickdesk/signaling/internal/config"
	"sync"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type RedisRuntime struct {
	Client *redis.Client

	server    *miniredis.Miniredis
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	closeErr  error
}

// InitRedisRuntime selects an external Redis or an in-process compatible
// server. The embedded server is process-local and intentionally ephemeral;
// SQLite data survives restart, while tokens, presence and replay streams do
// not.
func InitRedisRuntime(cfg *config.Config) (*RedisRuntime, error) {
	driver := cfg.Redis.Driver
	if driver == "" || driver == "redis" || driver == "external" {
		client := redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr(),
			Password: cfg.Redis.Password,
			DB:       0,
		})
		if err := client.Ping(context.Background()).Err(); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("connect to Redis: %w", err)
		}
		log.Printf("Successfully connected to Redis at %s", cfg.Redis.Addr())
		return &RedisRuntime{Client: client}, nil
	}
	if driver != "embedded" {
		return nil, fmt.Errorf("unsupported REDIS_DRIVER %q", driver)
	}

	server, err := miniredis.Run()
	if err != nil {
		return nil, fmt.Errorf("start embedded Redis: %w", err)
	}
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	if err := client.Ping(context.Background()).Err(); err != nil {
		server.Close()
		_ = client.Close()
		return nil, fmt.Errorf("connect to embedded Redis: %w", err)
	}

	runtime := &RedisRuntime{
		Client: client,
		server: server,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	go runtime.advanceClock()
	log.Printf("Embedded Redis started at %s (single-process, ephemeral state)", server.Addr())
	return runtime, nil
}

func (r *RedisRuntime) advanceClock() {
	defer close(r.done)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-r.stop:
			return
		case <-ticker.C:
			r.server.FastForward(100 * time.Millisecond)
		}
	}
}

func (r *RedisRuntime) Close() error {
	if r == nil {
		return nil
	}
	r.closeOnce.Do(func() {
		if r.stop != nil {
			close(r.stop)
		}
		if r.done != nil {
			<-r.done
		}
		if r.Client != nil {
			r.closeErr = r.Client.Close()
		}
		if r.server != nil {
			r.server.Close()
		}
	})
	return r.closeErr
}
