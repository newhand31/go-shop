package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/newhand31/go-shop/internal/service"
)

// Payments is the service capability the callback route needs, defined by the consumer.
type Payments interface {
	HandleECPayCallback(ctx context.Context, params url.Values) error
}

// maxCallbackBytes bounds the body we are willing to parse. A real callback is well under 2 KB.
const maxCallbackBytes = 16 << 10

// ECPayCallback serves POST /payments/ecpay/callback.
//
// It is a plain chi route and not part of the OpenAPI contract: the caller is ECPay's server, not our frontend, and it
// sends a form (application/x-www-form-urlencoded) and expects the literal text "1|OK" back, not JSON.
func ECPayCallback(payments Payments, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxCallbackBytes)
		if err := r.ParseForm(); err != nil {
			reply(w, http.StatusBadRequest, "0|bad form")
			return
		}
		err := payments.HandleECPayCallback(r.Context(), r.PostForm)
		switch {
		case err == nil:
			reply(w, http.StatusOK, "1|OK")
		case errors.Is(err, service.ErrInvalidCallback), errors.Is(err, service.ErrPaymentMismatch):
			logger.Warn("payment callback rejected", "err", err)
			reply(w, http.StatusBadRequest, "0|rejected")
		default:
			// Our fault (database down): a non-"1|OK" reply makes ECPay resend, which is what we want.
			logger.Error("payment callback failed", "err", err)
			reply(w, http.StatusInternalServerError, "0|error")
		}
	}
}

func reply(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// ECPayCallbackPath is where ECPay's server posts its payment notifications.
const ECPayCallbackPath = "/payments/ecpay/callback"

// WithPaymentCallback puts the callback route in front of the API handler. It sits outside the rate limiter and the
// session middleware on purpose: the caller is ECPay's server, which has no cookie and must not be throttled as a visitor.
// Its authentication is the CheckMacValue signature, checked in the service.
func WithPaymentCallback(api http.Handler, payments Payments, logger *slog.Logger, middleware ...func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.With(RequestLog(logger), Recover(logger)).With(middleware...).Post(ECPayCallbackPath, ECPayCallback(payments, logger))
	r.Mount("/", api)
	return r
}
