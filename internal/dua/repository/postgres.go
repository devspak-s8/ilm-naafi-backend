package repository

import (
	"context"
	"database/sql"
	"strings"

	"github.com/ilmnafi/backend/internal/dua/model"
	"github.com/jmoiron/sqlx"
)

type DuaRepository interface {
	GetCategories(ctx context.Context) ([]model.DhikrCategory, error)
	GetDuas(ctx context.Context, categoryID int) ([]model.Dua, error)
	GetDuaByID(ctx context.Context, id int) (*model.Dua, error)
	GetDuasBySlug(ctx context.Context, slug string) ([]model.Dua, error)
	GetSources(ctx context.Context, sourceIDs []int) ([]model.DuaSource, error)
}

type postgresDuaRepository struct {
	db *sqlx.DB
}

func NewPostgresDuaRepository(db *sqlx.DB) DuaRepository {
	return &postgresDuaRepository{db: db}
}

func (r *postgresDuaRepository) GetCategories(ctx context.Context) ([]model.DhikrCategory, error) {
	var cats []model.DhikrCategory
	q := `SELECT id, name, arabic_name, description, icon, sort_order, is_active, created_at, updated_at FROM dhikr_categories WHERE is_active = true ORDER BY sort_order ASC`
	err := r.db.SelectContext(ctx, &cats, q)
	return cats, err
}

func (r *postgresDuaRepository) GetDuas(ctx context.Context, categoryID int) ([]model.Dua, error) {
	var items []model.Dua
	q := `SELECT id, category_id, title, arabic_text, transliteration, translation, source_id, audio_url, sort_order, is_active, created_at, updated_at FROM duas WHERE category_id = $1 AND is_active = true ORDER BY sort_order ASC`
	err := r.db.SelectContext(ctx, &items, q, categoryID)
	return items, err
}

func (r *postgresDuaRepository) GetDuaByID(ctx context.Context, id int) (*model.Dua, error) {
	var d model.Dua
	q := `SELECT id, category_id, title, arabic_text, transliteration, translation, source_id, audio_url, sort_order, is_active, created_at, updated_at FROM duas WHERE id = $1`
	err := r.db.GetContext(ctx, &d, q, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &d, err
}

func (r *postgresDuaRepository) GetDuasBySlug(ctx context.Context, slug string) ([]model.Dua, error) {
	slugMap := map[string]string{
		"protection":  "Protection",
		"forgiveness": "Forgiveness",
		"guidance":    "Guidance",
		"family":      "Family",
		"travel":      "Travel",
		"health":      "Health",
		"rizq":        "Rizq",
		"patience":    "Patience",
		"general":     "General",
	}
	name, ok := slugMap[strings.ToLower(slug)]
	if !ok {
		name = strings.ToLower(slug)
	}
	var items []model.Dua
	q := `SELECT d.id, d.category_id, d.title, d.arabic_text, d.transliteration, d.translation, d.source_id, d.audio_url, d.sort_order, d.is_active, d.created_at, d.updated_at
		FROM duas d
		JOIN dhikr_categories c ON d.category_id = c.id
		WHERE c.name = $1 AND d.is_active = true
		ORDER BY d.sort_order ASC`
	err := r.db.SelectContext(ctx, &items, q, name)
	return items, err
}

func (r *postgresDuaRepository) GetSources(ctx context.Context, sourceIDs []int) ([]model.DuaSource, error) {
	if len(sourceIDs) == 0 {
		return nil, nil
	}
	query, args, err := sqlx.In(`SELECT id, collection, reference, source_text, verification_status FROM adhkar_sources WHERE id IN (?)`, sourceIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	var sources []model.DuaSource
	err = r.db.SelectContext(ctx, &sources, query, args...)
	return sources, err
}
