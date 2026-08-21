package repository

import (
	"context"
	"github.com/ilmnafi/backend/internal/user/model"
	"github.com/google/uuid"
)

type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*model.Profile, error)
	UpdateProfile(ctx context.Context, profile *model.Profile) error
	DeleteUser(ctx context.Context, userID uuid.UUID) error
}
