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
