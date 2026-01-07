package expenses

import (
	"github.com/go-chi/chi/v5"
	"github.com/lukivan8/look-at-finance-api/internal/modules/expenses/delivery"
	"github.com/lukivan8/look-at-finance-api/internal/modules/expenses/repository"
	"github.com/lukivan8/look-at-finance-api/internal/modules/expenses/service"
	"github.com/lukivan8/look-at-finance-api/internal/shared/database"
	"github.com/lukivan8/look-at-finance-api/internal/shared/logger"
)

type Module struct {
	handler *delivery.Handler
}

func NewModule(db *database.DB, log *logger.Logger) *Module {
	repo := repository.New(db)
	svc := service.New(repo)
	handler := delivery.NewHandler(svc, log)

	return &Module{handler: handler}
}

func (m *Module) RegisterRoutes(r chi.Router) {
	r.Route("/expenses", func(r chi.Router) {
		r.Get("/summary", m.handler.Summary)
		r.Get("/", m.handler.List)
	})
}
