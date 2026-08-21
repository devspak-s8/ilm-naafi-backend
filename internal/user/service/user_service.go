package service

import (
	"context"
	"net/http"
	"time"

	authModel "github.com/ilmnafi/backend/internal/auth/model"
	authRepo "github.com/ilmnafi/backend/internal/auth/repository"
	"github.com/ilmnafi/backend/internal/user/model"
	"github.com/ilmnafi/backend/internal/user/repository"
	"github.com/google/uuid"
)

type UserService struct {
	userRepo repository.UserRepository
	authRepo authRepo.AuthRepository
}

func NewUserService(userRepo repository.UserRepository, authRepo authRepo.AuthRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
		authRepo: authRepo,
	}
}

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

func (s *UserService) GetMe(ctx context.Context, userID uuid.UUID) (*model.UserResponse, error) {
	user, err := s.authRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, NewAppError(http.StatusNotFound, "USER_NOT_FOUND", "User not found")
	}

	profile, _ := s.authRepo.GetProfileByUserID(ctx, userID)

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

func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, name, bio, avatarURL string) (*model.UserResponse, error) {
	user, err := s.authRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, NewAppError(http.StatusNotFound, "USER_NOT_FOUND", "User not found")
	}

	profile, _ := s.authRepo.GetProfileByUserID(ctx, userID)
	if profile == nil {
		return nil, NewAppError(http.StatusNotFound, "PROFILE_NOT_FOUND", "Profile not found")
	}

	profile.Name = name
	profile.Bio = bio
	profile.AvatarURL = avatarURL
	profile.UpdatedAt = time.Now().UTC()

	if err := s.authRepo.UpdateProfile(ctx, profile); err != nil {
		return nil, NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update profile")
	}

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

func (s *UserService) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	_ = s.authRepo.InvalidateSessions(ctx, userID)
	_ = s.authRepo.DeleteProfile(ctx, userID)
	_ = s.authRepo.DeleteUser(ctx, userID)

	_ = s.authRepo.CreateAuditEvent(ctx, &authModel.AuditEvent{
		ID:        uuid.New(),
		UserID:    userID,
		EventType: "account_deleted",
		CreatedAt: time.Now().UTC(),
	})

	return nil
}

func (s *UserService) GetSessions(ctx context.Context, userID uuid.UUID) ([]model.SessionResponse, error) {
	sessions, err := s.authRepo.GetSessionsByUserID(ctx, userID)
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
