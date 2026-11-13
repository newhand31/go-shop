package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// memCartStore is a fake CartStore: carts live in a map keyed by cart id.
type memCartStore struct {
	carts map[string]map[int]int
	err   error
}

func newMemCartStore() *memCartStore { return &memCartStore{carts: map[string]map[int]int{}} }

func (m *memCartStore) Items(_ context.Context, id string) (map[int]int, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := map[int]int{}
	for k, v := range m.carts[id] {
		out[k] = v
	}
	return out, nil
}

func (m *memCartStore) SetItem(_ context.Context, id string, productID, qty int) error {
	if m.err != nil {
		return m.err
	}
	if m.carts[id] == nil {
		m.carts[id] = map[int]int{}
	}
	m.carts[id][productID] = qty
	return nil
}

func (m *memCartStore) RemoveItem(_ context.Context, id string, productID int) error {
	if m.err != nil {
		return m.err
	}
	delete(m.carts[id], productID)
	return nil
}

type fakeCatalog struct{ items []Product }

func (f fakeCatalog) List(context.Context) ([]Product, error) { return f.items, nil }

var testCatalog = fakeCatalog{items: []Product{
	{ID: 1, Name: "keyboard", PriceTWD: 1500},
	{ID: 2, Name: "mouse", PriceTWD: 500},
	{ID: 3, Name: "cable", PriceTWD: 100},
}}

func TestCartService_SetItem_Validation(t *testing.T) {
	tests := []struct {
		name      string
		productID int
		quantity  int
		wantField string // "" means no ValidationError
		wantErr   error
	}{
		{"ok", 1, 2, "", nil},
		{"quantity zero", 1, 0, "quantity", nil},
		{"quantity negative", 1, -1, "quantity", nil},
		{"quantity over limit", 1, 100, "quantity", nil},
		{"quantity at limit", 1, 99, "", nil},
		{"product id zero", 0, 1, "productId", nil},
		{"unknown product", 42, 1, "", ErrProductNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewCartService(newMemCartStore(), testCatalog)
			_, err := s.SetItem(context.Background(), "c1", tt.productID, tt.quantity)

			var ve *ValidationError
			switch {
			case tt.wantField != "":
				if !errors.As(err, &ve) || ve.Field != tt.wantField {
					t.Fatalf("err = %v, want ValidationError on %q", err, tt.wantField)
				}
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
			default:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCartService_GetTotalsAndOrder(t *testing.T) {
	store := newMemCartStore()
	s := NewCartService(store, testCatalog)
	ctx := context.Background()
	for _, step := range []struct{ id, qty int }{{3, 4}, {1, 1}, {2, 2}} {
		if _, err := s.SetItem(ctx, "c1", step.id, step.qty); err != nil {
			t.Fatal(err)
		}
	}
	// A line whose product was deleted from the catalog is skipped, not an error.
	store.carts["c1"][99] = 5

	cart, err := s.Get(ctx, "c1")
	if err != nil {
		t.Fatal(err)
	}
	var ids []int
	for _, it := range cart.Items {
		ids = append(ids, it.ProductID)
	}
	if want := []int{1, 2, 3}; !reflect.DeepEqual(ids, want) {
		t.Fatalf("item order = %v, want %v (sorted by product id)", ids, want)
	}
	if want := 1500 + 2*500 + 4*100; cart.TotalTWD != want {
		t.Fatalf("total = %d, want %d", cart.TotalTWD, want)
	}
}

func TestCartService_SetItemReplacesQuantityAndRemove(t *testing.T) {
	s := NewCartService(newMemCartStore(), testCatalog)
	ctx := context.Background()

	if _, err := s.SetItem(ctx, "c1", 1, 2); err != nil {
		t.Fatal(err)
	}
	cart, _ := s.SetItem(ctx, "c1", 1, 5) // set, not add
	if len(cart.Items) != 1 || cart.Items[0].Quantity != 5 {
		t.Fatalf("cart = %+v, want one line with quantity 5", cart)
	}
	cart, err := s.RemoveItem(ctx, "c1", 1)
	if err != nil || len(cart.Items) != 0 || cart.TotalTWD != 0 {
		t.Fatalf("after remove: %+v, %v", cart, err)
	}
	if _, err := s.RemoveItem(ctx, "c1", 1); err != nil {
		t.Fatalf("removing twice should be fine: %v", err)
	}
}

func TestCartService_CartsAreIsolated(t *testing.T) {
	s := NewCartService(newMemCartStore(), testCatalog)
	ctx := context.Background()
	if _, err := s.SetItem(ctx, "alice", 1, 1); err != nil {
		t.Fatal(err)
	}
	cart, err := s.Get(ctx, "bob")
	if err != nil || len(cart.Items) != 0 {
		t.Fatalf("bob sees %+v, %v; want an empty cart", cart, err)
	}
}

func TestCartService_StoreErrorIsWrapped(t *testing.T) {
	boom := errors.New("redis down")
	store := newMemCartStore()
	store.err = boom
	s := NewCartService(store, testCatalog)
	if _, err := s.Get(context.Background(), "c1"); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want it to wrap %v", err, boom)
	}
}
