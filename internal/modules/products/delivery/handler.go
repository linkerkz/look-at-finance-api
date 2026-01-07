package delivery

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/lukivan8/look-at-finance-api/internal/modules/products/models"
	"github.com/lukivan8/look-at-finance-api/internal/modules/products/repository"
	"github.com/lukivan8/look-at-finance-api/internal/modules/products/service"
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

	req := models.ListProductsRequest{
		Page:      parseIntQuery(r, "page", 1),
		Limit:     parseIntQuery(r, "limit", 50),
		Search:    r.URL.Query().Get("search"),
		Category:  r.URL.Query().Get("category"),
		StoreID:   r.URL.Query().Get("storeId"),
		SortBy:    r.URL.Query().Get("sortBy"),
		SortOrder: r.URL.Query().Get("sortOrder"),
	}

	result, err := h.service.List(r.Context(), userID, req)
	if err != nil {
		h.log.Errorw("failed to list products", "error", err, "user_id", userID)
		response.InternalServerError(w, "failed to list products")
		return
	}

	response.Success(w, http.StatusOK, result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, "user not authenticated")
		return
	}

	productID := chi.URLParam(r, "id")
	if productID == "" {
		response.BadRequest(w, "product ID is required")
		return
	}

	result, err := h.service.GetByID(r.Context(), userID, productID)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			response.NotFound(w, "product not found")
			return
		}
		h.log.Errorw("failed to get product", "error", err, "user_id", userID, "product_id", productID)
		response.InternalServerError(w, "failed to get product")
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
