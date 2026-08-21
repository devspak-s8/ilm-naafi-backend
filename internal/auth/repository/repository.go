package repository

import (
	"context"

	"github.com/ilmnafi/backend/internal/auth/model"
	"github.com/google/uuid"
)

type AuthRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	CreateProfile(ctx context.Context, profile *model.UserProfile) error
	CreateEmailVerification(ctx context.Context, ev *model.EmailVerification) error
	CreatePasswordReset(ctx context.Context, pr *model.PasswordReset) error
	CreateAuditEvent(ctx context.Context, event *model.AuditEvent) error
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*model.UserProfile, error)
	UpdateUser(ctx context.Context, user *model.User) error
	UpdateProfile(ctx context.Context, profile *model.UserProfile) error
	MarkEmailVerified(ctx context.Context, userID uuid.UUID) error
	InvalidateEmailVerifications(ctx context.Context, userID uuid.UUID) error
	InvalidatePasswordResets(ctx context.Context, userID uuid.UUID) error
	InvalidateSessions(ctx context.Context, userID uuid.UUID) error
	DeleteUser(ctx context.Context, userID uuid.UUID) error
	DeleteProfile(ctx context.Context, userID uuid.UUID) error
	GetActiveVerificationByUserID(ctx context.Context, userID uuid.UUID) (*model.EmailVerification, error)
	GetActiveResetByUserID(ctx context.Context, userID uuid.UUID) (*model.PasswordReset, error)
	GetVerificationByTokenHash(ctx context.Context, tokenHash string) (*model.EmailVerification, error)
	GetPasswordResetByTokenHash(ctx context.Context, tokenHash string) (*model.PasswordReset, error)
	GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]model.Session, error)
}
