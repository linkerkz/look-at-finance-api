package service

import (
	"context"
	"time"

	"github.com/lukivan8/look-at-finance-api/internal/modules/expenses/models"
	"github.com/lukivan8/look-at-finance-api/internal/modules/expenses/repository"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetSummary(ctx context.Context, userID string, req models.SummaryRequest) (*models.SummaryResponse, error) {
	startDate, endDate := s.calculatePeriod(req)
	return s.repo.GetSummary(ctx, userID, startDate, endDate)
}

func (s *Service) calculatePeriod(req models.SummaryRequest) (time.Time, time.Time) {
	now := time.Now()

	// If explicit dates are provided, use them
	if req.StartDate != "" && req.EndDate != "" {
		startDate, err1 := time.Parse("2006-01-02", req.StartDate)
		endDate, err2 := time.Parse("2006-01-02", req.EndDate)
		if err1 == nil && err2 == nil {
			// Add 1 day to end date to include it
			return startDate, endDate.Add(24 * time.Hour)
		}
	}

	// Calculate based on period type
	switch req.Period {
	case "day":
		startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return startDate, startDate.Add(24 * time.Hour)
	case "week":
		// Start from Monday of current week
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startDate := time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		return startDate, startDate.Add(7 * 24 * time.Hour)
	case "year":
		startDate := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return startDate, time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, now.Location())
	default: // month
		startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return startDate, startDate.AddDate(0, 1, 0)
	}
}

func (s *Service) List(ctx context.Context, userID string, req models.ListExpensesRequest) (*models.ListExpensesResponse, error) {
	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	items, total, err := s.repo.ListExpenses(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	if items == nil {
		items = []models.ExpenseListItem{}
	}

	totalPages := (total + req.Limit - 1) / req.Limit

	return &models.ListExpensesResponse{
		Data: items,
		Meta: models.PaginationMeta{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}
