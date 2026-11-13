// Package session stores session ids. The in-memory store is enough for one instance and is lost on restart;
// the handler only needs the two methods of its own SessionStore interface, so another store can replace it later.
package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// Memory keeps session ids in a map with a sliding expiry: every successful Exists extends the session.
type Memory struct {
	ttl time.Duration
	now func() time.Time

	mu       sync.Mutex
	sessions map[string]time.Time // id → expiry
	lastGC   time.Time
}

// NewMemory returns a store whose sessions expire after ttl without use.
func NewMemory(ttl time.Duration) *Memory {
	return &Memory{ttl: ttl, now: time.Now, sessions: make(map[string]time.Time)}
}

// Create makes a new session and returns its id: 128 random bits from crypto/rand, so ids cannot be guessed.
func (m *Memory) Create(_ context.Context) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	id := hex.EncodeToString(raw)

	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	if now.Sub(m.lastGC) >= m.ttl/10 { // sweep at most every ttl/10: one O(n) walk per Create would be a CPU sink
		m.lastGC = now
		m.gc(now)
	}
	m.sessions[id] = now.Add(m.ttl)
	return id, nil
}

// Exists reports whether id is a live session, and extends it if so.
func (m *Memory) Exists(_ context.Context, id string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	exp, ok := m.sessions[id]
	if !ok || !now.Before(exp) {
		delete(m.sessions, id)
		return false, nil
	}
	m.sessions[id] = now.Add(m.ttl)
	return true, nil
}

// gc removes expired sessions so abandoned ones do not pile up. Caller holds m.mu.
// It walks the whole map, which is fine at this size; a real deployment would let the store expire keys itself.
func (m *Memory) gc(now time.Time) {
	for id, exp := range m.sessions {
		if !now.Before(exp) {
			delete(m.sessions, id)
		}
	}
}
