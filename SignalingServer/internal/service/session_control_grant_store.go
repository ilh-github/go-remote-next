package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// ControlGrantStore is the cross-instance half of QD-314's exclusive control.
//
// ControlGrants alone is a map inside one process, which is correct for a
// single-instance install and wrong for a fleet: two instances holding two
// controllers on the same exclusive machine would each answer "you may drive
// it", producing exactly the invisible fight the mode exists to prevent. The
// socket lives in one process, but the *holder* is a fact about the device, so
// it belongs somewhere every instance can read.
//
// Redis holds it for the same reason the revoke cooldown does: it is short-lived
// state that must expire on its own even if the process that wrote it dies, so
// a crashed instance cannot leave a machine reserved for a socket that is gone.
// A nil store is not an error - it means this install has no fleet, and the
// per-instance map is then the whole truth.
type ControlGrantStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewControlGrantStore(rdb *redis.Client, ttl time.Duration) *ControlGrantStore {
	return &ControlGrantStore{rdb: rdb, ttl: ttl}
}

// TTL is the configured window. Zero means the feature is off, which is what an
// operator who wants control to survive a restart-less fleet does not set.
func (s *ControlGrantStore) TTL() time.Duration {
	if s == nil {
		return 0
	}
	return s.ttl
}

func (s *ControlGrantStore) key(deviceID string) string {
	return "qd:sessions:control:" + strings.TrimSpace(deviceID)
}

// controlGrantRecord is the stored shape. The mode travels with the holder
// because a device whose policy says exclusive is still exclusive after the
// holder goes away: "free" is not the same as "shared", and reading a released
// machine as shared would hand control to whoever asked next without the owner
// ever changing the setting.
type controlGrantRecord struct {
	Mode   string `json:"mode"`
	Holder string `json:"holder"`
}

// Get reads the device's grant, or the shared default when nothing is stored.
//
// An empty holder is a meaningful answer, so the record is kept rather than
// deleted on release; the TTL is what eventually reclaims it.
func (s *ControlGrantStore) Get(ctx context.Context, deviceID string) (ControlGrant, error) {
	if s == nil || s.rdb == nil {
		return ControlGrant{Mode: ControlModeShared}, nil
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return ControlGrant{Mode: ControlModeShared}, nil
	}
	raw, err := s.rdb.Get(ctx, s.key(deviceID)).Result()
	if errors.Is(err, redis.Nil) {
		return ControlGrant{Mode: ControlModeShared}, nil
	}
	if err != nil {
		return ControlGrant{Mode: ControlModeShared}, err
	}
	var record controlGrantRecord
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		// A corrupt value is treated as "nothing recorded" rather than as an
		// exclusive machine nobody holds: the safe answer for a controller is
		// the one it already had before this store existed.
		return ControlGrant{Mode: ControlModeShared}, nil
	}
	return ControlGrant{
		Mode:   NormalizeControlMode(record.Mode),
		Holder: strings.TrimSpace(record.Holder),
	}, nil
}

// Put writes the grant with the configured TTL. An empty device is ignored: a
// key without a device would answer every device's question.
func (s *ControlGrantStore) Put(ctx context.Context, deviceID string, grant ControlGrant) error {
	if s == nil || s.rdb == nil {
		return nil
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil
	}
	raw, err := json.Marshal(controlGrantRecord{
		Mode:   NormalizeControlMode(grant.Mode),
		Holder: strings.TrimSpace(grant.Holder),
	})
	if err != nil {
		return err
	}
	ttl := s.ttl
	if ttl <= 0 {
		// Without a TTL the key would outlive every process that wrote it, and a
		// machine would stay reserved for a socket that is long gone.
		ttl = 5 * time.Minute
	}
	return s.rdb.Set(ctx, s.key(deviceID), raw, ttl).Err()
}

// Forget drops the record. The host going away is the case: every controller is
// about to be disconnected, so a holder would be a stale answer.
func (s *ControlGrantStore) Forget(ctx context.Context, deviceID string) error {
	if s == nil || s.rdb == nil {
		return nil
	}
	return s.rdb.Del(ctx, s.key(deviceID)).Err()
}
