package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ilmnafi/backend/internal/auth/model"
	"github.com/ilmnafi/backend/internal/auth/service"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/ilmnafi/backend/internal/middleware"
)

var validate = validator.New()

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{service: svc}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	if req.Password != r.URL.Query().Get("password_confirmation") {
		writeError(w, http.StatusBadRequest, "PASSWORD_MISMATCH", "Passwords do not match")
		return
	}

	tokens, userResp, err := h.service.Register(r.Context(), &req, getIP(r), getUserAgent(r))
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, model.AuthResponse{
		Success: true,
		Data: map[string]interface{}{
			"user":         userResp,
			"tokens":       tokens,
		},
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := validate.Struct(req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	tokens, userResp, err := h.service.Login(r.Context(), &req, getIP(r), getUserAgent(r))
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, model.AuthResponse{
		Success: true,
		Data: map[string]interface{}{
			"user":         userResp,
			"tokens":       tokens,
		},
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := getSessionID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing session")
		return
	}

	userID, _ := getUserID(r)

	if err := h.service.Logout(r.Context(), sessionID, userID); err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, model.AuthResponse{Success: true})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req model.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	tokens, err := h.service.Refresh(r.Context(), &req)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, model.AuthResponse{
		Success: true,
		Data:    tokens,
	})
}

func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req model.VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := h.service.VerifyEmail(r.Context(), &req); err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, model.AuthResponse{Success: true})
}

func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req model.ResendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := h.service.ResendVerification(r.Context(), &req); err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, model.AuthResponse{Success: true, Data: map[string]string{"message": "If an account exists for this email, a verification link will be sent"}})
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req model.ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := h.service.ForgotPassword(r.Context(), &req); err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, model.AuthResponse{Success: true, Data: map[string]string{"message": "If an account exists for this email, a password reset link will be sent"}})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req model.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := h.service.ResetPassword(r.Context(), &req); err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, model.AuthResponse{Success: true, Data: map[string]string{"message": "Password has been reset successfully"}})
}

func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
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

	writeJSON(w, http.StatusOK, model.AuthResponse{
		Success: true,
		Data:    user,
	})
}

func (h *AuthHandler) GetSessions(w http.ResponseWriter, r *http.Request) {
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

	writeJSON(w, http.StatusOK, model.AuthResponse{
		Success: true,
		Data:    sessions,
	})
}

func (h *AuthHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication")
		return
	}

	if err := h.service.DeleteAccount(r.Context(), userID); err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, model.AuthResponse{Success: true})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, model.AuthResponse{
		Success: false,
		Error: &model.APIError{
			Code:    code,
			Message: message,
		},
	})
}

func handleError(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*service.AppError); ok {
		writeError(w, appErr.Status, appErr.Code, appErr.Message)
		return
	}
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
}

func getIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	if fwd := r.Header.Get("X-Real-IP"); fwd != "" {
		return fwd
	}
	return r.RemoteAddr
}

func getUserAgent(r *http.Request) string {
	return r.Header.Get("User-Agent")
}

func getUserID(r *http.Request) (uuid.UUID, bool) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	return userID, ok
}

func getSessionID(r *http.Request) (uuid.UUID, bool) {
	sessionID, ok := r.Context().Value(middleware.SessionIDKey).(uuid.UUID)
	return sessionID, ok
}
