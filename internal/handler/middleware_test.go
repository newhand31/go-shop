package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/newhand31/go-shop/internal/api"
)

// fakeLimiter records the key it was asked about and answers with a canned decision.
type fakeLimiter struct {
	ok         bool
	retryAfter time.Duration
	gotKey     string
}

func (f *fakeLimiter) Allow(key string) (bool, time.Duration) {
	f.gotKey = key
	return f.ok, f.retryAfter
}

// fakeStore is a SessionStore with a fixed set of live ids.
type fakeStore struct {
	live      map[string]bool
	createErr error
	created   int
}

func (f *fakeStore) Create(context.Context) (string, error) {
	if f.createErr != nil {
		return "", f.createErr
	}
	f.created++
	return "new-id", nil
}

func (f *fakeStore) Exists(_ context.Context, id string) (bool, error) { return f.live[id], nil }

func ok200(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) api.Error {
	t.Helper()
	dec := json.NewDecoder(rec.Body)
	dec.DisallowUnknownFields()
	var e api.Error
	if err := dec.Decode(&e); err != nil {
		t.Fatalf("body is not an api.Error: %v", err)
	}
	return e
}

func TestRecover(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	h := Recover(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom: postgres://user:secret@db/shop")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if e := decodeError(t, rec); e.Code != "internal_error" || e.Message != "internal error" {
		t.Fatalf("error = %+v", e)
	}
	if strings.Contains(rec.Body.String(), "secret") {
		t.Fatal("panic value leaked into the response")
	}
	if !strings.Contains(logs.String(), "panic recovered") {
		t.Fatal("panic was not logged")
	}
}

func TestRecoverLetsAbortHandlerThrough(t *testing.T) {
	h := Recover(quietLogger())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))
	defer func() {
		if rec := recover(); rec != http.ErrAbortHandler {
			t.Fatalf("recovered %v, want http.ErrAbortHandler re-panicked", rec)
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	t.Fatal("expected a panic")
}

func TestRateLimit(t *testing.T) {
	tests := []struct {
		name           string
		limiter        *fakeLimiter
		remoteAddr     string
		forwardedFor   string
		wantStatus     int
		wantRetryAfter string
		wantKey        string
	}{
		{"allowed", &fakeLimiter{ok: true}, "203.0.113.7:51000", "", http.StatusOK, "", "203.0.113.7"},
		{"blocked", &fakeLimiter{retryAfter: 300 * time.Millisecond}, "203.0.113.7:51000", "", http.StatusTooManyRequests, "1", "203.0.113.7"},
		{"retry-after rounds up", &fakeLimiter{retryAfter: 2100 * time.Millisecond}, "203.0.113.7:51000", "", http.StatusTooManyRequests, "3", "203.0.113.7"},
		{"x-forwarded-for is ignored", &fakeLimiter{ok: true}, "203.0.113.7:51000", "198.51.100.1", http.StatusOK, "", "203.0.113.7"},
		{"ipv6 remote address", &fakeLimiter{ok: true}, "[2001:db8::1]:51000", "", http.StatusOK, "", "2001:db8::1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := RateLimit(tt.limiter, quietLogger())(http.HandlerFunc(ok200))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.forwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.forwardedFor)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get("Retry-After"); got != tt.wantRetryAfter {
				t.Fatalf("Retry-After = %q, want %q", got, tt.wantRetryAfter)
			}
			if tt.limiter.gotKey != tt.wantKey {
				t.Fatalf("limiter key = %q, want %q", tt.limiter.gotKey, tt.wantKey)
			}
			if tt.wantStatus == http.StatusTooManyRequests {
				if e := decodeError(t, rec); e.Code != "rate_limited" {
					t.Fatalf("code = %q, want rate_limited", e.Code)
				}
			}
		})
	}
}

func TestSession(t *testing.T) {
	tests := []struct {
		name        string
		cookie      string // "" means no cookie
		store       *fakeStore
		wantID      string
		wantSetID   string // "" means no Set-Cookie
		wantCreated int
		wantStatus  int
	}{
		{"no cookie creates a session", "", &fakeStore{}, "new-id", "new-id", 1, http.StatusOK},
		{"known cookie is reused", "abc", &fakeStore{live: map[string]bool{"abc": true}}, "abc", "", 0, http.StatusOK},
		{"unknown cookie is replaced", "stale", &fakeStore{}, "new-id", "new-id", 1, http.StatusOK},
		{"store failure is a 500", "", &fakeStore{createErr: errors.New("store down")}, "", "", 0, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var seen string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seen = SessionID(r.Context())
				w.WriteHeader(http.StatusOK)
			})
			h := Session(tt.store, time.Hour, quietLogger())(next)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: sessionCookie, Value: tt.cookie})
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if seen != tt.wantID {
				t.Fatalf("session id in context = %q, want %q", seen, tt.wantID)
			}
			if tt.store.created != tt.wantCreated {
				t.Fatalf("sessions created = %d, want %d", tt.store.created, tt.wantCreated)
			}
			cookies := rec.Result().Cookies()
			if tt.wantSetID == "" {
				if len(cookies) != 0 {
					t.Fatalf("unexpected Set-Cookie: %v", cookies)
				}
				return
			}
			if len(cookies) != 1 {
				t.Fatalf("cookies = %v, want exactly one", cookies)
			}
			c := cookies[0]
			if c.Name != sessionCookie || c.Value != tt.wantSetID || !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.MaxAge != 3600 {
				t.Fatalf("cookie = %+v", c)
			}
		})
	}
}

// TestChainOrder shows what "a(b(c(handler)))" means: the request goes in a → b → c, the response comes out c → b → a.
func TestChainOrder(t *testing.T) {
	var trail []string
	mark := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				trail = append(trail, name+" in")
				next.ServeHTTP(w, r)
				trail = append(trail, name+" out")
			})
		}
	}
	h := New(fakeProducts{}, fakeCarts{}, quietLogger(), mark("first"), mark("second"))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

	want := "first in,second in,second out,first out"
	if got := strings.Join(trail, ","); got != want {
		t.Fatalf("trail = %s, want %s", got, want)
	}
}

// A panic deep inside the router must come back as the same api.Error shape every other failure uses.
func TestPanicInsideRouterIsAnAPIError(t *testing.T) {
	boom := func(http.Handler) http.Handler {
		return http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("kaboom") })
	}
	rec := httptest.NewRecorder()
	New(fakeProducts{}, fakeCarts{}, quietLogger(), boom).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/products", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if e := decodeError(t, rec); e.Code != "internal_error" {
		t.Fatalf("code = %q", e.Code)
	}
}
