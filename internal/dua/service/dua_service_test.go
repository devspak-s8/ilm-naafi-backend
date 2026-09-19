package service_test

import (
	"context"
	"testing"

	"github.com/ilmnafi/backend/internal/dua/model"
	"github.com/ilmnafi/backend/internal/dua/repository"
	"github.com/ilmnafi/backend/internal/dua/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDuaRepo struct {
	categories []model.DhikrCategory
	duas       []model.Dua
}

func (f *fakeDuaRepo) GetCategories(ctx context.Context) ([]model.DhikrCategory, error) {
	return f.categories, nil
}
func (f *fakeDuaRepo) GetDuas(ctx context.Context, categoryID int) ([]model.Dua, error) {
	var out []model.Dua
	for _, d := range f.duas {
		if d.CategoryID == categoryID {
			out = append(out, d)
		}
	}
	return out, nil
}
func (f *fakeDuaRepo) GetDuaByID(ctx context.Context, id int) (*model.Dua, error) {
	for _, d := range f.duas {
		if d.ID == id {
			return &d, nil
		}
	}
	return nil, nil
}
func (f *fakeDuaRepo) GetDuasBySlug(ctx context.Context, slug string) ([]model.Dua, error) {
	return nil, nil
}
func (f *fakeDuaRepo) GetSources(ctx context.Context, sourceIDs []int) ([]model.DuaSource, error) {
	return nil, nil
}

func TestGetDua(t *testing.T) {
	repo := &fakeDuaRepo{
		categories: []model.DhikrCategory{{ID: 1, Name: "Protection", IsActive: true}},
		duas:       []model.Dua{{ID: 1, CategoryID: 1, Title: "Dua 1", IsActive: true}},
	}
	svc := service.NewDuaService(repo)
	d, err := svc.GetDua(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, "Dua 1", d.Title)
}

func TestGetDuaNotFound(t *testing.T) {
	repo := &fakeDuaRepo{}
	svc := service.NewDuaService(repo)
	_, err := svc.GetDua(context.Background(), 99)
	require.Error(t, err)
}

func TestGetDuasValidation(t *testing.T) {
	repo := &fakeDuaRepo{}
	svc := service.NewDuaService(repo)
	_, err := svc.GetDuas(context.Background(), 0)
	require.Error(t, err)
}

var _ repository.DuaRepository = (*fakeDuaRepo)(nil)
