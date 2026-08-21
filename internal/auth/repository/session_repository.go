package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/ilmnafi/backend/internal/auth/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type SessionRepository interface {
	CreateSession(ctx context.Context, session *model.Session) error
	GetSessionByID(ctx context.Context, id uuid.UUID) (*model.Session, error)
	GetSessionByRefreshTokenHash(ctx context.Context, tokenHash string) (*model.Session, error)
	GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]model.Session, error)
	RevokeSession(ctx context.Context, id uuid.UUID) error
	RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error
	UpdateSessionExpiry(ctx context.Context, id uuid.UUID, expiresAt time.Time) error
}

type postgresSessionRepository struct {
	db *sqlx.DB
}

func NewPostgresSessionRepository(db *sqlx.DB) SessionRepository {
	return &postgresSessionRepository{db: db}
}

func (r *postgresSessionRepository) CreateSession(ctx context.Context, session *model.Session) error {
	query := `INSERT INTO sessions (id, user_id, device_name, device_type, ip_address, user_agent, created_at, expires_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query,
		session.ID, session.UserID, session.DeviceName, session.DeviceType,
		session.IPAddress, session.UserAgent, session.CreatedAt, session.ExpiresAt)
	return err
}

func (r *postgresSessionRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*model.Session, error) {
	var session model.Session
	query := `SELECT id, user_id, device_name, device_type, ip_address, user_agent, created_at, expires_at, revoked_at FROM sessions WHERE id = $1 AND revoked_at IS NULL AND expires_at > $2`
	err := r.db.GetContext(ctx, &session, query, id, time.Now().UTC())
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *postgresSessionRepository) GetSessionByRefreshTokenHash(ctx context.Context, tokenHash string) (*model.Session, error) {
	var session model.Session
	query := `SELECT id, user_id, device_name, device_type, ip_address, user_agent, created_at, expires_at, revoked_at FROM sessions WHERE refresh_token_hash = $1 AND revoked_at IS NULL AND expires_at > $2`
	err := r.db.GetContext(ctx, &session, query, tokenHash, time.Now().UTC())
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *postgresSessionRepository) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]model.Session, error) {
	var sessions []model.Session
	query := `SELECT id, user_id, device_name, device_type, ip_address, user_agent, created_at, expires_at, revoked_at FROM sessions WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > $2`
	err := r.db.SelectContext(ctx, &sessions, query, userID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *postgresSessionRepository) RevokeSession(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE sessions SET revoked_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, time.Now().UTC())
	return err
}

func (r *postgresSessionRepository) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE sessions SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, userID, time.Now().UTC())
	return err
}

func (r *postgresSessionRepository) UpdateSessionExpiry(ctx context.Context, id uuid.UUID, expiresAt time.Time) error {
	query := `UPDATE sessions SET expires_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, expiresAt)
	return err
}
