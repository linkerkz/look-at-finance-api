package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/lukivan8/look-at-finance-api/internal/modules/auth/dto"
	"github.com/lukivan8/look-at-finance-api/internal/modules/auth/service"
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

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := validateRegisterRequest(req); err != nil {
		response.ValidationError(w, err.Error())
		return
	}

	result, err := h.service.Register(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrUserExists) {
			response.Conflict(w, "user with this email already exists")
			return
		}
		h.log.Errorw("failed to register user", "error", err)
		response.InternalServerError(w, "failed to register user")
		return
	}

	response.Success(w, http.StatusCreated, result)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		response.ValidationError(w, "email and password are required")
		return
	}

	result, err := h.service.Login(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			response.Unauthorized(w, "invalid email or password")
			return
		}
		h.log.Errorw("failed to login user", "error", err)
		response.InternalServerError(w, "failed to login")
		return
	}

	response.Success(w, http.StatusOK, result)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		response.ValidationError(w, "refreshToken is required")
		return
	}

	result, err := h.service.RefreshTokens(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			response.Unauthorized(w, "invalid or expired refresh token")
			return
		}
		h.log.Errorw("failed to refresh token", "error", err)
		response.InternalServerError(w, "failed to refresh token")
		return
	}

	response.Success(w, http.StatusOK, result)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Unauthorized(w, "user not authenticated")
		return
	}

	result, err := h.service.GetCurrentUser(r.Context(), userID)
	if err != nil {
		h.log.Errorw("failed to get current user", "error", err, "user_id", userID)
		response.InternalServerError(w, "failed to get user profile")
		return
	}

	response.Success(w, http.StatusOK, result)
}

func validateRegisterRequest(req dto.RegisterRequest) error {
	if req.Email == "" {
		return errors.New("email is required")
	}
	if req.Password == "" {
		return errors.New("password is required")
	}
	if len(req.Password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	return nil
}
