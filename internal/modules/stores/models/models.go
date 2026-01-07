package models

import "time"

// Request DTOs
type ListStoresRequest struct {
	Search string
}

// Response DTOs
type StoreResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Address      string    `json:"address,omitempty"`
	ReceiptCount int       `json:"receiptCount"`
	TotalSpent   float64   `json:"totalSpent"`
	LastVisit    time.Time `json:"lastVisit"`
}

type ListStoresResponse struct {
	Data []StoreResponse `json:"data"`
}

// Internal models
type Store struct {
	ID           string
	Name         string
	Address      string
	ReceiptCount int
	TotalSpent   float64
	LastVisit    time.Time
	CreatedAt    time.Time
}

func (s *Store) ToResponse() StoreResponse {
	return StoreResponse{
		ID:           s.ID,
		Name:         s.Name,
		Address:      s.Address,
		ReceiptCount: s.ReceiptCount,
		TotalSpent:   s.TotalSpent,
		LastVisit:    s.LastVisit,
	}
}

