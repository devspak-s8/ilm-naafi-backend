package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/ilmnafi/backend/internal/quran/model"
	"github.com/jmoiron/sqlx"
)

type QuranRepository interface {
	GetSurahs(ctx context.Context) ([]model.Surah, error)
	GetSurahByID(ctx context.Context, id int) (*model.Surah, error)
	GetSurahByNumber(ctx context.Context, number int) (*model.Surah, error)
	GetAyahsBySurahID(ctx context.Context, surahID int) ([]model.Ayah, error)
	GetAyahByID(ctx context.Context, id int) (*model.Ayah, error)
	GetAyahBySurahAndNumber(ctx context.Context, surahID, ayahNumber int) (*model.Ayah, error)
	GetJuz(ctx context.Context, juzID int) ([]model.Ayah, error)
	GetAyahsByPage(ctx context.Context, pageNumber int) ([]model.Ayah, error)
	SearchAyahs(ctx context.Context, query string, limit int) ([]model.SearchResult, error)
	GetReciters(ctx context.Context) ([]model.Reciter, error)
	GetReciterByID(ctx context.Context, id int) (*model.Reciter, error)
	GetAudioMetadata(ctx context.Context, surahID int, reciterID int) ([]model.AudioMetadata, error)
	GetAudioByAyahID(ctx context.Context, ayahID int, reciterID int) (*model.AudioMetadata, error)
	CreateBookmark(ctx context.Context, userID string, ayahID int) error
	GetBookmarks(ctx context.Context, userID string) ([]model.Bookmark, error)
	DeleteBookmark(ctx context.Context, userID string, ayahID int) error
	UpsertReadingProgress(ctx context.Context, userID string, surahID, ayahID int, pageNumber *int) error
	GetReadingProgress(ctx context.Context, userID string) (*model.ReadingProgress, error)
	CreateReadingHistory(ctx context.Context, userID string, surahID, ayahID int) error
	GetContinueReading(ctx context.Context, userID string) (*model.ReadingProgress, error)
}

type postgresQuranRepository struct {
	db *sqlx.DB
}

func NewPostgresQuranRepository(db *sqlx.DB) QuranRepository {
	return &postgresQuranRepository{db: db}
}

func (r *postgresQuranRepository) GetSurahs(ctx context.Context) ([]model.Surah, error) {
	var surahs []model.Surah
	query := `SELECT id, number, name, arabic_name, english_name, revelation_type, ayah_count, page_number, created_at, updated_at FROM quran_surahs ORDER BY number ASC`
	err := r.db.SelectContext(ctx, &surahs, query)
	return surahs, err
}

