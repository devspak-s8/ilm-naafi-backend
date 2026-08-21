package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/ilmnafi/backend/internal/adhkar/model"
	"github.com/jmoiron/sqlx"
)

type AdhkarRepository interface {
	GetCategories(ctx context.Context) ([]model.DhikrCategory, error)
	GetCategoryByID(ctx context.Context, id int) (*model.DhikrCategory, error)
	GetCategoryByName(ctx context.Context, name string) (*model.DhikrCategory, error)
	GetAdhkarByCategory(ctx context.Context, categoryID int) ([]model.Dhikr, error)
	GetAdhkarBySlug(ctx context.Context, slug string) ([]model.Dhikr, error)
	GetDhikrByID(ctx context.Context, id int) (*model.Dhikr, error)
	GetSourcesByDhikr(ctx context.Context, dhikrID int) ([]model.DhikrSource, error)
	CreateCompletion(ctx context.Context, c *model.AdhkarCompletion) error
	GetCompletion(ctx context.Context, userID string, dhikrID int, date string) (*model.AdhkarCompletion, error)
	UpsertCompletion(ctx context.Context, c *model.AdhkarCompletion) error
	GetDailyCompletion(ctx context.Context, userID string, date string) (map[int]*model.AdhkarCompletion, error)
	CreateReminder(ctx context.Context, r *model.AdhkarReminder) error
	UpdateReminder(ctx context.Context, r *model.AdhkarReminder) error
	GetReminder(ctx context.Context, userID string, categoryID int) (*model.AdhkarReminder, error)
	GetReminders(ctx context.Context, userID string) ([]model.AdhkarReminder, error)
	DeleteReminder(ctx context.Context, userID string, categoryID int) error
	GetActiveAdhkarCount(ctx context.Context) (int, error)
	GetCategoryAdhkarCount(ctx context.Context, categoryID int) (int, error)
	GetCategoryCompletedCount(ctx context.Context, userID string, categoryID int, date string) (int, error)
	GetCompletedAdhkarCount(ctx context.Context, userID string, date string) (int, error)
}

type postgresAdhkarRepository struct {
	db *sqlx.DB
}

func NewPostgresAdhkarRepository(db *sqlx.DB) AdhkarRepository {
	return &postgresAdhkarRepository{db: db}
}

func (r *postgresAdhkarRepository) GetCategories(ctx context.Context) ([]model.DhikrCategory, error) {
	var cats []model.DhikrCategory
	q := `SELECT id, name, arabic_name, description, icon, sort_order, is_active, created_at, updated_at FROM dhikr_categories WHERE is_active = true ORDER BY sort_order ASC`
	err := r.db.SelectContext(ctx, &cats, q)
	return cats, err
}

