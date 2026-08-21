package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ilmnafi/backend/internal/config"
	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "user_id"
const SessionIDKey contextKey = "session_id"

func AuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "MISSING_TOKEN", "Missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid authorization header")
				return
			}

			tokenString := parts[1]
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(cfg.JWT.AccessSecret), nil
			})

			if err != nil || !token.Valid {
				writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid or expired token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid token claims")
				return
			}

			userIDStr, ok := claims["sub"].(string)
			if !ok {
				writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid token subject")
				return
			}

			sessionIDStr, ok := claims["session_id"].(string)
			if !ok {
				writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid token session")
				return
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid user ID")
				return
			}

			sessionID, err := uuid.Parse(sessionIDStr)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid session ID")
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, userID)
			ctx = context.WithValue(ctx, SessionIDKey, sessionID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(r *http.Request) (uuid.UUID, bool) {
	userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)
	return userID, ok
}

func GetSessionID(r *http.Request) (uuid.UUID, bool) {
	sessionID, ok := r.Context().Value(SessionIDKey).(uuid.UUID)
	return sessionID, ok
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
