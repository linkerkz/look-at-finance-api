package models

import "time"

// Request DTOs
type SummaryRequest struct {
	Period    string // day, week, month, year
	StartDate string
	EndDate   string
}

type ListExpensesRequest struct {
	Page      int
	Limit     int
	StartDate string
	EndDate   string
	StoreID   string
	Category  string
	MinAmount float64
	MaxAmount float64
}

// Response DTOs
type PeriodInfo struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type CategoryStat struct {
	Category   string  `json:"category"`
	Amount     float64 `json:"amount"`
	Percentage float64 `json:"percentage"`
}

type StoreStat struct {
	StoreID    string  `json:"storeId"`
	StoreName  string  `json:"storeName"`
	Amount     float64 `json:"amount"`
	Percentage float64 `json:"percentage"`
}

type TrendPoint struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
}

type Comparison struct {
	PreviousPeriod float64 `json:"previousPeriod"`
	Change         float64 `json:"change"`
	ChangeType     string  `json:"changeType"` // increase, decrease, same
}

type SummaryResponse struct {
	Period            PeriodInfo     `json:"period"`
	TotalSpent        float64        `json:"totalSpent"`
	ReceiptCount      int            `json:"receiptCount"`
	AveragePerReceipt float64        `json:"averagePerReceipt"`
	ByCategory        []CategoryStat `json:"byCategory"`
	ByStore           []StoreStat    `json:"byStore"`
	Trend             []TrendPoint   `json:"trend"`
	Comparison        Comparison     `json:"comparison"`
}

type ExpenseListItem struct {
	ID        string    `json:"id"`
	Date      time.Time `json:"date"`
	StoreName string    `json:"storeName"`
	StoreID   string    `json:"storeId"`
	Amount    float64   `json:"amount"`
	Category  string    `json:"category"`
	ItemCount int       `json:"itemCount"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type ListExpensesResponse struct {
	Data []ExpenseListItem `json:"data"`
	Meta PaginationMeta    `json:"meta"`
}