func (r *postgresAdhkarRepository) GetCategoryByID(ctx context.Context, id int) (*model.DhikrCategory, error) {
	var c model.DhikrCategory
	q := `SELECT id, name, arabic_name, description, icon, sort_order, is_active, created_at, updated_at FROM dhikr_categories WHERE id = $1`
	err := r.db.GetContext(ctx, &c, q, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (r *postgresAdhkarRepository) GetCategoryByName(ctx context.Context, name string) (*model.DhikrCategory, error) {
	var c model.DhikrCategory
	q := `SELECT id, name, arabic_name, description, icon, sort_order, is_active, created_at, updated_at FROM dhikr_categories WHERE name = $1 AND is_active = true LIMIT 1`
	err := r.db.GetContext(ctx, &c, q, name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (r *postgresAdhkarRepository) GetAdhkarByCategory(ctx context.Context, categoryID int) ([]model.Dhikr, error) {
	var items []model.Dhikr
	q := `SELECT id, category_id, title, arabic_text, transliteration, translation, repeat_count, sort_order, audio_url, is_active, created_at, updated_at FROM adhkar WHERE category_id = $1 AND is_active = true ORDER BY sort_order ASC`
	err := r.db.SelectContext(ctx, &items, q, categoryID)
	return items, err
}

func (r *postgresAdhkarRepository) GetAdhkarBySlug(ctx context.Context, slug string) ([]model.Dhikr, error) {
	categoryID, err := r.lookupCategoryBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if categoryID == 0 {
		return nil, nil
	}
	return r.GetAdhkarByCategory(ctx, categoryID)
}

func (r *postgresAdhkarRepository) lookupCategoryBySlug(ctx context.Context, slug string) (int, error) {
	slugMap := map[string]string{
		"morning":     "Morning",
		"evening":     "Evening",
		"post-salah":  "Post-Salah",
		"before-sleep": "Before Sleeping",
		"upon-waking": "Upon Waking",
	}
	name, ok := slugMap[slug]
	if !ok {
		name = slug
	}
	var id int
	q := `SELECT id FROM dhikr_categories WHERE name = $1 AND is_active = true LIMIT 1`
	err := r.db.GetContext(ctx, &id, q, name)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

func (r *postgresAdhkarRepository) GetDhikrByID(ctx context.Context, id int) (*model.Dhikr, error) {
	var d model.Dhikr
	q := `SELECT id, category_id, title, arabic_text, transliteration, translation, repeat_count, sort_order, audio_url, is_active, created_at, updated_at FROM adhkar WHERE id = $1`
	err := r.db.GetContext(ctx, &d, q, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &d, err
}

func (r *postgresAdhkarRepository) GetSourcesByDhikr(ctx context.Context, dhikrID int) ([]model.DhikrSource, error) {
	var s []model.DhikrSource
	q := `SELECT id, dhikr_id, source_type, collection, reference, source_text, verification_status, created_at FROM adhkar_sources WHERE dhikr_id = $1`
	err := r.db.SelectContext(ctx, &s, q, dhikrID)
	return s, err
}

func (r *postgresAdhkarRepository) CreateCompletion(ctx context.Context, c *model.AdhkarCompletion) error {
	q := `INSERT INTO adhkar_completion (user_id, dhikr_id, completion_date, completed_count, required_count, completed_at) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, q, c.UserID, c.DhikrID, c.CompletionDate, c.CompletedCount, c.RequiredCount, c.CompletedAt)
	return err
}

func (r *postgresAdhkarRepository) GetCompletion(ctx context.Context, userID string, dhikrID int, date string) (*model.AdhkarCompletion, error) {
	var c model.AdhkarCompletion
	q := `SELECT id, user_id, dhikr_id, completion_date, completed_count, required_count, completed_at, created_at FROM adhkar_completion WHERE user_id = $1 AND dhikr_id = $2 AND completion_date = $3`
	err := r.db.GetContext(ctx, &c, q, userID, dhikrID, date)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &c, err
}

func (r *postgresAdhkarRepository) UpsertCompletion(ctx context.Context, c *model.AdhkarCompletion) error {
	q := `
		INSERT INTO adhkar_completion (user_id, dhikr_id, completion_date, completed_count, required_count, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id, dhikr_id, completion_date) DO UPDATE SET
			completed_count = EXCLUDED.completed_count,
			required_count = EXCLUDED.required_count,
			completed_at = EXCLUDED.completed_at`
	_, err := r.db.ExecContext(ctx, q, c.UserID, c.DhikrID, c.CompletionDate, c.CompletedCount, c.RequiredCount, c.CompletedAt)
	return err
}

func (r *postgresAdhkarRepository) GetDailyCompletion(ctx context.Context, userID string, date string) (map[int]*model.AdhkarCompletion, error) {
	var items []model.AdhkarCompletion
	q := `SELECT ac.id, ac.user_id, ac.dhikr_id, ac.completion_date, ac.completed_count, ac.required_count, ac.completed_at, ac.created_at
		FROM adhkar_completion ac
		JOIN adhkar a ON ac.dhikr_id = a.id
		WHERE ac.user_id = $1 AND ac.completion_date = $2`
	err := r.db.SelectContext(ctx, &items, q, userID, date)
	if err != nil {
		return nil, err
	}
	m := make(map[int]*model.AdhkarCompletion)
	for i := range items {
		m[items[i].DhikrID] = &items[i]
	}
	return m, nil
}

func (r *postgresAdhkarRepository) CreateReminder(ctx context.Context, rem *model.AdhkarReminder) error {
	q := `INSERT INTO adhkar_reminders (user_id, category_id, enabled, time, notification_type) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id, category_id) DO UPDATE SET enabled = EXCLUDED.enabled, time = EXCLUDED.time, notification_type = EXCLUDED.notification_type, updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, q, rem.UserID, rem.CategoryID, rem.Enabled, rem.Time, rem.NotificationType)
	return err
}

func (r *postgresAdhkarRepository) UpdateReminder(ctx context.Context, rem *model.AdhkarReminder) error {
	q := `UPDATE adhkar_reminders SET enabled = $3, time = $4, notification_type = $5, updated_at = NOW() WHERE user_id = $1 AND category_id = $2`
	_, err := r.db.ExecContext(ctx, q, rem.UserID, rem.CategoryID, rem.Enabled, rem.Time, rem.NotificationType)
	return err
}

func (r *postgresAdhkarRepository) GetReminder(ctx context.Context, userID string, categoryID int) (*model.AdhkarReminder, error) {
	var rem model.AdhkarReminder
	q := `SELECT id, user_id, category_id, enabled, time, notification_type, created_at, updated_at FROM adhkar_reminders WHERE user_id = $1 AND category_id = $2`
	err := r.db.GetContext(ctx, &rem, q, userID, categoryID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &rem, err
}

func (r *postgresAdhkarRepository) GetReminders(ctx context.Context, userID string) ([]model.AdhkarReminder, error) {
	var rems []model.AdhkarReminder
	q := `SELECT id, user_id, category_id, enabled, time, notification_type, created_at, updated_at FROM adhkar_reminders WHERE user_id = $1`
	err := r.db.SelectContext(ctx, &rems, q, userID)
	return rems, err
}

func (r *postgresAdhkarRepository) DeleteReminder(ctx context.Context, userID string, categoryID int) error {
	q := `DELETE FROM adhkar_reminders WHERE user_id = $1 AND category_id = $2`
	_, err := r.db.ExecContext(ctx, q, userID, categoryID)
	return err
}

func (r *postgresAdhkarRepository) GetActiveAdhkarCount(ctx context.Context) (int, error) {
	var n int
	q := `SELECT COUNT(*) FROM adhkar WHERE is_active = true`
	err := r.db.GetContext(ctx, &n, q)
	return n, err
}

func (r *postgresAdhkarRepository) GetCategoryAdhkarCount(ctx context.Context, categoryID int) (int, error) {
	var n int
	q := `SELECT COUNT(*) FROM adhkar WHERE category_id = $1 AND is_active = true`
	err := r.db.GetContext(ctx, &n, q, categoryID)
	return n, err
}

func (r *postgresAdhkarRepository) GetCategoryCompletedCount(ctx context.Context, userID string, categoryID int, date string) (int, error) {
	var n int
	q := `SELECT COUNT(DISTINCT ac.dhikr_id)
		FROM adhkar_completion ac
		JOIN adhkar a ON ac.dhikr_id = a.id
		WHERE ac.user_id = $1 AND a.category_id = $2 AND ac.completion_date = $3 AND ac.completed_count >= ac.required_count`
	err := r.db.GetContext(ctx, &n, q, userID, categoryID, date)
	return n, err
}

func (r *postgresAdhkarRepository) GetCompletedAdhkarCount(ctx context.Context, userID string, date string) (int, error) {
	var n int
	q := `SELECT COUNT(DISTINCT ac.dhikr_id)
		FROM adhkar_completion ac
		WHERE ac.user_id = $1 AND ac.completion_date = $2 AND ac.completed_count >= ac.required_count`
	err := r.db.GetContext(ctx, &n, q, userID, date)
	return n, err
}

var _ = time.Now
