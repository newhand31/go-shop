package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"

	"github.com/newhand31/go-shop/internal/repository"
)

// fakeOrderStore records what it was asked to place and answers with a canned result.
type fakeOrderStore struct {
	got   []repository.OrderLine
	order repository.Order
	err   error
}

func (f *fakeOrderStore) Place(_ context.Context, lines []repository.OrderLine) (repository.Order, error) {
	f.got = lines
	return f.order, f.err
}

// fakeCheckoutCart is the two-method cart checkout needs; removeErr lets a test break the cleanup.
type fakeCheckoutCart struct {
	items     map[int]int
	removed   []int
	err       error
	removeErr error
}

func (f *fakeCheckoutCart) Items(context.Context, string) (map[int]int, error) { return f.items, f.err }
func (f *fakeCheckoutCart) RemoveItem(_ context.Context, _ string, id int) error {
	f.removed = append(f.removed, id)
	return f.removeErr
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestCheckout_PlacesSortedLinesAndEmptiesCart(t *testing.T) {
	cart := &fakeCheckoutCart{items: map[int]int{3: 1, 1: 2}}
	store := &fakeOrderStore{order: repository.Order{ID: 9, TotalTWD: 8470, Lines: []repository.OrderedLine{
		{ProductID: 1, Quantity: 2, UnitPriceTWD: 2490}, {ProductID: 3, Quantity: 1, UnitPriceTWD: 5990},
	}}}
	got, err := NewOrderService(store, cart, quiet()).Checkout(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if want := []repository.OrderLine{{ProductID: 1, Quantity: 2}, {ProductID: 3, Quantity: 1}}; !reflect.DeepEqual(store.got, want) {
		t.Fatalf("store got %v, want %v (sorted by product id)", store.got, want)
	}
	if got.ID != 9 || got.TotalTWD != 8470 || len(got.Lines) != 2 {
		t.Fatalf("order = %+v", got)
	}
	if !reflect.DeepEqual(cart.removed, []int{1, 3}) {
		t.Fatalf("cart cleanup removed %v, want [1 3]", cart.removed)
	}
}

func TestCheckout_EmptyCartIsValidationError(t *testing.T) {
	store := &fakeOrderStore{}
	_, err := NewOrderService(store, &fakeCheckoutCart{items: map[int]int{}}, quiet()).Checkout(context.Background(), "c1")
	var ve *ValidationError
	if !errors.As(err, &ve) || ve.Field != "cart" {
		t.Fatalf("err = %v, want ValidationError on cart", err)
	}
	if store.got != nil {
		t.Fatal("the store must not be called for an empty cart")
	}
}

func TestCheckout_ErrorsAreTranslatedAndKeepTheCart(t *testing.T) {
	tests := []struct {
		name    string
		storeEr error
		want    error
	}{
		{"out of stock", repository.ErrOutOfStock, ErrOutOfStock},
		{"product vanished", repository.ErrProductNotFound, ErrProductNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cart := &fakeCheckoutCart{items: map[int]int{3: 1}}
			store := &fakeOrderStore{err: tt.storeEr}
			_, err := NewOrderService(store, cart, quiet()).Checkout(context.Background(), "c1")
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if len(cart.removed) != 0 {
				t.Fatalf("cart was touched on failure: removed %v", cart.removed)
			}
		})
	}
}

func TestCheckout_UnknownStoreErrorIsWrapped(t *testing.T) {
	boom := errors.New("connection reset")
	store := &fakeOrderStore{err: boom}
	_, err := NewOrderService(store, &fakeCheckoutCart{items: map[int]int{1: 1}}, quiet()).Checkout(context.Background(), "c1")
	if !errors.Is(err, boom) || errors.Is(err, ErrOutOfStock) {
		t.Fatalf("err = %v, want it to wrap the cause and not be out-of-stock", err)
	}
}

// The order is already committed, so a cart that cannot be emptied must not turn it into an error.
func TestCheckout_CartCleanupFailureDoesNotFailTheOrder(t *testing.T) {
	cart := &fakeCheckoutCart{items: map[int]int{1: 1}, removeErr: errors.New("redis down")}
	store := &fakeOrderStore{order: repository.Order{ID: 1}}
	if _, err := NewOrderService(store, cart, quiet()).Checkout(context.Background(), "c1"); err != nil {
		t.Fatalf("err = %v, want nil: the order was placed", err)
	}
}
