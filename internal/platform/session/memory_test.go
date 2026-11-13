package session

import (
	"context"
	"testing"
	"time"
)

func TestMemory(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 11, 20, 31, 0, 0, time.UTC)
	m := NewMemory(10 * time.Minute)
	m.now = func() time.Time { return now }

	id, err := m.Create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 32 {
		t.Fatalf("id length = %d, want 32 hex chars", len(id))
	}

	// The steps run in order on one session: each advances the clock, then asks.
	tests := []struct {
		name    string
		advance time.Duration
		id      string
		want    bool
	}{
		{"unknown id", 0, "nope", false},
		{"fresh session", time.Minute, id, true},
		{"sliding expiry: alive 9 min after last use", 9 * time.Minute, id, true},
		{"expired after ttl without use", 11 * time.Minute, id, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now = now.Add(tt.advance)
			got, err := m.Exists(ctx, tt.id)
			if err != nil || got != tt.want {
				t.Fatalf("Exists = %v, %v; want %v", got, err, tt.want)
			}
		})
	}
}

func TestCreateIDsAreUnique(t *testing.T) {
	m := NewMemory(time.Minute)
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id, err := m.Create(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if seen[id] {
			t.Fatalf("duplicate id %s", id)
		}
		seen[id] = true
	}
}
