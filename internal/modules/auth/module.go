package auth

import (
	"github.com/go-chi/chi/v5"
	"github.com/lukivan8/look-at-finance-api/internal/modules/auth/delivery"
	"github.com/lukivan8/look-at-finance-api/internal/modules/auth/repository"
	"github.com/lukivan8/look-at-finance-api/internal/modules/auth/service"
	"github.com/lukivan8/look-at-finance-api/internal/shared/database"
	"github.com/lukivan8/look-at-finance-api/internal/shared/logger"
)

type Module struct {
	handler *delivery.Handler
}

func NewModule(db *database.DB, log *logger.Logger, jwtSecret string) *Module {
	repo := repository.New(db)
	svc := service.New(repo, jwtSecret)
	handler := delivery.NewHandler(svc, log)

	return &Module{handler: handler}
}

func (m *Module) RegisterPublicRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", m.handler.Register)
		r.Post("/login", m.handler.Login)
		r.Post("/refresh", m.handler.Refresh)
	})
}

func (m *Module) RegisterProtectedRoutes(r chi.Router) {
	r.Get("/auth/me", m.handler.Me)
}
