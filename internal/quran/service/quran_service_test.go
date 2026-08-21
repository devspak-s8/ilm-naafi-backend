package quran_test

import (
	"context"
	"testing"

	"github.com/ilmnafi/backend/internal/quran/model"
	"github.com/ilmnafi/backend/internal/quran/repository"
	"github.com/ilmnafi/backend/internal/quran/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	surahs []model.Surah
	ayahs  []model.Ayah
}

func (f *fakeRepo) GetSurahs(ctx context.Context) ([]model.Surah, error)              { return f.surahs, nil }
func (f *fakeRepo) GetSurahByID(ctx context.Context, id int) (*model.Surah, error)   { return &f.surahs[id-1], nil }
func (f *fakeRepo) GetSurahByNumber(ctx context.Context, number int) (*model.Surah, error) { return nil, nil }
func (f *fakeRepo) GetAyahsBySurahID(ctx context.Context, surahID int) ([]model.Ayah, error) { return f.ayahs, nil }
func (f *fakeRepo) GetAyahByID(ctx context.Context, id int) (*model.Ayah, error)     { return nil, nil }
func (f *fakeRepo) GetAyahBySurahAndNumber(ctx context.Context, surahID, ayahNumber int) (*model.Ayah, error) { return nil, nil }
func (f *fakeRepo) GetJuz(ctx context.Context, juzID int) ([]model.Ayah, error)      { return nil, nil }
func (f *fakeRepo) GetAyahsByPage(ctx context.Context, pageNumber int) ([]model.Ayah, error) { return nil, nil }
func (f *fakeRepo) SearchAyahs(ctx context.Context, query string, limit int) ([]model.SearchResult, error) { return nil, nil }
func (f *fakeRepo) GetReciters(ctx context.Context) ([]model.Reciter, error)          { return nil, nil }
func (f *fakeRepo) GetReciterByID(ctx context.Context, id int) (*model.Reciter, error){ return nil, nil }
func (f *fakeRepo) GetAudioMetadata(ctx context.Context, surahID int, reciterID int) ([]model.AudioMetadata, error) { return nil, nil }
func (f *fakeRepo) GetAudioByAyahID(ctx context.Context, ayahID int, reciterID int) (*model.AudioMetadata, error) { return nil, nil }
func (f *fakeRepo) CreateBookmark(ctx context.Context, userID string, ayahID int) error { return nil }
func (f *fakeRepo) GetBookmarks(ctx context.Context, userID string) ([]model.Bookmark, error) { return nil, nil }
func (f *fakeRepo) DeleteBookmark(ctx context.Context, userID string, ayahID int) error { return nil }
func (f *fakeRepo) UpsertReadingProgress(ctx context.Context, userID string, surahID, ayahID int, pageNumber *int) error { return nil }
func (f *fakeRepo) GetReadingProgress(ctx context.Context, userID string) (*model.ReadingProgress, error) { return nil, nil }
func (f *fakeRepo) CreateReadingHistory(ctx context.Context, userID string, surahID, ayahID int) error { return nil }
func (f *fakeRepo) GetContinueReading(ctx context.Context, userID string) (*model.ReadingProgress, error) { return nil, nil }

func TestGetSurah(t *testing.T) {
	repo := &fakeRepo{surahs: []model.Surah{{ID: 1, Number: 1, Name: "Al-Fatihah", ArabicName: "الفاتحة", EnglishName: "The Opener", RevelationType: "Meccan", AyahCount: 7}}}
	svc := service.NewQuranService(repo)
	surah, ayahs, err := svc.GetSurah(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "Al-Fatihah", surah.Name)
	assert.Len(t, ayahs, 0)
}

func TestGetSurahNotFound(t *testing.T) {
	repo := &fakeRepo{surahs: []model.Surah{}}
	svc := service.NewQuranService(repo)
	_, _, err := svc.GetSurah(context.Background(), 99)
	require.Error(t, err)
}

func TestSearchValidation(t *testing.T) {
	repo := &fakeRepo{}
	svc := service.NewQuranService(repo)
	_, err := svc.Search(context.Background(), "   ", 10)
	require.Error(t, err)
}

var _ repository.QuranRepository = (*fakeRepo)(nil)
