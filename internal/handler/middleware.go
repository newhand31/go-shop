package handler

import (
	"context"
	"log/slog"
	"math"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/newhand31/go-shop/internal/api"
)

// A middleware is a function that takes a handler and returns a handler: func(http.Handler) http.Handler.
// It wraps the next one, so it can run code before it, after it, or instead of it. A chain is just wrapping repeated:
// chi's r.Use(a, b, c) builds a(b(c(router))), so a sees the request first and the response last.
// Compare ASP.NET Core's ActionFilter/middleware pipeline or a Servlet Filter's doFilter(req, res, chain).

// Limiter is what RateLimit needs, defined here by the consumer. Anything with this one method satisfies it.
type Limiter interface {
	Allow(key string) (ok bool, retryAfter time.Duration)
}

// SessionStore is what Session needs from a store of session ids.
type SessionStore interface {
	Create(ctx context.Context) (id string, err error)
	Exists(ctx context.Context, id string) (bool, error)
}

// Recover turns a panic in any later handler into a 500 api.Error instead of a dropped connection.
// Without it, net/http recovers the panic for you but closes the connection and logs plain text: the client sees nothing.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				if rec == http.ErrAbortHandler {
					panic(rec) // the sentinel net/http itself uses to abort a response; let it through
				}
				// If the handler already wrote part of a response, this JSON lands after it; acceptable here.
				logger.ErrorContext(r.Context(), "panic recovered", "panic", rec, "stack", string(debug.Stack()))
				writeJSON(w, logger, http.StatusInternalServerError, api.Error{Code: "internal_error", Message: "internal error"})
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit answers 429 once a client IP has used up its tokens, and tells it when to come back via Retry-After.
// The key is the connection's remote address. X-Forwarded-For is deliberately ignored: any client can write that
// header, so trusting it lets an attacker pick a fresh "IP" per request. Behind a proxy you trust, that is a
// deployment decision for a later episode.
func RateLimit(limiter Limiter, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok, retryAfter := limiter.Allow(clientIP(r))
			if !ok {
				secs := int(math.Ceil(retryAfter.Seconds()))
				w.Header().Set("Retry-After", strconv.Itoa(max(secs, 1)))
				writeJSON(w, logger, http.StatusTooManyRequests, api.Error{Code: "rate_limited", Message: "too many requests"})
				return // not calling next is how a middleware says "stop here"
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type ctxKey struct{}

// SessionID returns the session id Session put on the request context, or "" if there is none.
func SessionID(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

const sessionCookie = "sid"

// Session gives every visitor a session id: reuse the cookie if the store knows it, otherwise create one.
// Later handlers read it with SessionID(ctx). The context is how a middleware hands a value to the code after it;
// the key is an unexported type, so no other package can collide with it.
func Session(store SessionStore, ttl time.Duration, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var id string
			if c, err := r.Cookie(sessionCookie); err == nil {
				ok, err := store.Exists(r.Context(), c.Value)
				if err != nil {
					writeError(w, logger, err)
					return
				}
				if ok {
					id = c.Value
				}
			}
			if id == "" {
				var err error
				if id, err = store.Create(r.Context()); err != nil {
					writeError(w, logger, err)
					return
				}
				// The cookie lives ttl from creation (absolute); the store slides its own expiry. After ttl the
				// browser drops the cookie and the next request simply starts a new session.
				http.SetCookie(w, &http.Cookie{
					Name:     sessionCookie,
					Value:    id,
					Path:     "/",
					MaxAge:   int(ttl.Seconds()),
					HttpOnly: true, // JavaScript cannot read it, so an XSS bug cannot steal it
					SameSite: http.SameSiteLaxMode,
					// Secure is left off so plain http://localhost works; turn it on behind TLS.
				})
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, id)))
		})
	}
}
