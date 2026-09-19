package service

import (
	"context"

	"github.com/ilmnafi/backend/internal/audio/model"
	apperr "github.com/ilmnafi/backend/internal/errors"
)

type AudioService interface {
	GetSurahAudio(ctx context.Context, surahID, reciterID int) ([]model.AudioTrack, error)
	GetAyahAudio(ctx context.Context, ayahID, reciterID int) (*model.AudioTrack, error)
	UpsertReciterPreference(ctx context.Context, userID string, reciterID int, autoPlay, translation bool) error
	GetReciterPreference(ctx context.Context, userID string) (*model.ReciterPreference, error)
}

type audioService struct{}

func NewAudioService() AudioService {
	return &audioService{}
}

func (s *audioService) GetSurahAudio(ctx context.Context, surahID, reciterID int) ([]model.AudioTrack, error) {
	if surahID <= 0 || reciterID <= 0 {
		return nil, apperr.Validation("INVALID_AUDIO_REQUEST")
	}
	// Delegate to quran repository via a shared DB call. For MVP, keep this thin.
	return nil, apperr.NotFound("AUDIO_NOT_FOUND")
}

func (s *audioService) GetAyahAudio(ctx context.Context, ayahID, reciterID int) (*model.AudioTrack, error) {
	if ayahID <= 0 || reciterID <= 0 {
		return nil, apperr.Validation("INVALID_AUDIO_REQUEST")
	}
	return nil, apperr.NotFound("AUDIO_NOT_FOUND")
}

func (s *audioService) UpsertReciterPreference(ctx context.Context, userID string, reciterID int, autoPlay, translation bool) error {
	if userID == "" || reciterID <= 0 {
		return apperr.Validation("INVALID_PREFERENCE_REQUEST")
	}
	return nil
}

func (s *audioService) GetReciterPreference(ctx context.Context, userID string) (*model.ReciterPreference, error) {
	if userID == "" {
		return nil, apperr.Validation("INVALID_USER")
	}
	return nil, nil
}
