package service

import (
	"context"
	"errors"

	productModels "github.com/lukivan8/look-at-finance-api/internal/modules/products/models"
	productService "github.com/lukivan8/look-at-finance-api/internal/modules/products/service"
	"github.com/lukivan8/look-at-finance-api/internal/modules/receipts/models"
	"github.com/lukivan8/look-at-finance-api/internal/modules/receipts/ofd"
	"github.com/lukivan8/look-at-finance-api/internal/modules/receipts/repository"
	storeModels "github.com/lukivan8/look-at-finance-api/internal/modules/stores/models"
	storeService "github.com/lukivan8/look-at-finance-api/internal/modules/stores/service"
)

var (
	ErrInvalidURL       = errors.New("invalid OFD URL format")
	ErrReceiptNotFound  = errors.New("receipt not found on OFD portal")
	ErrParsingFailed    = errors.New("failed to parse receipt data")
	ErrDuplicateReceipt = errors.New("receipt already exists")
)

// StoreServiceInterface defines what we need from the store service
type StoreServiceInterface interface {
	GetOrCreate(ctx context.Context, name string, address string) (*storeModels.Store, error)
}

// ProductServiceInterface defines what we need from the product service
type ProductServiceInterface interface {
	GetOrCreate(ctx context.Context, name string, category string) (*productModels.Product, error)
	RecordPrice(ctx context.Context, productID string, storeID string, price float64) error
}

type Service struct {
	repo           *repository.Repository
	parser         *ofd.Parser
	storeService   StoreServiceInterface
	productService ProductServiceInterface
}

func New(
	repo *repository.Repository,
	storeSvc *storeService.Service,
	productSvc *productService.Service,
) *Service {
	return &Service{
		repo:           repo,
		parser:         ofd.NewParser(),
		storeService:   storeSvc,
		productService: productSvc,
	}
}

func (s *Service) Scan(ctx context.Context, userID string, req models.ScanReceiptRequest) (*models.ReceiptResponse, error) {
	// Parse the receipt from OFD portal
	parsed, err := s.parser.Parse(req.URL)
	if err != nil {
		if errors.Is(err, ofd.ErrInvalidURL) {
			return nil, ErrInvalidURL
		}
		if errors.Is(err, ofd.ErrReceiptNotFound) {
			return nil, ErrReceiptNotFound
		}
		if errors.Is(err, ofd.ErrParsingFailed) {
			return nil, ErrParsingFailed
		}
		return nil, err
	}

	// Check if receipt already exists for this user
	exists, err := s.repo.ExistsByFiscalID(ctx, userID, parsed.FiscalID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicateReceipt
	}

	// Get or create store
	store, err := s.storeService.GetOrCreate(ctx, parsed.StoreName, parsed.StoreAddress)
	if err != nil {
		return nil, err
	}

	// Create receipt
	receipt := &models.Receipt{
		UserID:       userID,
		StoreID:      store.ID,
		StoreName:    store.Name,
		StoreAddress: store.Address,
		FiscalID:     parsed.FiscalID,
		PurchaseDate: parsed.PurchaseDate,
		TotalAmount:  parsed.TotalAmount,
		RawURL:       parsed.RawURL,
		Items:        make([]models.ReceiptItem, 0, len(parsed.Items)),
	}

	// Process items - create products and record prices
	for _, parsedItem := range parsed.Items {
		// Get or create product
		product, err := s.productService.GetOrCreate(ctx, parsedItem.Name, "")
		if err != nil {
			return nil, err
		}

		// Record price for this store
		if err := s.productService.RecordPrice(ctx, product.ID, store.ID, parsedItem.Price); err != nil {
			return nil, err
		}

		item := models.ReceiptItem{
			ProductID:  product.ID,
			Name:       parsedItem.Name,
			Quantity:   parsedItem.Quantity,
			Price:      parsedItem.Price,
			TotalPrice: parsedItem.TotalPrice,
			Unit:       parsedItem.Unit,
		}
		receipt.Items = append(receipt.Items, item)
	}

	// Save receipt
	if err := s.repo.Create(ctx, receipt); err != nil {
		if errors.Is(err, repository.ErrDuplicateReceipt) {
			return nil, ErrDuplicateReceipt
		}
		return nil, err
	}

	resp := receipt.ToResponse()
	return &resp, nil
}

func (s *Service) GetByID(ctx context.Context, userID string, receiptID string) (*models.ReceiptResponse, error) {
	receipt, err := s.repo.GetByID(ctx, userID, receiptID)
	if err != nil {
		return nil, err
	}

	resp := receipt.ToResponse()
	return &resp, nil
}

func (s *Service) List(ctx context.Context, userID string, req models.ListReceiptsRequest) (*models.ListReceiptsResponse, error) {
	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.Limit < 1 || req.Limit > 100 {
		req.Limit = 20
	}

	receiptList, total, err := s.repo.List(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	data := make([]models.ReceiptListItem, len(receiptList))
	for i, r := range receiptList {
		data[i] = r.ToListItem(len(r.Items))
	}

	totalPages := (total + req.Limit - 1) / req.Limit

	return &models.ListReceiptsResponse{
		Data: data,
		Meta: models.PaginationMeta{
			Page:       req.Page,
			Limit:      req.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *Service) Delete(ctx context.Context, userID string, receiptID string) error {
	return s.repo.Delete(ctx, userID, receiptID)
}
