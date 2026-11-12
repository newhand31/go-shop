package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/newhand31/go-shop/internal/api"
	"github.com/newhand31/go-shop/internal/service"
)

// writeJSON is the only place that writes a JSON body, so success and error responses share one code path.
func writeJSON(w http.ResponseWriter, logger *slog.Logger, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		logger.Error("write response failed", "err", err)
	}
}

// writeError is the single exit for every error response. Whatever went wrong, the client sees api.Error.
//
// The switch is the whole trick: errors.Is asks "is this error, anywhere in its %w chain, that sentinel?",
// errors.As asks "is there a value of this type in the chain, and give it to me".
func writeError(w http.ResponseWriter, logger *slog.Logger, err error) {
	var (
		status = http.StatusInternalServerError
		body   = api.Error{Code: "internal_error", Message: "internal error"}

		validation *service.ValidationError
		badParam   *api.InvalidParamFormatError
		missing    *api.RequiredParamError
		tooMany    *api.TooManyValuesForParamError
	)
	switch {
	case errors.Is(err, service.ErrProductNotFound):
		status, body = http.StatusNotFound, api.Error{Code: "product_not_found", Message: "product not found"}
	case errors.As(err, &validation):
		status, body = http.StatusBadRequest, api.Error{Code: "invalid_request", Message: validation.Message, Field: &validation.Field}
	case errors.As(err, &badParam):
		// Raised by the generated code before our handler even runs, e.g. GET /products/abc.
		status, body = http.StatusBadRequest, api.Error{Code: "invalid_request", Message: "invalid format", Field: &badParam.ParamName}
	case errors.As(err, &missing):
		status, body = http.StatusBadRequest, api.Error{Code: "invalid_request", Message: "is required", Field: &missing.ParamName}
	case errors.As(err, &tooMany):
		status, body = http.StatusBadRequest, api.Error{Code: "invalid_request", Message: "must be given once", Field: &tooMany.ParamName}
	// The generator has more binder error types (header, cookie, JSON-encoded params). This spec declares none
	// of those inputs, so they cannot occur yet; add a case here when the spec starts using them.
	default:
		// Unknown error: log the real cause, never send it to the client.
		logger.Error("request failed", "err", err)
	}
	writeJSON(w, logger, status, body)
}

// notFound and methodNotAllowed replace chi's plain-text defaults, so even unknown routes answer in api.Error.
func notFound(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, logger, http.StatusNotFound, api.Error{Code: "route_not_found", Message: "no such route"})
	}
}

// methodNotAllowed also sets the Allow header, which RFC 9110 requires on a 405 (chi's default did).
// It asks the router which methods match this path, so the list never drifts from the real routes.
func methodNotAllowed(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if rctx := chi.RouteContext(r.Context()); rctx != nil && rctx.Routes != nil {
			var allowed []string
			for _, m := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions} {
				if rctx.Routes.Match(chi.NewRouteContext(), m, r.URL.Path) {
					allowed = append(allowed, m)
				}
			}
			if len(allowed) > 0 {
				w.Header().Set("Allow", strings.Join(allowed, ", "))
			}
		}
		writeJSON(w, logger, http.StatusMethodNotAllowed, api.Error{Code: "method_not_allowed", Message: "method not allowed"})
	}
}
