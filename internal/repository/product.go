// Package repository 是資料存取層。第 1 集先用寫死在記憶體的資料，第 2 集換成 PostgreSQL。
package repository

import "context"

// Product 是 repository 回傳的具體型別。
type Product struct {
	ID       int
	Name     string
	PriceTWD int
}

// MemoryProducts 把商品寫死在記憶體裡。
type MemoryProducts struct {
	items []Product
}

// NewMemoryProducts 建立內建三件商品的 repository。
func NewMemoryProducts() *MemoryProducts {
	return &MemoryProducts{items: []Product{
		{ID: 1, Name: "機械式鍵盤", PriceTWD: 2490},
		{ID: 2, Name: "人體工學滑鼠", PriceTWD: 1290},
		{ID: 3, Name: "27 吋螢幕", PriceTWD: 5990},
	}}
}

// List 回傳全部商品的複本，呼叫端改動不會影響內部資料。
func (r *MemoryProducts) List(_ context.Context) ([]Product, error) {
	out := make([]Product, len(r.items))
	copy(out, r.items)
	return out, nil
}
