package stores

import (
	"github.com/go-chi/chi/v5"
	"github.com/lukivan8/look-at-finance-api/internal/modules/stores/delivery"
	"github.com/lukivan8/look-at-finance-api/internal/modules/stores/repository"
	"github.com/lukivan8/look-at-finance-api/internal/modules/stores/service"
	"github.com/lukivan8/look-at-finance-api/internal/shared/database"
	"github.com/lukivan8/look-at-finance-api/internal/shared/logger"
)

type Module struct {
	handler *delivery.Handler
	Service *service.Service
}

func NewModule(db *database.DB, log *logger.Logger) *Module {
	repo := repository.New(db)
	svc := service.New(repo)
	handler := delivery.NewHandler(svc, log)

	return &Module{
		handler: handler,
		Service: svc,
	}
}

func (m *Module) RegisterRoutes(r chi.Router) {
	r.Get("/stores", m.handler.List)
}
