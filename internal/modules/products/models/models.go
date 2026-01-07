package models

import "time"

// Request DTOs
type ListProductsRequest struct {
	Page      int
	Limit     int
	Search    string
	Category  string
	StoreID   string
	SortBy    string
	SortOrder string
}

// Response DTOs
type PriceRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type ProductListItem struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Category   string     `json:"category,omitempty"`
	LastPrice  float64    `json:"lastPrice"`
	LastStore  string     `json:"lastStore"`
	LastSeen   time.Time  `json:"lastSeen"`
	PriceRange PriceRange `json:"priceRange"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type ListProductsResponse struct {
	Data []ProductListItem `json:"data"`
	Meta PaginationMeta    `json:"meta"`
}

type PriceRecord struct {
	Price float64   `json:"price"`
	Date  time.Time `json:"date"`
}

type StorePriceHistory struct {
	StoreID      string        `json:"storeId"`
	StoreName    string        `json:"storeName"`
	Prices       []PriceRecord `json:"prices"`
	CurrentPrice float64       `json:"currentPrice"`
	MinPrice     float64       `json:"minPrice"`
	MaxPrice     float64       `json:"maxPrice"`
}

type ProductDetailResponse struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Category       string              `json:"category,omitempty"`
	PriceHistory   []StorePriceHistory `json:"priceHistory"`
	LastSeen       time.Time           `json:"lastSeen"`
	TotalPurchases int                 `json:"totalPurchases"`
}

// Internal models
type Product struct {
	ID        string
	Name      string
	Category  string
	CreatedAt time.Time
}

type ProductPrice struct {
	ID         string
	ProductID  string
	StoreID    string
	StoreName  string
	Price      float64
	RecordedAt time.Time
}