func (r *postgresQuranRepository) GetSurahByID(ctx context.Context, id int) (*model.Surah, error) {
	var surah model.Surah
	query := `SELECT id, number, name, arabic_name, english_name, revelation_type, ayah_count, page_number, created_at, updated_at FROM quran_surahs WHERE id = $1`
	err := r.db.GetContext(ctx, &surah, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &surah, err
}

func (r *postgresQuranRepository) GetSurahByNumber(ctx context.Context, number int) (*model.Surah, error) {
	var surah model.Surah
	query := `SELECT id, number, name, arabic_name, english_name, revelation_type, ayah_count, page_number, created_at, updated_at FROM quran_surahs WHERE number = $1`
	err := r.db.GetContext(ctx, &surah, query, number)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &surah, err
}

func (r *postgresQuranRepository) GetAyahsBySurahID(ctx context.Context, surahID int) ([]model.Ayah, error) {
	var ayahs []model.Ayah
	query := `SELECT id, surah_id, ayah_number, text, page_number, juz_number, hizb_number, rub_el_hizb, sajdah, created_at FROM quran_ayahs WHERE surah_id = $1 ORDER BY ayah_number ASC`
	err := r.db.SelectContext(ctx, &ayahs, query, surahID)
	return ayahs, err
}

func (r *postgresQuranRepository) GetAyahByID(ctx context.Context, id int) (*model.Ayah, error) {
	var ayah model.Ayah
	query := `SELECT id, surah_id, ayah_number, text, page_number, juz_number, hizb_number, rub_el_hizb, sajdah, created_at FROM quran_ayahs WHERE id = $1`
	err := r.db.GetContext(ctx, &ayah, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &ayah, err
}

func (r *postgresQuranRepository) GetAyahBySurahAndNumber(ctx context.Context, surahID, ayahNumber int) (*model.Ayah, error) {
	var ayah model.Ayah
	query := `SELECT id, surah_id, ayah_number, text, page_number, juz_number, hizb_number, rub_el_hizb, sajdah, created_at FROM quran_ayahs WHERE surah_id = $1 AND ayah_number = $2`
	err := r.db.GetContext(ctx, &ayah, query, surahID, ayahNumber)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &ayah, err
}

func (r *postgresQuranRepository) GetJuz(ctx context.Context, juzID int) ([]model.Ayah, error) {
	var ayahs []model.Ayah
	query := `SELECT id, surah_id, ayah_number, text, page_number, juz_number, hizb_number, rub_el_hizb, sajdah, created_at FROM quran_ayahs WHERE juz_number = $1 ORDER BY surah_id, ayah_number ASC`
	err := r.db.SelectContext(ctx, &ayahs, query, juzID)
	return ayahs, err
}

func (r *postgresQuranRepository) GetAyahsByPage(ctx context.Context, pageNumber int) ([]model.Ayah, error) {
	var ayahs []model.Ayah
	query := `SELECT id, surah_id, ayah_number, text, page_number, juz_number, hizb_number, rub_el_hizb, sajdah, created_at FROM quran_ayahs WHERE page_number = $1 ORDER BY surah_id, ayah_number ASC`
	err := r.db.SelectContext(ctx, &ayahs, query, pageNumber)
	return ayahs, err
}

func (r *postgresQuranRepository) SearchAyahs(ctx context.Context, query string, limit int) ([]model.SearchResult, error) {
	var results []model.SearchResult
	searchQuery := `
		SELECT qa.surah_id, qs.name as surah_name, qa.ayah_number, qa.id as ayah_id, qa.text
		FROM quran_ayahs qa
		JOIN quran_surahs qs ON qa.surah_id = qs.id
		WHERE qa.text ILIKE $1
		ORDER BY qa.surah_id, qa.ayah_number
		LIMIT $2`
	err := r.db.SelectContext(ctx, &results, searchQuery, "%"+strings.TrimSpace(query)+"%", limit)
	return results, err
}

func (r *postgresQuranRepository) GetReciters(ctx context.Context) ([]model.Reciter, error) {
	var reciters []model.Reciter
	query := `SELECT id, name, arabic_name, identifier, description, audio_base_url, is_active, created_at, updated_at FROM quran_reciters WHERE is_active = true ORDER BY name ASC`
	err := r.db.SelectContext(ctx, &reciters, query)
	return reciters, err
}

func (r *postgresQuranRepository) GetReciterByID(ctx context.Context, id int) (*model.Reciter, error) {
	var reciter model.Reciter
	query := `SELECT id, name, arabic_name, identifier, description, audio_base_url, is_active, created_at, updated_at FROM quran_reciters WHERE id = $1`
	err := r.db.GetContext(ctx, &reciter, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &reciter, err
}

func (r *postgresQuranRepository) GetAudioMetadata(ctx context.Context, surahID int, reciterID int) ([]model.AudioMetadata, error) {
	var audios []model.AudioMetadata
	query := `SELECT id, reciter_id, surah_id, ayah_id, audio_url, duration_ms, start_ms, end_ms, format, created_at FROM quran_audio WHERE surah_id = $1 AND reciter_id = $2 ORDER BY ayah_id ASC`
	err := r.db.SelectContext(ctx, &audios, query, surahID, reciterID)
	return audios, err
}

func (r *postgresQuranRepository) GetAudioByAyahID(ctx context.Context, ayahID int, reciterID int) (*model.AudioMetadata, error) {
	var audio model.AudioMetadata
	query := `SELECT id, reciter_id, surah_id, ayah_id, audio_url, duration_ms, start_ms, end_ms, format, created_at FROM quran_audio WHERE ayah_id = $1 AND reciter_id = $2`
	err := r.db.GetContext(ctx, &audio, query, ayahID, reciterID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &audio, err
}

func (r *postgresQuranRepository) CreateBookmark(ctx context.Context, userID string, ayahID int) error {
	query := `INSERT INTO quran_bookmarks (user_id, ayah_id) VALUES ($1, $2) ON CONFLICT (user_id, ayah_id) DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, userID, ayahID)
	return err
}

func (r *postgresQuranRepository) GetBookmarks(ctx context.Context, userID string) ([]model.Bookmark, error) {
	var bookmarks []model.Bookmark
	query := `SELECT id, user_id, ayah_id, created_at FROM quran_bookmarks WHERE user_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &bookmarks, query, userID)
	return bookmarks, err
}

func (r *postgresQuranRepository) DeleteBookmark(ctx context.Context, userID string, ayahID int) error {
	query := `DELETE FROM quran_bookmarks WHERE user_id = $1 AND ayah_id = $2`
	_, err := r.db.ExecContext(ctx, query, userID, ayahID)
	return err
}

func (r *postgresQuranRepository) UpsertReadingProgress(ctx context.Context, userID string, surahID, ayahID int, pageNumber *int) error {
	query := `
		INSERT INTO quran_reading_progress (user_id, last_surah_id, last_ayah_id, last_page_number, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			last_surah_id = EXCLUDED.last_surah_id,
			last_ayah_id = EXCLUDED.last_ayah_id,
			last_page_number = EXCLUDED.last_page_number,
			updated_at = EXCLUDED.updated_at`
	_, err := r.db.ExecContext(ctx, query, userID, surahID, ayahID, pageNumber, time.Now().UTC())
	return err
}

func (r *postgresQuranRepository) GetReadingProgress(ctx context.Context, userID string) (*model.ReadingProgress, error) {
	var progress model.ReadingProgress
	query := `SELECT id, user_id, last_surah_id, last_ayah_id, last_page_number, updated_at FROM quran_reading_progress WHERE user_id = $1`
	err := r.db.GetContext(ctx, &progress, query, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &progress, err
}

func (r *postgresQuranRepository) CreateReadingHistory(ctx context.Context, userID string, surahID, ayahID int) error {
	query := `INSERT INTO quran_reading_history (user_id, surah_id, ayah_id) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, query, userID, surahID, ayahID)
	return err
}

func (r *postgresQuranRepository) GetContinueReading(ctx context.Context, userID string) (*model.ReadingProgress, error) {
	return r.GetReadingProgress(ctx, userID)
}
