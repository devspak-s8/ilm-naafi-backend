package service

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/ilmnafi/backend/internal/auth/model"
	"github.com/ilmnafi/backend/internal/auth/repository"
	"github.com/ilmnafi/backend/internal/config"
	"github.com/ilmnafi/backend/internal/email"
	"github.com/ilmnafi/backend/internal/security"
	"github.com/google/uuid"
	"github.com/golang-jwt/jwt/v5"
)

type AppError struct {
	Code    string
	Message string
	Status  int
}

func NewAppError(status int, code, message string) *AppError {
	return &AppError{Code: code, Message: message, Status: status}
}

func (e *AppError) Error() string {
	return e.Message
}

type AuthService struct {
	repo        repository.AuthRepository
	sessionRepo repository.SessionRepository
	emailSvc    *email.EmailService
	passwordSvc *security.PasswordService
	tokenSvc    *security.TokenService
	cfg         *config.Config
}

func NewAuthService(repo repository.AuthRepository, sessionRepo repository.SessionRepository, emailSvc *email.EmailService, passwordSvc *security.PasswordService, tokenSvc *security.TokenService, cfg *config.Config) *AuthService {
	return &AuthService{
		repo:        repo,
		sessionRepo: sessionRepo,
		emailSvc:    emailSvc,
		passwordSvc: passwordSvc,
		tokenSvc:    tokenSvc,
		cfg:         cfg,
	}
}

func (s *AuthService) Register(ctx context.Context, req *model.RegisterRequest, ip, userAgent string) (*model.TokenPair, *model.UserResponse, error) {
	now := s.tokenSvc.Now()

	existing, _ := s.repo.GetUserByEmail(ctx, req.Email)
	if existing != nil {
		return nil, nil, NewAppError(http.StatusConflict, "EMAIL_ALREADY_EXISTS", "An account with this email already exists")
	}

	passwordHash, err := s.passwordSvc.Hash(req.Password)
	if err != nil {
		return nil, nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to process registration")
	}

	userID := uuid.New()
	profileID := uuid.New()

	user := &model.User{
		ID:            userID,
		Email:         strings.ToLower(req.Email),
		PasswordHash:  passwordHash,
		EmailVerified: false,
		Status:        "active",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	profile := &model.UserProfile{
		ID:        profileID,
		UserID:    userID,
		Name:      req.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create account")
	}

	if err := s.repo.CreateProfile(ctx, profile); err != nil {
		return nil, nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create account")
	}

	verificationToken, _ := s.tokenSvc.GenerateSecureToken(32)
	verificationHash := s.tokenSvc.GenerateSecureHash(verificationToken)

	verification := &model.EmailVerification{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: verificationHash,
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}

	if err := s.repo.CreateEmailVerification(ctx, verification); err != nil {
		return nil, nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create account")
	}

	auditEvent := &model.AuditEvent{
		ID:        uuid.New(),
		UserID:    userID,
		EventType: "registration",
		IPAddress: ip,
		UserAgent: userAgent,
		CreatedAt: now,
	}
	_ = s.repo.CreateAuditEvent(ctx, auditEvent)

	go s.emailSvc.SendVerificationEmail(user.Email, verificationToken)

	tokens, err := s.createSession(ctx, user.ID, "", "", ip, userAgent)
	if err != nil {
		return nil, nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create session")
	}

	userResp := &model.UserResponse{
		ID:            user.ID.String(),
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Name:          profile.Name,
		Status:        user.Status,
		LastLoginAt:   user.LastLoginAt,
		CreatedAt:     user.CreatedAt,
	}

	return tokens, userResp, nil
}

func (s *AuthService) Login(ctx context.Context, req *model.LoginRequest, ip, userAgent string) (*model.TokenPair, *model.UserResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
	}
	if user == nil {
		return nil, nil, NewAppError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
	}

	if user.Status != "active" {
		return nil, nil, NewAppError(http.StatusForbidden, "ACCOUNT_DISABLED", "Your account has been disabled")
	}

	valid, err := s.passwordSvc.Verify(req.Password, user.PasswordHash)
	if err != nil {
		return nil, nil, NewAppError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
	}
	if !valid {
		_ = s.repo.CreateAuditEvent(ctx, &model.AuditEvent{
			ID:        uuid.New(),
			UserID:    user.ID,
			EventType: "login_failed",
			IPAddress: ip,
			UserAgent: userAgent,
			CreatedAt: s.tokenSvc.Now(),
		})
		return nil, nil, NewAppError(http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
	}

	now := s.tokenSvc.Now()
	user.LastLoginAt = &now
	user.UpdatedAt = now

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return nil, nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to login")
	}

	profile, _ := s.repo.GetProfileByUserID(ctx, user.ID)

	tokens, err := s.createSession(ctx, user.ID, "", "", ip, userAgent)
	if err != nil {
		return nil, nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create session")
	}

	_ = s.repo.CreateAuditEvent(ctx, &model.AuditEvent{
		ID:        uuid.New(),
		UserID:    user.ID,
		EventType: "login_success",
		IPAddress: ip,
		UserAgent: userAgent,
		CreatedAt: now,
	})

	userResp := &model.UserResponse{
		ID:            user.ID.String(),
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Name:          profile.Name,
		Status:        user.Status,
		LastLoginAt:   user.LastLoginAt,
		CreatedAt:     user.CreatedAt,
	}

	return tokens, userResp, nil
}

