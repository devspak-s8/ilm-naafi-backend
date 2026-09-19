package service

import (
	"context"

	"github.com/ilmnafi/backend/internal/dua/model"
	"github.com/ilmnafi/backend/internal/dua/repository"
	apperr "github.com/ilmnafi/backend/internal/errors"
)

type DuaService interface {
	GetCategories(ctx context.Context) ([]model.DhikrCategory, error)
	GetDuas(ctx context.Context, categoryID int) ([]model.Dua, error)
	GetDua(ctx context.Context, id int) (*model.Dua, error)
	GetDuasBySlug(ctx context.Context, slug string) ([]model.Dua, error)
	GetDuaWithSources(ctx context.Context, id int) (*model.Dua, []model.DuaSource, error)
}

type duaService struct {
	repo repository.DuaRepository
}

func NewDuaService(repo repository.DuaRepository) DuaService {
	return &duaService{repo: repo}
}

func (s *duaService) GetCategories(ctx context.Context) ([]model.DhikrCategory, error) {
	return s.repo.GetCategories(ctx)
}

func (s *duaService) GetDuas(ctx context.Context, categoryID int) ([]model.Dua, error) {
	if categoryID <= 0 {
		return nil, apperr.Validation("INVALID_CATEGORY")
	}
	c, err := s.repo.GetCategories(ctx)
	if err != nil {
		return nil, err
	}
	found := false
	for _, cat := range c {
		if cat.ID == categoryID {
			found = true
			break
		}
	}
	if !found {
		return nil, apperr.NotFound("DUA_CATEGORY_NOT_FOUND")
	}
	return s.repo.GetDuas(ctx, categoryID)
}

func (s *duaService) GetDua(ctx context.Context, id int) (*model.Dua, error) {
	if id <= 0 {
		return nil, apperr.Validation("INVALID_DUA_ID")
	}
	d, err := s.repo.GetDuaByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, apperr.NotFound("DUA_NOT_FOUND")
	}
	return d, nil
}

func (s *duaService) GetDuasBySlug(ctx context.Context, slug string) ([]model.Dua, error) {
	if slug == "" {
		return nil, apperr.Validation("INVALID_SLUG")
	}
	items, err := s.repo.GetDuasBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return nil, apperr.NotFound("DUA_NOT_FOUND")
	}
	return items, nil
}

func (s *duaService) GetDuaWithSources(ctx context.Context, id int) (*model.Dua, []model.DuaSource, error) {
	d, err := s.GetDua(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	var sources []model.DuaSource
	if d.SourceID != nil {
		sources, _ = s.repo.GetSources(ctx, []int{*d.SourceID})
	}
	return d, sources, nil
}
