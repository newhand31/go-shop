package service

import (
	"context"
	"errors"
	"testing"

	"github.com/newhand31/go-shop/internal/repository"
)

type fakeStore struct {
	rows []repository.Product
	err  error
}

func (f fakeStore) List(context.Context) ([]repository.Product, error) { return f.rows, f.err }

func TestProductService_Get(t *testing.T) {
	boom := errors.New("boom")
	store := fakeStore{rows: []repository.Product{{ID: 1, Name: "a", PriceTWD: 10}, {ID: 2, Name: "b", PriceTWD: 20}}}
	tests := []struct {
		name        string
		store       fakeStore
		id          int
		want        Product
		wantIs      error
		wantInvalid bool
	}{
		{"found", store, 2, Product{ID: 2, Name: "b", PriceTWD: 20}, nil, false},
		{"unknown id wraps sentinel", store, 9, Product{}, ErrProductNotFound, false},
		{"zero id is a validation error", store, 0, Product{}, nil, true},
		{"negative id is a validation error", store, -5, Product{}, nil, true},
		{"store error stays in the chain", fakeStore{err: boom}, 1, Product{}, boom, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewProductService(tt.store).Get(context.Background(), tt.id)
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Fatalf("err = %v, want errors.Is %v", err, tt.wantIs)
			}
			var ve *ValidationError
			if gotInvalid := errors.As(err, &ve); gotInvalid != tt.wantInvalid {
				t.Fatalf("errors.As(*ValidationError) = %v, want %v (err = %v)", gotInvalid, tt.wantInvalid, err)
			}
			if tt.wantInvalid && ve.Field != "id" {
				t.Errorf("field = %q, want id", ve.Field)
			}
			if tt.wantIs == nil && !tt.wantInvalid && err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestProductService_List(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name    string
		store   fakeStore
		want    []Product
		wantErr error
	}{
		{"empty list", fakeStore{}, []Product{}, nil},
		{"maps fields", fakeStore{rows: []repository.Product{{ID: 1, Name: "a", PriceTWD: 10}}}, []Product{{ID: 1, Name: "a", PriceTWD: 10}}, nil},
		{"wraps store error", fakeStore{err: boom}, nil, boom},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewProductService(tt.store).List(context.Background())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
