package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/newhand31/go-shop/internal/api"
	"github.com/newhand31/go-shop/internal/service"
)

// FlashSales is the service capability the flash-sale route needs, defined by the consumer.
type FlashSales interface {
	Buy(ctx context.Context, productID, qty int) (service.Order, error)
}

// FlashSalePath is the flash-sale route: one unit of one product, no cart. A stampede does not fill carts first.
const FlashSalePath = "/flash-sales/{productID}/orders"

// FlashSaleBuy serves POST /flash-sales/{productID}/orders. It buys one unit. A full shop is a 503 with Retry-After
// (see writeError), a sold-out product a 409.
//
// Like the payment callback it is a plain chi route outside the OpenAPI file for now: it is a temporary event, and the
// contract stays about the everyday shop.
func FlashSaleBuy(sales FlashSales, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productID, err := strconv.Atoi(chi.URLParam(r, "productID"))
		if err != nil || productID <= 0 {
			field := "productID"
			writeJSON(w, logger, http.StatusBadRequest, api.Error{Code: "invalid_request", Message: "invalid format", Field: &field})
			return
		}
		order, err := sales.Buy(r.Context(), productID, 1)
		if err != nil {
			writeError(w, logger, err)
			return
		}
		out := api.Order{Id: order.ID, TotalTwd: order.TotalTWD, Lines: make([]api.OrderLine, 0, len(order.Lines))}
		for _, l := range order.Lines {
			out.Lines = append(out.Lines, api.OrderLine{ProductId: l.ProductID, Quantity: l.Quantity, UnitPriceTwd: l.UnitPriceTWD})
		}
		writeJSON(w, logger, http.StatusCreated, out)
	}
}

// WithFlashSale puts the flash-sale route in front of the API handler, behind the given middleware (the rate limiter).
// There is no session middleware on purpose: this route has no cart, so it has no use for a session.
func WithFlashSale(api http.Handler, sales FlashSales, logger *slog.Logger, middleware ...func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()
	r.With(RequestLog(logger), Recover(logger)).With(middleware...).Post(FlashSalePath, FlashSaleBuy(sales, logger))
	r.Mount("/", api)
	return r
}
