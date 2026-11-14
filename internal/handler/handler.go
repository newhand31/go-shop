// Package handler is the HTTP layer: parse the request, call the service, write the response. No SQL here.
package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/newhand31/go-shop/internal/api"
	"github.com/newhand31/go-shop/internal/service"
)

// Products is the service capability the handler needs, defined by the consumer.
type Products interface {
	List(ctx context.Context) ([]service.Product, error)
	Get(ctx context.Context, id int) (service.Product, error)
}

// server implements api.ServerInterface, the interface oapi-codegen generated from api/openapi.yaml.
// If the spec adds an operation and this type lacks the method, the build fails: the contract is enforced by the compiler.
type server struct {
	products Products
	carts    Carts
	orders   Orders
	logger   *slog.Logger
}

var _ api.ServerInterface = (*server)(nil)

func (s *server) ListProducts(w http.ResponseWriter, r *http.Request) {
	items, err := s.products.List(r.Context())
	if err != nil {
		writeError(w, s.logger, err)
		return
	}
	out := make([]api.Product, 0, len(items))
	for _, p := range items {
		out = append(out, toAPI(p))
	}
	writeJSON(w, s.logger, http.StatusOK, out)
}

func (s *server) GetProduct(w http.ResponseWriter, r *http.Request, id int) {
	p, err := s.products.Get(r.Context(), id)
	if err != nil {
		writeError(w, s.logger, err)
		return
	}
	writeJSON(w, s.logger, http.StatusOK, toAPI(p))
}

func toAPI(p service.Product) api.Product {
	return api.Product{Id: p.ID, Name: p.Name, PriceTwd: p.PriceTWD}
}

// New builds the router. The chain runs top to bottom: RequestLog, then Recover (so a panic still gets a logged 500),
// then any extra middleware the caller passes (rate limit, session), then the route itself.
func New(products Products, carts Carts, orders Orders, logger *slog.Logger, middleware ...func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(RequestLog(logger), Recover(logger))
	r.Use(middleware...)
	r.NotFound(notFound(logger))
	r.MethodNotAllowed(methodNotAllowed(logger))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return api.HandlerWithOptions(&server{products: products, carts: carts, orders: orders, logger: logger}, api.ChiServerOptions{
		BaseRouter: r,
		// Path/param binding errors happen inside generated code; send them through the same writeError.
		ErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) { writeError(w, logger, err) },
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// RequestLog logs each request's method, path, status code and duration with slog.
func RequestLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.InfoContext(r.Context(), "request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
