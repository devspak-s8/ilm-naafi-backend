package handler

import (
	"encoding/json"
	"net/http"

	authModel "github.com/ilmnafi/backend/internal/auth/model"
	userSvc "github.com/ilmnafi/backend/internal/user/service"
	"github.com/google/uuid"
	"github.com/ilmnafi/backend/internal/middleware"
)

type UserHandler struct {
	service *userSvc.UserService
}

func NewUserHandler(svc *userSvc.UserService) *UserHandler {
	return &UserHandler{service: svc}
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication")
		return
	}

	user, err := h.service.GetMe(r.Context(), userID)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, authModel.AuthResponse{
		Success: true,
		Data:    user,
	})
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication")
		return
	}

	var req struct {
		Name      string `json:"name"`
		Bio       string `json:"bio"`
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	user, err := h.service.UpdateProfile(r.Context(), userID, req.Name, req.Bio, req.AvatarURL)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, authModel.AuthResponse{
		Success: true,
		Data:    user,
	})
}

func (h *UserHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication")
		return
	}

	if err := h.service.DeleteAccount(r.Context(), userID); err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, authModel.AuthResponse{Success: true})
}

func (h *UserHandler) GetSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication")
		return
	}

	sessions, err := h.service.GetSessions(r.Context(), userID)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, authModel.AuthResponse{
		Success: true,
		Data:    sessions,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, authModel.AuthResponse{
		Success: false,
		Error: &authModel.APIError{
			Code:    code,
			Message: message,
		},
	})
}

func handleError(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*userSvc.AppError); ok {
		writeError(w, appErr.Status, appErr.Code, appErr.Message)
		return
	}
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
}

func getUserID(r *http.Request) (uuid.UUID, bool) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	return userID, ok
}
