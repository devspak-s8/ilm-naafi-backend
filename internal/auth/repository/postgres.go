package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/ilmnafi/backend/internal/auth/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type postgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) AuthRepository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) CreateUser(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, email_verified, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, strings.ToLower(user.Email), user.PasswordHash, user.EmailVerified, user.Status,
		user.CreatedAt, user.UpdatedAt)
	return err
}

func (r *postgresRepository) CreateProfile(ctx context.Context, profile *model.UserProfile) error {
	query := `
		INSERT INTO user_profiles (id, user_id, name, bio, avatar_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query,
		profile.ID, profile.UserID, profile.Name, profile.Bio, profile.AvatarURL,
		profile.CreatedAt, profile.UpdatedAt)
	return err
}

func (r *postgresRepository) CreateEmailVerification(ctx context.Context, ev *model.EmailVerification) error {
	query := `
		INSERT INTO email_verifications (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query,
		ev.ID, ev.UserID, ev.TokenHash, ev.ExpiresAt, ev.CreatedAt)
	return err
}

func (r *postgresRepository) CreatePasswordReset(ctx context.Context, pr *model.PasswordReset) error {
	query := `
		INSERT INTO password_resets (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query,
		pr.ID, pr.UserID, pr.TokenHash, pr.ExpiresAt, pr.CreatedAt)
	return err
}

func (r *postgresRepository) CreateAuditEvent(ctx context.Context, event *model.AuditEvent) error {
	query := `
		INSERT INTO audit_events (id, user_id, event_type, ip_address, user_agent, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query,
		event.ID, event.UserID, event.EventType, event.IPAddress, event.UserAgent,
		event.Metadata, event.CreatedAt)
	return err
}

func (r *postgresRepository) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	query := `SELECT id, email, password_hash, email_verified, status, last_login_at, created_at, updated_at, deleted_at FROM users WHERE email = $1 AND deleted_at IS NULL`
	err := r.db.GetContext(ctx, &user, query, strings.ToLower(email))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *postgresRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
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

func (r *postgresRepository) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (*model.UserProfile, error) {
	var profile model.UserProfile
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

func (r *postgresRepository) UpdateUser(ctx context.Context, user *model.User) error {
	query := `UPDATE users SET email = $2, password_hash = $3, email_verified = $4, status = $5, last_login_at = $6, updated_at = $7 WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.ExecContext(ctx, query,
		user.ID, strings.ToLower(user.Email), user.PasswordHash, user.EmailVerified,
		user.Status, user.LastLoginAt, user.UpdatedAt)
	return err
}

func (r *postgresRepository) UpdateProfile(ctx context.Context, profile *model.UserProfile) error {
	query := `UPDATE user_profiles SET name = $2, bio = $3, avatar_url = $4, updated_at = $5 WHERE user_id = $6`
	_, err := r.db.ExecContext(ctx, query,
		profile.Name, profile.Bio, profile.AvatarURL, profile.UpdatedAt, profile.UserID)
	return err
}

func (r *postgresRepository) MarkEmailVerified(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE users SET email_verified = true, updated_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, userID, time.Now().UTC())
	return err
}

func (r *postgresRepository) InvalidateEmailVerifications(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE email_verifications SET used_at = $2 WHERE user_id = $1 AND used_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, userID, time.Now().UTC())
	return err
}

func (r *postgresRepository) InvalidatePasswordResets(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE password_resets SET used_at = $2 WHERE user_id = $1 AND used_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, userID, time.Now().UTC())
	return err
}

func (r *postgresRepository) InvalidateSessions(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE sessions SET revoked_at = $2 WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, userID, time.Now().UTC())
	return err
}

func (r *postgresRepository) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE users SET deleted_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, userID, time.Now().UTC())
	return err
}

func (r *postgresRepository) DeleteProfile(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM user_profiles WHERE user_id = $1`
	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

func (r *postgresRepository) GetActiveVerificationByUserID(ctx context.Context, userID uuid.UUID) (*model.EmailVerification, error) {
	var ev model.EmailVerification
	query := `SELECT id, user_id, token_hash, expires_at, used_at, created_at FROM email_verifications WHERE user_id = $1 AND used_at IS NULL AND expires_at > $2 ORDER BY created_at DESC LIMIT 1`
	err := r.db.GetContext(ctx, &ev, query, userID, time.Now().UTC())
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

func (r *postgresRepository) GetActiveResetByUserID(ctx context.Context, userID uuid.UUID) (*model.PasswordReset, error) {
	var pr model.PasswordReset
	query := `SELECT id, user_id, token_hash, expires_at, used_at, created_at FROM password_resets WHERE user_id = $1 AND used_at IS NULL AND expires_at > $2 ORDER BY created_at DESC LIMIT 1`
	err := r.db.GetContext(ctx, &pr, query, userID, time.Now().UTC())
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

func (r *postgresRepository) GetVerificationByTokenHash(ctx context.Context, tokenHash string) (*model.EmailVerification, error) {
	var ev model.EmailVerification
	query := `SELECT id, user_id, token_hash, expires_at, used_at, created_at FROM email_verifications WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2`
	err := r.db.GetContext(ctx, &ev, query, tokenHash, time.Now().UTC())
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

func (r *postgresRepository) GetPasswordResetByTokenHash(ctx context.Context, tokenHash string) (*model.PasswordReset, error) {
	var pr model.PasswordReset
	query := `SELECT id, user_id, token_hash, expires_at, used_at, created_at FROM password_resets WHERE token_hash = $1 AND used_at IS NULL AND expires_at > $2`
	err := r.db.GetContext(ctx, &pr, query, tokenHash, time.Now().UTC())
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pr, nil
}

func (r *postgresRepository) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]model.Session, error) {
	var sessions []model.Session
	query := `SELECT id, user_id, device_name, device_type, ip_address, user_agent, created_at, expires_at, revoked_at FROM sessions WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > $2`
	err := r.db.SelectContext(ctx, &sessions, query, userID, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return sessions, nil
}
