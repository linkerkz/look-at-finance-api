package service

import (
	"context"
	"time"

	"github.com/lukivan8/look-at-finance-api/internal/modules/products/models"
	"github.com/lukivan8/look-at-finance-api/internal/modules/products/repository"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID string, req models.ListProductsRequest) (*models.ListProductsResponse, error) {
	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 50
	}

	items, total, err := s.repo.List(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	if items == nil {
		items = []models.ProductListItem{}
	}

	totalPages := (total + req.Limit - 1) / req.Limit

	return &models.ListProductsResponse{
		Data: items,
		Meta: models.PaginationMeta{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *Service) GetByID(ctx context.Context, userID string, productID string) (*models.ProductDetailResponse, error) {
	product, err := s.repo.GetByID(ctx, userID, productID)
	if err != nil {
		return nil, err
	}

	priceHistory, err := s.repo.GetPriceHistory(ctx, productID)
	if err != nil {
		return nil, err
	}

	totalPurchases, err := s.repo.GetTotalPurchases(ctx, userID, productID)
	if err != nil {
		return nil, err
	}

	// Group prices by store
	storeMap := make(map[string]*models.StorePriceHistory)
	for _, p := range priceHistory {
		if _, exists := storeMap[p.StoreID]; !exists {
			storeMap[p.StoreID] = &models.StorePriceHistory{
				StoreID:   p.StoreID,
				StoreName: p.StoreName,
				Prices:    []models.PriceRecord{},
				MinPrice:  p.Price,
				MaxPrice:  p.Price,
			}
		}
		sph := storeMap[p.StoreID]
		sph.Prices = append(sph.Prices, models.PriceRecord{
			Price: p.Price,
			Date:  p.RecordedAt,
		})
		if p.Price < sph.MinPrice {
			sph.MinPrice = p.Price
		}
		if p.Price > sph.MaxPrice {
			sph.MaxPrice = p.Price
		}
		// First price is the most recent (current)
		if len(sph.Prices) == 1 {
			sph.CurrentPrice = p.Price
		}
	}

	// Convert map to slice
	history := make([]models.StorePriceHistory, 0, len(storeMap))
	for _, sph := range storeMap {
		history = append(history, *sph)
	}

	// Get last seen date
	var lastSeen time.Time
	lastPrice, err := s.repo.GetLastSeen(ctx, productID)
	if err != nil {
		return nil, err
	}
	if lastPrice != nil {
		lastSeen = lastPrice.RecordedAt
	} else {
		lastSeen = product.CreatedAt
	}

	return &models.ProductDetailResponse{
		ID:             product.ID,
		Name:           product.Name,
		Category:       product.Category,
		PriceHistory:   history,
		LastSeen:       lastSeen,
		TotalPurchases: totalPurchases,
	}, nil
}

func (s *Service) GetOrCreate(ctx context.Context, name string, category string) (*models.Product, error) {
	return s.repo.GetOrCreateByName(ctx, name, category)
}

func (s *Service) RecordPrice(ctx context.Context, productID string, storeID string, price float64) error {
	return s.repo.RecordPrice(ctx, productID, storeID, price)
}
