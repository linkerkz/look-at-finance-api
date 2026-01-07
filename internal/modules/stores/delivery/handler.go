package delivery

import (
	"net/http"

	"github.com/lukivan8/look-at-finance-api/internal/modules/stores/service"
	"github.com/lukivan8/look-at-finance-api/internal/shared/logger"
	"github.com/lukivan8/look-at-finance-api/internal/shared/middleware"
	"github.com/lukivan8/look-at-finance-api/internal/shared/response"
)

type Handler struct {
	service *service.Service
	log     *logger.Logger
}

func NewHandler(service *service.Service, log *logger.Logger) *Handler {
	return &Handler{
		service: service,
		log:     log,
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, "user not authenticated")
		return
	}

	search := r.URL.Query().Get("search")

	result, err := h.service.List(r.Context(), userID, search)
	if err != nil {
		h.log.Errorw("failed to list stores", "error", err, "user_id", userID)
		response.InternalServerError(w, "failed to list stores")
		return
	}

	response.Success(w, http.StatusOK, result)
}
