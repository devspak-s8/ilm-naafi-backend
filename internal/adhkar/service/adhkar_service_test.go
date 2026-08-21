package adhkar_test

import (
	"context"
	"testing"
	"time"

	"github.com/ilmnafi/backend/internal/adhkar/model"
	"github.com/ilmnafi/backend/internal/adhkar/repository"
	"github.com/ilmnafi/backend/internal/adhkar/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAdhkarRepo struct {
	categories []model.DhikrCategory
	adhkar     []model.Dhikr
	completions map[string]*model.AdhkarCompletion
}

func (f *fakeAdhkarRepo) GetCategories(ctx context.Context) ([]model.DhikrCategory, error) { return f.categories, nil }
func (f *fakeAdhkarRepo) GetCategoryByID(ctx context.Context, id int) (*model.DhikrCategory, error) {
	for _, c := range f.categories { if c.ID == id { return &c, nil } }
	return nil, nil
}
func (f *fakeAdhkarRepo) GetCategoryByName(ctx context.Context, name string) (*model.DhikrCategory, error) {
	for _, c := range f.categories { if c.Name == name { return &c, nil } }
	return nil, nil
}
func (f *fakeAdhkarRepo) GetAdhkarByCategory(ctx context.Context, categoryID int) ([]model.Dhikr, error) {
	var out []model.Dhikr
	for _, a := range f.adhkar { if a.CategoryID == categoryID { out = append(out, a) } }
	return out, nil
}
func (f *fakeAdhkarRepo) GetAdhkarBySlug(ctx context.Context, slug string) ([]model.Dhikr, error) { return nil, nil }
func (f *fakeAdhkarRepo) GetDhikrByID(ctx context.Context, id int) (*model.Dhikr, error) {
	for _, a := range f.adhkar { if a.ID == id { return &a, nil } }
	return nil, nil
}
func (f *fakeAdhkarRepo) GetSourcesByDhikr(ctx context.Context, dhikrID int) ([]model.DhikrSource, error) { return nil, nil }
func (f *fakeAdhkarRepo) CreateCompletion(ctx context.Context, c *model.AdhkarCompletion) error { return nil }
func (f *fakeAdhkarRepo) GetCompletion(ctx context.Context, userID string, dhikrID int, date string) (*model.AdhkarCompletion, error) {
	key := userID + ":" + string(rune(dhikrID)) + ":" + date
	if v, ok := f.completions[key]; ok { return v, nil }
	return nil, nil
}
func (f *fakeAdhkarRepo) UpsertCompletion(ctx context.Context, c *model.AdhkarCompletion) error {
	key := c.UserID + ":" + string(rune(c.DhikrID)) + ":" + c.CompletionDate
	f.completions[key] = c
	return nil
}
func (f *fakeAdhkarRepo) GetDailyCompletion(ctx context.Context, userID string, date string) (map[int]*model.AdhkarCompletion, error) {
	m := make(map[int]*model.AdhkarCompletion)
	for k, v := range f.completions {
		if len(k) > 0 && v.CompletionDate == date {
			m[v.DhikrID] = v
		}
	}
	return m, nil
}
func (f *fakeAdhkarRepo) CreateReminder(ctx context.Context, rem *model.AdhkarReminder) error { return nil }
func (f *fakeAdhkarRepo) UpdateReminder(ctx context.Context, rem *model.AdhkarReminder) error { return nil }
func (f *fakeAdhkarRepo) GetReminder(ctx context.Context, userID string, categoryID int) (*model.AdhkarReminder, error) { return nil, nil }
func (f *fakeAdhkarRepo) GetReminders(ctx context.Context, userID string) ([]model.AdhkarReminder, error) { return nil, nil }
func (f *fakeAdhkarRepo) DeleteReminder(ctx context.Context, userID string, categoryID int) error { return nil }
func (f *fakeAdhkarRepo) GetActiveAdhkarCount(ctx context.Context) (int, error) { return len(f.adhkar), nil }
func (f *fakeAdhkarRepo) GetCategoryAdhkarCount(ctx context.Context, categoryID int) (int, error) {
	n := 0
	for _, a := range f.adhkar { if a.CategoryID == categoryID { n++ } }
	return n, nil
}
func (f *fakeAdhkarRepo) GetCategoryCompletedCount(ctx context.Context, userID string, categoryID int, date string) (int, error) {
	m, _ := f.GetDailyCompletion(ctx, userID, date)
	n := 0
	for _, c := range m {
		for _, a := range f.adhkar {
			if a.ID == c.DhikrID && a.CategoryID == categoryID && c.CompletedCount >= c.RequiredCount {
				n++
			}
		}
	}
	return n, nil
}
func (f *fakeAdhkarRepo) GetCompletedAdhkarCount(ctx context.Context, userID string, date string) (int, error) {
	m, _ := f.GetDailyCompletion(ctx, userID, date)
	return len(m), nil
}

func TestRecordCompletion(t *testing.T) {
	repo := &fakeAdhkarRepo{
		categories: []model.DhikrCategory{{ID: 1, Name: "Morning", IsActive: true}},
		adhkar:     []model.Dhikr{{ID: 1, CategoryID: 1, RepeatCount: 3}},
		completions: map[string]*model.AdhkarCompletion{},
	}
	svc := service.NewAdhkarService(repo)
	c, err := svc.RecordCompletion(context.Background(), "user-1", 1, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, c.CompletedCount)
	assert.Equal(t, 3, c.RequiredCount)
}

func TestRecordCompletionReachesRequired(t *testing.T) {
	repo := &fakeAdhkarRepo{
		categories: []model.DhikrCategory{{ID: 1, Name: "Morning", IsActive: true}},
		adhkar:     []model.Dhikr{{ID: 1, CategoryID: 1, RepeatCount: 2}},
		completions: map[string]*model.AdhkarCompletion{},
	}
	svc := service.NewAdhkarService(repo)
	c, err := svc.RecordCompletion(context.Background(), "user-1", 1, 2)
	require.NoError(t, err)
	assert.Equal(t, 2, c.CompletedCount)
	assert.NotNil(t, c.CompletedAt)
}

func TestGetDailyProgress(t *testing.T) {
	repo := &fakeAdhkarRepo{
		categories: []model.DhikrCategory{
			{ID: 1, Name: "Morning", IsActive: true},
			{ID: 2, Name: "Evening", IsActive: true},
			{ID: 3, Name: "Post-Salah", IsActive: true},
		},
		adhkar: []model.Dhikr{
			{ID: 1, CategoryID: 1, RepeatCount: 1},
			{ID: 2, CategoryID: 2, RepeatCount: 1},
		},
		completions: map[string]*model.AdhkarCompletion{},
	}
	svc := service.NewAdhkarService(repo)
	p, err := svc.GetDailyProgress(context.Background(), "user-1", time.Now().UTC().Format("2006-01-02"))
	require.NoError(t, err)
	assert.Equal(t, "2006-01-02", p.Date)
}

var _ repository.AdhkarRepository = (*fakeAdhkarRepo)(nil)
