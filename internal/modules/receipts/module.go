package receipts

import (
	"github.com/go-chi/chi/v5"
	productService "github.com/lukivan8/look-at-finance-api/internal/modules/products/service"
	"github.com/lukivan8/look-at-finance-api/internal/modules/receipts/delivery"
	"github.com/lukivan8/look-at-finance-api/internal/modules/receipts/repository"
	receiptService "github.com/lukivan8/look-at-finance-api/internal/modules/receipts/service"
	storeService "github.com/lukivan8/look-at-finance-api/internal/modules/stores/service"
	"github.com/lukivan8/look-at-finance-api/internal/shared/database"
	"github.com/lukivan8/look-at-finance-api/internal/shared/logger"
)

type Module struct {
	handler *delivery.Handler
}

func NewModule(
	db *database.DB,
	log *logger.Logger,
	storesSvc *storeService.Service,
	productsSvc *productService.Service,
) *Module {
	repo := repository.New(db)
	svc := receiptService.New(repo, storesSvc, productsSvc)
	handler := delivery.NewHandler(svc, log)

	return &Module{handler: handler}
}

func (m *Module) RegisterRoutes(r chi.Router) {
	r.Route("/receipts", func(r chi.Router) {
		r.Post("/scan", m.handler.Scan)
		r.Get("/", m.handler.List)
		r.Get("/{id}", m.handler.Get)
		r.Delete("/{id}", m.handler.Delete)
	})
}
