package service

import (
	"context"
	"net/http"
	"strings"

	apperr "github.com/ilmnafi/backend/internal/errors"
	"github.com/ilmnafi/backend/internal/quran/model"
	"github.com/ilmnafi/backend/internal/quran/repository"
)

type QuranService interface {
	GetSurahs(ctx context.Context) ([]model.Surah, error)
	GetSurah(ctx context.Context, id int) (*model.Surah, []model.Ayah, error)
	GetAyahByID(ctx context.Context, id int) (*model.Ayah, error)
	GetAyah(ctx context.Context, surahID, ayahNumber int) (*model.Ayah, error)
	GetJuz(ctx context.Context, juzID int) ([]model.Ayah, error)
	GetPage(ctx context.Context, pageNumber int) ([]model.Ayah, error)
	Search(ctx context.Context, query string, limit int) ([]model.SearchResult, error)
	GetReciters(ctx context.Context) ([]model.Reciter, error)
	GetReciter(ctx context.Context, id int) (*model.Reciter, error)
	GetAudio(ctx context.Context, surahID, reciterID int) ([]model.AudioMetadata, error)
	GetAyahAudio(ctx context.Context, ayahID, reciterID int) (*model.AudioMetadata, error)
	CreateBookmark(ctx context.Context, userID string, ayahID int) error
	GetBookmarks(ctx context.Context, userID string) ([]model.Bookmark, error)
	DeleteBookmark(ctx context.Context, userID string, ayahID int) error
	UpdateProgress(ctx context.Context, userID string, surahID, ayahID int, pageNumber *int) error
	GetProgress(ctx context.Context, userID string) (*model.ReadingProgress, error)
	ContinueReading(ctx context.Context, userID string) (*model.ReadingProgress, error)
	RecordReading(ctx context.Context, userID string, surahID, ayahID int) error
}

type quranService struct {
	repo repository.QuranRepository
}

func NewQuranService(repo repository.QuranRepository) QuranService {
	return &quranService{repo: repo}
}

func (s *quranService) GetSurahs(ctx context.Context) ([]model.Surah, error) {
	return s.repo.GetSurahs(ctx)
}

func (s *quranService) GetSurah(ctx context.Context, id int) (*model.Surah, []model.Ayah, error) {
	surah, err := s.repo.GetSurahByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if surah == nil {
		return nil, nil, apperr.NotFound("SURAH_NOT_FOUND")
	}
	ayahs, err := s.repo.GetAyahsBySurahID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return surah, ayahs, nil
}

func (s *quranService) GetAyahByID(ctx context.Context, id int) (*model.Ayah, error) {
	if id <= 0 {
		return nil, apperr.Validation("INVALID_AYAH_ID")
	}
	ayah, err := s.repo.GetAyahByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if ayah == nil {
		return nil, apperr.NotFound("AYAH_NOT_FOUND")
	}
	return ayah, nil
}

func (s *quranService) GetAyah(ctx context.Context, surahID, ayahNumber int) (*model.Ayah, error) {
	if surahID <= 0 || surahID > 114 || ayahNumber <= 0 {
		return nil, apperr.Validation("INVALID_AYAH_REFERENCE")
	}
	ayah, err := s.repo.GetAyahBySurahAndNumber(ctx, surahID, ayahNumber)
	if err != nil {
		return nil, err
	}
	if ayah == nil {
		return nil, apperr.NotFound("AYAH_NOT_FOUND")
	}
	return ayah, nil
}

func (s *quranService) GetJuz(ctx context.Context, juzID int) ([]model.Ayah, error) {
	if juzID <= 0 || juzID > 30 {
		return nil, apperr.Validation("INVALID_JUZ")
	}
	ayahs, err := s.repo.GetJuz(ctx, juzID)
	if err != nil {
		return nil, err
	}
	if len(ayahs) == 0 {
		return nil, apperr.NotFound("JUZ_NOT_FOUND")
	}
	return ayahs, nil
}

func (s *quranService) GetPage(ctx context.Context, pageNumber int) ([]model.Ayah, error) {
	if pageNumber <= 0 || pageNumber > 604 {
		return nil, apperr.Validation("INVALID_PAGE")
	}
	ayahs, err := s.repo.GetAyahsByPage(ctx, pageNumber)
	if err != nil {
		return nil, err
	}
	if len(ayahs) == 0 {
		return nil, apperr.NotFound("PAGE_NOT_FOUND")
	}
	return ayahs, nil
}

