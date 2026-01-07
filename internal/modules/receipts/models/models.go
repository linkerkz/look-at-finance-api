package models

import "time"

// Request DTOs
type ScanReceiptRequest struct {
	URL string `json:"url"`
}

type ListReceiptsRequest struct {
	Page      int
	Limit     int
	StartDate string
	EndDate   string
	StoreID   string
}

// Response DTOs
type ReceiptItemResponse struct {
	ID         string  `json:"id"`
	ProductID  string  `json:"productId,omitempty"`
	Name       string  `json:"name"`
	Quantity   float64 `json:"quantity"`
	Price      float64 `json:"price"`
	TotalPrice float64 `json:"totalPrice"`
	Unit       string  `json:"unit"`
}

type ReceiptResponse struct {
	ID           string                `json:"id"`
	FiscalID     string                `json:"fiscalId"`
	StoreName    string                `json:"storeName"`
	StoreAddress string                `json:"storeAddress,omitempty"`
	StoreID      string                `json:"storeId"`
	PurchaseDate time.Time             `json:"purchaseDate"`
	TotalAmount  float64               `json:"totalAmount"`
	Items        []ReceiptItemResponse `json:"items"`
	RawURL       string                `json:"rawUrl"`
	CreatedAt    time.Time             `json:"createdAt"`
}

type ReceiptListItem struct {
	ID           string    `json:"id"`
	StoreName    string    `json:"storeName"`
	PurchaseDate time.Time `json:"purchaseDate"`
	TotalAmount  float64   `json:"totalAmount"`
	ItemCount    int       `json:"itemCount"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type ListReceiptsResponse struct {
	Data []ReceiptListItem `json:"data"`
	Meta PaginationMeta    `json:"meta"`
}

// Internal models
type Receipt struct {
	ID           string
	UserID       string
	StoreID      string
	StoreName    string
	StoreAddress string
	FiscalID     string
	PurchaseDate time.Time
	TotalAmount  float64
	RawURL       string
	CreatedAt    time.Time
	Items        []ReceiptItem
}

type ReceiptItem struct {
	ID         string
	ReceiptID  string
	ProductID  string
	Name       string
	Quantity   float64
	Price      float64
	TotalPrice float64
	Unit       string
}

func (r *Receipt) ToResponse() ReceiptResponse {
	items := make([]ReceiptItemResponse, len(r.Items))
	for i, item := range r.Items {
		items[i] = ReceiptItemResponse{
			ID:         item.ID,
			ProductID:  item.ProductID,
			Name:       item.Name,
			Quantity:   item.Quantity,
			Price:      item.Price,
			TotalPrice: item.TotalPrice,
			Unit:       item.Unit,
		}
	}

	return ReceiptResponse{
		ID:           r.ID,
		FiscalID:     r.FiscalID,
		StoreName:    r.StoreName,
		StoreAddress: r.StoreAddress,
		StoreID:      r.StoreID,
		PurchaseDate: r.PurchaseDate,
		TotalAmount:  r.TotalAmount,
		Items:        items,
		RawURL:       r.RawURL,
		CreatedAt:    r.CreatedAt,
	}
}

func (r *Receipt) ToListItem(itemCount int) ReceiptListItem {
	return ReceiptListItem{
		ID:           r.ID,
		StoreName:    r.StoreName,
		PurchaseDate: r.PurchaseDate,
		TotalAmount:  r.TotalAmount,
		ItemCount:    itemCount,
	}
}

