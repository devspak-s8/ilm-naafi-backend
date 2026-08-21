package response

import (
	"encoding/json"
	"net/http"

	apperr "github.com/ilmnafi/backend/internal/errors"
)

// APIResponse is the standard envelope returned by all endpoints.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *apiError   `json:"error,omitempty"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{Success: true, Data: data})
}

func WriteSuccess(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusOK, data)
}

func WriteCreated(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusCreated, data)
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   &apiError{Code: code, Message: message},
	})
}

// WriteAppError inspects the error: if it is an *apperr.AppError it uses the
// embedded code/status; otherwise it returns a generic internal error.
func WriteAppError(w http.ResponseWriter, err error) {
	if e, ok := err.(*apperr.AppError); ok {
		WriteError(w, e.Status, e.Code, e.Message)
		return
	}
	WriteError(w, http.StatusInternalServerError, apperr.CodeInternal, "INTERNAL_ERROR")
}
