package service

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/newhand31/go-shop/internal/repository"
)

// Checkout opens an "order.place" child span under whatever span the caller's context carries.
func TestCheckout_RecordsAChildSpan(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	tests := []struct {
		name      string
		storeErr  error
		wantError bool
	}{
		{"success", nil, false},
		{"store fails", errors.New("boom"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := len(rec.Ended())
			ctx, parent := tp.Tracer("test").Start(context.Background(), "request")
			cart := &fakeCheckoutCart{items: map[int]int{1: 1}}
			_, _ = NewOrderService(&fakeOrderStore{err: tt.storeErr, order: repository.Order{ID: 1}}, cart, quiet()).Checkout(ctx, "c1")
			parent.End()

			ended := rec.Ended()[before:]
			if len(ended) != 2 || ended[0].Name() != "order.place" {
				t.Fatalf("ended spans = %d, first %q; want order.place then request", len(ended), ended[0].Name())
			}
			if ended[0].Parent().SpanID() != parent.SpanContext().SpanID() {
				t.Error("order.place is not a child of the request span")
			}
			if isErr := ended[0].Status().Code.String() == "Error"; isErr != tt.wantError {
				t.Errorf("error status = %v, want %v", isErr, tt.wantError)
			}
		})
	}
}
