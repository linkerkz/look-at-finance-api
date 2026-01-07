package service

import (
	"context"

	"github.com/lukivan8/look-at-finance-api/internal/modules/stores/models"
	"github.com/lukivan8/look-at-finance-api/internal/modules/stores/repository"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID string, search string) (*models.ListStoresResponse, error) {
	storeList, err := s.repo.ListByUser(ctx, userID, search)
	if err != nil {
		return nil, err
	}

	data := make([]models.StoreResponse, len(storeList))
	for i, store := range storeList {
		data[i] = store.ToResponse()
	}

	return &models.ListStoresResponse{Data: data}, nil
}

func (s *Service) GetOrCreate(ctx context.Context, name string, address string) (*models.Store, error) {
	return s.repo.GetOrCreateByName(ctx, name, address)
}
