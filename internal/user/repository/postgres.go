package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/ilmnafi/backend/internal/user/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type postgresUserRepository struct {
	db *sqlx.DB
}

func NewPostgresUserRepository(db *sqlx.DB) UserRepository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	query := `SELECT id, email, password_hash, email_verified, status, last_login_at, created_at, updated_at, deleted_at FROM users WHERE id = $1 AND deleted_at IS NULL`
	err := r.db.GetContext(ctx, &user, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *postgresUserRepository) GetProfile(ctx context.Context, userID uuid.UUID) (*model.Profile, error) {
	var profile model.Profile
	query := `SELECT id, user_id, name, bio, avatar_url, created_at, updated_at FROM user_profiles WHERE user_id = $1`
	err := r.db.GetContext(ctx, &profile, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *postgresUserRepository) UpdateProfile(ctx context.Context, profile *model.Profile) error {
	query := `UPDATE user_profiles SET name = $2, bio = $3, avatar_url = $4, updated_at = $5 WHERE user_id = $6`
	_, err := r.db.ExecContext(ctx, query,
		profile.Name, profile.Bio, profile.AvatarURL, time.Now().UTC(), profile.UserID)
	return err
}

func (r *postgresUserRepository) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE users SET deleted_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, userID, time.Now().UTC())
	return err
}
