package delivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/lukivan8/look-at-finance-api/internal/modules/receipts/models"
	"github.com/lukivan8/look-at-finance-api/internal/modules/receipts/repository"
	"github.com/lukivan8/look-at-finance-api/internal/modules/receipts/service"
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

func (h *Handler) Scan(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, "user not authenticated")
		return
	}

	var req models.ScanReceiptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.URL == "" {
		response.ValidationError(w, "url is required")
		return
	}

	result, err := h.service.Scan(r.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidURL):
			response.BadRequest(w, "invalid OFD URL format")
		case errors.Is(err, service.ErrReceiptNotFound):
			response.NotFound(w, "receipt not found on OFD portal")
		case errors.Is(err, service.ErrParsingFailed):
			response.UnprocessableEntity(w, "failed to parse receipt data")
		case errors.Is(err, service.ErrDuplicateReceipt):
			response.Conflict(w, "receipt already exists")
		default:
			h.log.Errorw("failed to scan receipt", "error", err, "user_id", userID, "url", req.URL)
			response.InternalServerError(w, "failed to scan receipt")
		}
		return
	}

	response.Success(w, http.StatusCreated, result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, "user not authenticated")
		return
	}

	receiptID := chi.URLParam(r, "id")
	if receiptID == "" {
		response.BadRequest(w, "receipt ID is required")
		return
	}

	result, err := h.service.GetByID(r.Context(), userID, receiptID)
	if err != nil {
		if errors.Is(err, repository.ErrReceiptNotFound) {
			response.NotFound(w, "receipt not found")
			return
		}
		h.log.Errorw("failed to get receipt", "error", err, "user_id", userID, "receipt_id", receiptID)
		response.InternalServerError(w, "failed to get receipt")
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

	req := models.ListReceiptsRequest{
		Page:      parseIntQuery(r, "page", 1),
		Limit:     parseIntQuery(r, "limit", 20),
		StartDate: r.URL.Query().Get("startDate"),
		EndDate:   r.URL.Query().Get("endDate"),
		StoreID:   r.URL.Query().Get("storeId"),
	}

	result, err := h.service.List(r.Context(), userID, req)
	if err != nil {
		h.log.Errorw("failed to list receipts", "error", err, "user_id", userID)
		response.InternalServerError(w, "failed to list receipts")
		return
	}

	response.Success(w, http.StatusOK, result)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, "user not authenticated")
		return
	}

	receiptID := chi.URLParam(r, "id")
	if receiptID == "" {
		response.BadRequest(w, "receipt ID is required")
		return
	}

	err := h.service.Delete(r.Context(), userID, receiptID)
	if err != nil {
		if errors.Is(err, repository.ErrReceiptNotFound) {
			response.NotFound(w, "receipt not found")
			return
		}
		h.log.Errorw("failed to delete receipt", "error", err, "user_id", userID, "receipt_id", receiptID)
		response.InternalServerError(w, "failed to delete receipt")
		return
	}

	response.Success(w, http.StatusNoContent, nil)
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
