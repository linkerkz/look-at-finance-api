package delivery

import (
	"net/http"
	"strconv"

	"github.com/lukivan8/look-at-finance-api/internal/modules/expenses/models"
	"github.com/lukivan8/look-at-finance-api/internal/modules/expenses/service"
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

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, "user not authenticated")
		return
	}

	req := models.SummaryRequest{
		Period:    r.URL.Query().Get("period"),
		StartDate: r.URL.Query().Get("startDate"),
		EndDate:   r.URL.Query().Get("endDate"),
	}

	result, err := h.service.GetSummary(r.Context(), userID, req)
	if err != nil {
		h.log.Errorw("failed to get expense summary", "error", err, "user_id", userID)
		response.InternalServerError(w, "failed to get expense summary")
		return
	}

	response.Success(w, http.StatusOK, result)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, "user not authenticated")
		return
	}

	req := models.ListExpensesRequest{
		Page:      parseIntQuery(r, "page", 1),
		Limit:     parseIntQuery(r, "limit", 20),
		StartDate: r.URL.Query().Get("startDate"),
		EndDate:   r.URL.Query().Get("endDate"),
		StoreID:   r.URL.Query().Get("storeId"),
		Category:  r.URL.Query().Get("category"),
		MinAmount: parseFloatQuery(r, "minAmount", 0),
		MaxAmount: parseFloatQuery(r, "maxAmount", 0),
	}

	result, err := h.service.List(r.Context(), userID, req)
	if err != nil {
		h.log.Errorw("failed to list expenses", "error", err, "user_id", userID)
		response.InternalServerError(w, "failed to list expenses")
		return
	}

	response.Success(w, http.StatusOK, result)
}

func parseIntQuery(r *http.Request, key string, defaultValue int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func parseFloatQuery(r *http.Request, key string, defaultValue float64) float64 {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return defaultValue
	}
	return parsed
}