func (s *AuthService) Logout(ctx context.Context, sessionID uuid.UUID, userID uuid.UUID) error {
	return s.sessionRepo.RevokeSession(ctx, sessionID)
}

func (s *AuthService) Refresh(ctx context.Context, req *model.RefreshTokenRequest) (*model.TokenPair, error) {
	tokenHash := s.tokenSvc.GenerateSecureHash(req.RefreshToken)
	session, err := s.sessionRepo.GetSessionByRefreshTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, NewAppError(http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Invalid refresh token")
	}
	if session == nil {
		return nil, NewAppError(http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Invalid refresh token")
	}

	_ = s.sessionRepo.RevokeSession(ctx, session.ID)

	tokens, err := s.createSession(ctx, session.UserID, session.DeviceName, session.DeviceType, session.IPAddress, session.UserAgent)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to refresh token")
	}

	_ = s.repo.CreateAuditEvent(ctx, &model.AuditEvent{
		ID:        uuid.New(),
		UserID:    session.UserID,
		EventType: "token_refresh",
		IPAddress: session.IPAddress,
		UserAgent: session.UserAgent,
		CreatedAt: s.tokenSvc.Now(),
	})

	return tokens, nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, req *model.VerifyEmailRequest) error {
	tokenHash := s.tokenSvc.GenerateSecureHash(req.Token)
	ev, err := s.repo.GetVerificationByTokenHash(ctx, tokenHash)
	if err != nil {
		return NewAppError(http.StatusBadRequest, "INVALID_TOKEN", "Invalid or expired verification token")
	}
	if ev == nil {
		return NewAppError(http.StatusBadRequest, "INVALID_TOKEN", "Invalid or expired verification token")
	}

	_ = s.repo.MarkEmailVerified(ctx, ev.UserID)
	_ = s.repo.InvalidateEmailVerifications(ctx, ev.UserID)

	_ = s.repo.CreateAuditEvent(ctx, &model.AuditEvent{
		ID:        uuid.New(),
		UserID:    ev.UserID,
		EventType: "email_verified",
		CreatedAt: s.tokenSvc.Now(),
	})

	return nil
}

func (s *AuthService) ResendVerification(ctx context.Context, req *model.ResendVerificationRequest) error {
	user, _ := s.repo.GetUserByEmail(ctx, req.Email)
	if user == nil || user.EmailVerified {
		return nil
	}

	_ = s.repo.InvalidateEmailVerifications(ctx, user.ID)

	token, _ := s.tokenSvc.GenerateSecureToken(32)
	tokenHash := s.tokenSvc.GenerateSecureHash(token)

	verification := &model.EmailVerification{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: s.tokenSvc.Now().Add(24 * time.Hour),
		CreatedAt: s.tokenSvc.Now(),
	}
	_ = s.repo.CreateEmailVerification(ctx, verification)
	_ = s.emailSvc.SendVerificationEmail(user.Email, token)

	return nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, req *model.ForgotPasswordRequest) error {
	user, _ := s.repo.GetUserByEmail(ctx, req.Email)
	if user == nil {
		return nil
	}

	_ = s.repo.InvalidatePasswordResets(ctx, user.ID)

	token, _ := s.tokenSvc.GenerateSecureToken(32)
	tokenHash := s.tokenSvc.GenerateSecureHash(token)

	reset := &model.PasswordReset{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: s.tokenSvc.Now().Add(time.Hour),
		CreatedAt: s.tokenSvc.Now(),
	}
	_ = s.repo.CreatePasswordReset(ctx, reset)
	_ = s.emailSvc.SendPasswordResetEmail(user.Email, token)

	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, req *model.ResetPasswordRequest) error {
	tokenHash := s.tokenSvc.GenerateSecureHash(req.Token)
	pr, err := s.repo.GetPasswordResetByTokenHash(ctx, tokenHash)
	if err != nil {
		return NewAppError(http.StatusBadRequest, "INVALID_TOKEN", "Invalid or expired reset token")
	}
	if pr == nil {
		return NewAppError(http.StatusBadRequest, "INVALID_TOKEN", "Invalid or expired reset token")
	}

	if err := security.ValidatePassword(req.Password); err != nil {
		return NewAppError(http.StatusBadRequest, "INVALID_PASSWORD", err.Error())
	}

	passwordHash, err := s.passwordSvc.Hash(req.Password)
	if err != nil {
		return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to reset password")
	}

	user, _ := s.repo.GetUserByID(ctx, pr.UserID)
	if user == nil {
		return NewAppError(http.StatusBadRequest, "INVALID_TOKEN", "Invalid or expired reset token")
	}

	user.PasswordHash = passwordHash
	user.UpdatedAt = s.tokenSvc.Now()
	_ = s.repo.UpdateUser(ctx, user)
	_ = s.repo.InvalidateSessions(ctx, user.ID)
	_ = s.repo.InvalidatePasswordResets(ctx, user.ID)

	_ = s.repo.CreateAuditEvent(ctx, &model.AuditEvent{
		ID:        uuid.New(),
		UserID:    user.ID,
		EventType: "password_reset",
		CreatedAt: s.tokenSvc.Now(),
	})

	return nil
}

