package response

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

type ErrorResponse struct {
	Success bool       `json:"success"`
	Error   ErrorBody  `json:"error"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	TraceID string `json:"trace_id"`
}

func Success(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(SuccessResponse{
		Success: true,
		Data:    data,
	})
}

func Error(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Success: false,
		Error: ErrorBody{
			Code:    code,
			Message: message,
			TraceID: uuid.New().String(),
		},
	})
}

// Common error codes
const (
	ErrCodeBadRequest          = "BAD_REQUEST"
	ErrCodeUnauthorized        = "UNAUTHORIZED"
	ErrCodeForbidden           = "FORBIDDEN"
	ErrCodeNotFound            = "RESOURCE_NOT_FOUND"
	ErrCodeConflict            = "CONFLICT"
	ErrCodeUnprocessableEntity = "UNPROCESSABLE_ENTITY"
	ErrCodeInternalServer      = "INTERNAL_SERVER_ERROR"
	ErrCodeValidation          = "VALIDATION_ERROR"
)

func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, ErrCodeBadRequest, message)
}

func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, ErrCodeUnauthorized, message)
}

func Forbidden(w http.ResponseWriter, message string) {
	Error(w, http.StatusForbidden, ErrCodeForbidden, message)
}

func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, ErrCodeNotFound, message)
}

func Conflict(w http.ResponseWriter, message string) {
	Error(w, http.StatusConflict, ErrCodeConflict, message)
}

func UnprocessableEntity(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnprocessableEntity, ErrCodeUnprocessableEntity, message)
}

func InternalServerError(w http.ResponseWriter, message string) {
	Error(w, http.StatusInternalServerError, ErrCodeInternalServer, message)
}

func ValidationError(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, ErrCodeValidation, message)
}