func (s *quranService) Search(ctx context.Context, query string, limit int) ([]model.SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, apperr.Validation("INVALID_QUERY")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.repo.SearchAyahs(ctx, query, limit)
}

func (s *quranService) GetReciters(ctx context.Context) ([]model.Reciter, error) {
	return s.repo.GetReciters(ctx)
}

func (s *quranService) GetReciter(ctx context.Context, id int) (*model.Reciter, error) {
	reciter, err := s.repo.GetReciterByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if reciter == nil {
		return nil, apperr.NotFound("RECITER_NOT_FOUND")
	}
	return reciter, nil
}

func (s *quranService) GetAudio(ctx context.Context, surahID, reciterID int) ([]model.AudioMetadata, error) {
	if surahID <= 0 || reciterID <= 0 {
		return nil, apperr.Validation("INVALID_AUDIO_REQUEST")
	}
	return s.repo.GetAudioMetadata(ctx, surahID, reciterID)
}

func (s *quranService) GetAyahAudio(ctx context.Context, ayahID, reciterID int) (*model.AudioMetadata, error) {
	if ayahID <= 0 || reciterID <= 0 {
		return nil, apperr.Validation("INVALID_AUDIO_REQUEST")
	}
	audio, err := s.repo.GetAudioByAyahID(ctx, ayahID, reciterID)
	if err != nil {
		return nil, err
	}
	if audio == nil {
		return nil, apperr.NotFound("AUDIO_NOT_FOUND")
	}
	return audio, nil
}

func (s *quranService) CreateBookmark(ctx context.Context, userID string, ayahID int) error {
	if userID == "" || ayahID <= 0 {
		return apperr.Validation("INVALID_BOOKMARK_REQUEST")
	}
	ayah, err := s.repo.GetAyahByID(ctx, ayahID)
	if err != nil {
		return err
	}
	if ayah == nil {
		return apperr.NotFound("AYAH_NOT_FOUND")
	}
	return s.repo.CreateBookmark(ctx, userID, ayahID)
}

func (s *quranService) GetBookmarks(ctx context.Context, userID string) ([]model.Bookmark, error) {
	if userID == "" {
		return nil, apperr.Validation("INVALID_USER")
	}
	return s.repo.GetBookmarks(ctx, userID)
}

func (s *quranService) DeleteBookmark(ctx context.Context, userID string, ayahID int) error {
	if userID == "" || ayahID <= 0 {
		return apperr.Validation("INVALID_BOOKMARK_REQUEST")
	}
	return s.repo.DeleteBookmark(ctx, userID, ayahID)
}

func (s *quranService) UpdateProgress(ctx context.Context, userID string, surahID, ayahID int, pageNumber *int) error {
	if userID == "" || surahID <= 0 || ayahID <= 0 {
		return apperr.Validation("INVALID_READING_POSITION")
	}
	ayah, err := s.repo.GetAyahBySurahAndNumber(ctx, surahID, ayahID)
	if err != nil {
		return err
	}
	if ayah == nil {
		return apperr.Validation("INVALID_READING_POSITION")
	}
	if err := s.repo.UpsertReadingProgress(ctx, userID, surahID, ayahID, pageNumber); err != nil {
		return err
	}
	return s.repo.CreateReadingHistory(ctx, userID, surahID, ayahID)
}

func (s *quranService) GetProgress(ctx context.Context, userID string) (*model.ReadingProgress, error) {
	if userID == "" {
		return nil, apperr.Validation("INVALID_USER")
	}
	return s.repo.GetReadingProgress(ctx, userID)
}

func (s *quranService) ContinueReading(ctx context.Context, userID string) (*model.ReadingProgress, error) {
	if userID == "" {
		return nil, apperr.Validation("INVALID_USER")
	}
	return s.repo.GetContinueReading(ctx, userID)
}

func (s *quranService) RecordReading(ctx context.Context, userID string, surahID, ayahID int) error {
	if userID == "" || surahID <= 0 || ayahID <= 0 {
		return apperr.Validation("INVALID_READING_POSITION")
	}
	return s.repo.CreateReadingHistory(ctx, userID, surahID, ayahID)
}
