package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

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
	)
	switch {
	case errors.Is(err, service.ErrProductNotFound):
		status, body = http.StatusNotFound, api.Error{Code: "product_not_found", Message: "product not found"}
	case errors.As(err, &validation):
		status, body = http.StatusBadRequest, api.Error{Code: "invalid_request", Message: validation.Message, Field: &validation.Field}
	case errors.As(err, &badParam):
		// Raised by the generated code before our handler even runs, e.g. GET /products/abc.
		status, body = http.StatusBadRequest, api.Error{Code: "invalid_request", Message: "must be an integer", Field: &badParam.ParamName}
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

func methodNotAllowed(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, logger, http.StatusMethodNotAllowed, api.Error{Code: "method_not_allowed", Message: "method not allowed"})
	}
}
