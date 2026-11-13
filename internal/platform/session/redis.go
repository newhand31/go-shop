package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Redis keeps session ids in Redis, so every API instance sees the same sessions and a restart loses nothing.
// Redis expires the keys itself, which removes the sweep the in-memory store needs.
//
// Like Memory, it has the methods the handler's SessionStore interface lists and no declaration saying so.
type Redis struct {
	rdb *goredis.Client
	ttl time.Duration
}

// NewRedis returns a store whose sessions expire after ttl without use.
func NewRedis(rdb *goredis.Client, ttl time.Duration) *Redis {
	return &Redis{rdb: rdb, ttl: ttl}
}

func sessionKey(id string) string { return "session:" + id }

// Create makes a new session and returns its id (128 random bits from crypto/rand).
func (r *Redis) Create(ctx context.Context) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	id := hex.EncodeToString(raw)
	if err := r.rdb.Set(ctx, sessionKey(id), "1", r.ttl).Err(); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return id, nil
}

// Exists reports whether id is a live session, and extends it if so (EXPIRE returns false for a missing key).
func (r *Redis) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := r.rdb.Expire(ctx, sessionKey(id), r.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("check session: %w", err)
	}
	return ok, nil
}
