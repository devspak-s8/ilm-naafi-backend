package service

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	apperr "github.com/ilmnafi/backend/internal/errors"
	"github.com/ilmnafi/backend/internal/quran/provider"
)

type ContentService interface {
	ListChapters(ctx context.Context, language string) ([]provider.Chapter, error)
	GetChapter(ctx context.Context, chapterID int, language string) (*provider.Chapter, error)
	GetVerses(ctx context.Context, chapterID int, query provider.VerseQuery) (provider.VersePage, error)
	GetVerse(ctx context.Context, verseKey string, query provider.VerseQuery) (*provider.Verse, error)
	GetJuz(ctx context.Context, juzID int, mushafID int) (*provider.Juz, error)
	GetPage(ctx context.Context, pageID int, query provider.VerseQuery) (provider.VersePage, error)
	Search(ctx context.Context, query provider.SearchQuery) (provider.SearchResponse, error)
	ListResources(ctx context.Context, resourceType string) ([]provider.Resource, error)
}

type contentService struct{ provider provider.QuranProvider }

func NewContentService(contentProvider provider.QuranProvider) ContentService {
	return &contentService{provider: contentProvider}
}

func (s *contentService) ListChapters(ctx context.Context, language string) ([]provider.Chapter, error) {
	chapters, err := s.provider.ListChapters(ctx, language)
	return chapters, mapProviderError(err)
}

func (s *contentService) GetChapter(ctx context.Context, chapterID int, language string) (*provider.Chapter, error) {
	if chapterID < 1 || chapterID > 114 {
		return nil, apperr.Validation("INVALID_CHAPTER")
	}
	chapter, err := s.provider.GetChapter(ctx, chapterID, language)
	return chapter, mapProviderError(err)
}

func (s *contentService) GetVerses(ctx context.Context, chapterID int, query provider.VerseQuery) (provider.VersePage, error) {
	if chapterID < 1 || chapterID > 114 {
		return provider.VersePage{}, apperr.Validation("INVALID_CHAPTER")
	}
	query = normalizeVerseQuery(query)
	verses, err := s.provider.GetVerses(ctx, chapterID, query)
	return verses, mapProviderError(err)
}

var verseKeyPattern = regexp.MustCompile(`^[1-9][0-9]*:[1-9][0-9]*$`)

func (s *contentService) GetVerse(ctx context.Context, verseKey string, query provider.VerseQuery) (*provider.Verse, error) {
	verseKey = strings.TrimSpace(verseKey)
	if !verseKeyPattern.MatchString(verseKey) {
		return nil, apperr.Validation("INVALID_VERSE_KEY")
	}
	query = normalizeVerseQuery(query)
	verse, err := s.provider.GetVerse(ctx, verseKey, query)
	return verse, mapProviderError(err)
}

func (s *contentService) GetJuz(ctx context.Context, juzID int, mushafID int) (*provider.Juz, error) {
	if juzID < 1 || juzID > 30 {
		return nil, apperr.Validation("INVALID_JUZ")
	}
	if mushafID < 0 {
		return nil, apperr.Validation("INVALID_MUSHAF")
	}
	juz, err := s.provider.GetJuz(ctx, juzID, mushafID)
	return juz, mapProviderError(err)
}

func (s *contentService) GetPage(ctx context.Context, pageID int, query provider.VerseQuery) (provider.VersePage, error) {
	if pageID < 1 || pageID > 1000 {
		return provider.VersePage{}, apperr.Validation("INVALID_PAGE")
	}
	query = normalizeVerseQuery(query)
	page, err := s.provider.GetPage(ctx, pageID, query)
	return page, mapProviderError(err)
}

func (s *contentService) Search(ctx context.Context, query provider.SearchQuery) (provider.SearchResponse, error) {
	query.Query = strings.TrimSpace(query.Query)
	if query.Query == "" {
		return provider.SearchResponse{}, apperr.Validation("INVALID_QUERY")
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Size <= 0 {
		query.Size = 20
	}
	if query.Size > 50 {
		return provider.SearchResponse{}, apperr.Validation("SEARCH_LIMIT_TOO_LARGE")
	}
	results, err := s.provider.Search(ctx, query)
	return results, mapProviderError(err)
}

func (s *contentService) ListResources(ctx context.Context, resourceType string) ([]provider.Resource, error) {
	resourceType = strings.ToLower(strings.TrimSpace(resourceType))
	switch resourceType {
	case "translations", "tafsirs", "recitations":
	default:
		return nil, apperr.Validation("INVALID_RESOURCE_TYPE")
	}
	resources, err := s.provider.ListResources(ctx, resourceType)
	return resources, mapProviderError(err)
}

func normalizeVerseQuery(query provider.VerseQuery) provider.VerseQuery {
	if query.Language == "" {
		query.Language = "en"
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PerPage <= 0 {
		query.PerPage = 20
	}
	if query.PerPage > 50 {
		query.PerPage = 50
	}
	return query
}

func mapProviderError(err error) error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*provider.ProviderError); ok {
		message := e.Code
		if e.Code == "QURAN_RESOURCE_NOT_FOUND" {
			message = "Quran resource not found"
		}
		if e.Code == "QURAN_PROVIDER_UNAVAILABLE" {
			message = "Quran provider is unavailable"
		}
		return apperr.NewAppError(e.Status, e.Code, message)
	}
	return apperr.NewAppError(http.StatusBadGateway, "QURAN_PROVIDER_UNAVAILABLE", fmt.Sprintf("Quran provider request failed: %v", err))
}