func (s *AuthService) GetMe(ctx context.Context, userID uuid.UUID) (*model.UserResponse, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "An error occurred")
	}
	if user == nil {
		return nil, NewAppError(http.StatusNotFound, "USER_NOT_FOUND", "User not found")
	}

	profile, _ := s.repo.GetProfileByUserID(ctx, userID)

	return &model.UserResponse{
		ID:            user.ID.String(),
		Email:         user.Email,
		EmailVerified: user.EmailVerified,
		Name:          profile.Name,
		Status:        user.Status,
		LastLoginAt:   user.LastLoginAt,
		CreatedAt:     user.CreatedAt,
	}, nil
}

func (s *AuthService) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	_ = s.repo.InvalidateSessions(ctx, userID)
	_ = s.repo.DeleteProfile(ctx, userID)
	_ = s.repo.DeleteUser(ctx, userID)

	_ = s.repo.CreateAuditEvent(ctx, &model.AuditEvent{
		ID:        uuid.New(),
		UserID:    userID,
		EventType: "account_deleted",
		CreatedAt: s.tokenSvc.Now(),
	})

	return nil
}

func (s *AuthService) GetSessions(ctx context.Context, userID uuid.UUID) ([]model.SessionResponse, error) {
	sessions, err := s.sessionRepo.GetSessionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var responses []model.SessionResponse
	for _, session := range sessions {
		responses = append(responses, model.SessionResponse{
			ID:         session.ID.String(),
			DeviceName: session.DeviceName,
			DeviceType: session.DeviceType,
			IPAddress:  session.IPAddress,
			CreatedAt:  session.CreatedAt,
			ExpiresAt:  session.ExpiresAt,
			Current:    false,
		})
	}
	return responses, nil
}

func (s *AuthService) createSession(ctx context.Context, userID uuid.UUID, deviceName, deviceType, ip, userAgent string) (*model.TokenPair, error) {
	now := s.tokenSvc.Now()
	sessionID := uuid.New()
	refreshToken, _ := s.tokenSvc.GenerateSecureToken(32)
	refreshHash := s.tokenSvc.GenerateSecureHash(refreshToken)

	if deviceName == "" {
		deviceName = "Unknown"
	}
	if deviceType == "" {
		deviceType = "web"
	}

	session := &model.Session{
		ID:                sessionID,
		UserID:            userID,
		DeviceName:        deviceName,
		DeviceType:        deviceType,
		IPAddress:         ip,
		UserAgent:         userAgent,
		CreatedAt:         now,
		ExpiresAt:         now.Add(s.cfg.Token.RefreshExpiration),
		RefreshTokenHash:  refreshHash,
	}

	if err := s.sessionRepo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	accessToken, err := s.generateAccessToken(userID, session.ID)
	if err != nil {
		return nil, err
	}

	return &model.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.Token.AccessExpiration.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

func (s *AuthService) generateAccessToken(userID, sessionID uuid.UUID) (string, error) {
	claims := jwt.MapClaims{
		"sub":        userID.String(),
		"session_id": sessionID.String(),
		"iss":        s.cfg.JWT.Issuer,
		"iat":        s.tokenSvc.Now().Unix(),
		"exp":        s.tokenSvc.Now().Add(s.cfg.Token.AccessExpiration).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWT.AccessSecret))
}
