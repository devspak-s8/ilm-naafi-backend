package service

import (
	"context"
	"time"

	"github.com/ilmnafi/backend/internal/adhkar/model"
	"github.com/ilmnafi/backend/internal/adhkar/repository"
	apperr "github.com/ilmnafi/backend/internal/errors"
)

type AdhkarService interface {
	GetCategories(ctx context.Context) ([]model.DhikrCategory, error)
	GetCategory(ctx context.Context, id int) (*model.DhikrCategory, error)
	GetAdhkarByCategory(ctx context.Context, categoryID int) ([]model.Dhikr, error)
	GetAdhkarBySlug(ctx context.Context, slug string) ([]model.Dhikr, error)
	GetDhikrWithSources(ctx context.Context, dhikrID int) (*model.Dhikr, []model.DhikrSource, error)
	RecordCompletion(ctx context.Context, userID string, dhikrID, increment int) (*model.AdhkarCompletion, error)
	GetDailyProgress(ctx context.Context, userID string, date string) (*model.DailyProgress, error)
	GetCompletionHistory(ctx context.Context, userID string, days int) ([]model.AdhkarCompletion, error)
	CreateReminder(ctx context.Context, userID string, categoryID int, enabled bool, timeStr, notifType string) error
	UpdateReminder(ctx context.Context, userID string, categoryID int, enabled bool, timeStr, notifType string) error
	GetReminders(ctx context.Context, userID string) ([]model.AdhkarReminder, error)
	GetReminder(ctx context.Context, userID string, categoryID int) (*model.AdhkarReminder, error)
	DeleteReminder(ctx context.Context, userID string, categoryID int) error
}

type adhkarService struct {
	repo repository.AdhkarRepository
}

func NewAdhkarService(repo repository.AdhkarRepository) AdhkarService {
	return &adhkarService{repo: repo}
}

func (s *adhkarService) GetCategories(ctx context.Context) ([]model.DhikrCategory, error) {
	return s.repo.GetCategories(ctx)
}

func (s *adhkarService) GetCategory(ctx context.Context, id int) (*model.DhikrCategory, error) {
	if id <= 0 {
		return nil, apperr.Validation("INVALID_CATEGORY")
	}
	c, err := s.repo.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperr.NotFound("ADHKAR_CATEGORY_NOT_FOUND")
	}
	return c, nil
}

func (s *adhkarService) GetAdhkarByCategory(ctx context.Context, categoryID int) ([]model.Dhikr, error) {
	if categoryID <= 0 {
		return nil, apperr.Validation("INVALID_CATEGORY")
	}
	c, err := s.repo.GetCategoryByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperr.NotFound("ADHKAR_CATEGORY_NOT_FOUND")
	}
	return s.repo.GetAdhkarByCategory(ctx, categoryID)
}

func (s *adhkarService) GetAdhkarBySlug(ctx context.Context, slug string) ([]model.Dhikr, error) {
	if slug == "" {
		return nil, apperr.Validation("INVALID_SLUG")
	}
	items, err := s.repo.GetAdhkarBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if items == nil {
		return nil, apperr.NotFound("ADHKAR_NOT_FOUND")
	}
	return items, nil
}

func (s *adhkarService) GetDhikrWithSources(ctx context.Context, dhikrID int) (*model.Dhikr, []model.DhikrSource, error) {
	if dhikrID <= 0 {
		return nil, nil, apperr.Validation("INVALID_DHIKR")
	}
	d, err := s.repo.GetDhikrByID(ctx, dhikrID)
	if err != nil {
		return nil, nil, err
	}
	if d == nil {
		return nil, nil, apperr.NotFound("ADHKAR_NOT_FOUND")
	}
	sources, err := s.repo.GetSourcesByDhikr(ctx, dhikrID)
	if err != nil {
		return nil, nil, err
	}
	return d, sources, nil
}

func (s *adhkarService) RecordCompletion(ctx context.Context, userID string, dhikrID, increment int) (*model.AdhkarCompletion, error) {
	if userID == "" || dhikrID <= 0 {
		return nil, apperr.Validation("INVALID_COMPLETION_REQUEST")
	}
	if increment <= 0 {
		increment = 1
	}
	d, err := s.repo.GetDhikrByID(ctx, dhikrID)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, apperr.NotFound("ADHKAR_NOT_FOUND")
	}

	date := time.Now().UTC().Format("2006-01-02")
	existing, err := s.repo.GetCompletion(ctx, userID, dhikrID, date)
	if err != nil {
		return nil, err
	}

	required := d.RepeatCount
	if required <= 0 {
		required = 1
	}

	completedCount := increment
	if existing != nil {
		completedCount = existing.CompletedCount + increment
	}
	if completedCount >= required {
		completedCount = required
	}

	var completedAt *string
	if completedCount >= required {
		t := time.Now().UTC().Format(time.RFC3339)
		completedAt = &t
	}

	completion := &model.AdhkarCompletion{
		UserID:         userID,
		DhikrID:        dhikrID,
		CompletionDate: date,
		CompletedCount: completedCount,
		RequiredCount:  required,
		CompletedAt:    completedAt,
	}

	if err := s.repo.UpsertCompletion(ctx, completion); err != nil {
		return nil, err
	}
	return completion, nil
}

func (s *adhkarService) GetDailyProgress(ctx context.Context, userID string, date string) (*model.DailyProgress, error) {
	if userID == "" {
		return nil, apperr.Validation("INVALID_USER")
	}
	if date == "" {
		date = time.Now().UTC().Format("2006-01-02")
	}

	progress := &model.DailyProgress{Date: date}

	morningID, _ := s.categoryIDByName(ctx, "Morning")
	eveningID, _ := s.categoryIDByName(ctx, "Evening")
	postSalahID, _ := s.categoryIDByName(ctx, "Post-Salah")

	progress.MorningCompleted = s.categoryFullyCompleted(ctx, userID, morningID, date)
	progress.EveningCompleted = s.categoryFullyCompleted(ctx, userID, eveningID, date)
	progress.PostSalahCompleted = s.categoryFullyCompleted(ctx, userID, postSalahID, date)

	total, err := s.repo.GetActiveAdhkarCount(ctx)
	if err != nil {
		return nil, err
	}
	if total > 0 {
		done, err := s.repo.GetCompletedAdhkarCount(ctx, userID, date)
		if err != nil {
			return nil, err
		}
		progress.OverallPercent = done * 100 / total
	}
	return progress, nil
}

func (s *adhkarService) categoryIDByName(ctx context.Context, name string) (int, error) {
	c, err := s.repo.GetCategoryByName(ctx, name)
	if err != nil {
		return 0, err
	}
	if c == nil {
		return 0, nil
	}
	return c.ID, nil
}

func (s *adhkarService) categoryFullyCompleted(ctx context.Context, userID string, categoryID int, date string) bool {
	if categoryID == 0 {
		return false
	}
	total, err := s.repo.GetCategoryAdhkarCount(ctx, categoryID)
	if err != nil || total == 0 {
		return false
	}
	done, err := s.repo.GetCategoryCompletedCount(ctx, userID, categoryID, date)
	if err != nil {
		return false
	}
	return done >= total
}

func (s *adhkarService) GetCompletionHistory(ctx context.Context, userID string, days int) ([]model.AdhkarCompletion, error) {
	if userID == "" {
		return nil, apperr.Validation("INVALID_USER")
	}
	if days <= 0 {
		days = 7
	}
	var history []model.AdhkarCompletion
	for i := 0; i < days; i++ {
		date := time.Now().UTC().AddDate(0, 0, -i).Format("2006-01-02")
		m, err := s.repo.GetDailyCompletion(ctx, userID, date)
		if err != nil {
			return nil, err
		}
		for _, c := range m {
			history = append(history, *c)
		}
	}
	return history, nil
}

func (s *adhkarService) CreateReminder(ctx context.Context, userID string, categoryID int, enabled bool, timeStr, notifType string) error {
	if userID == "" || categoryID <= 0 {
		return apperr.Validation("INVALID_REMINDER_REQUEST")
	}
	c, err := s.repo.GetCategoryByID(ctx, categoryID)
	if err != nil {
		return err
	}
	if c == nil {
		return apperr.NotFound("ADHKAR_CATEGORY_NOT_FOUND")
	}
	if notifType == "" {
		notifType = "normal"
	}
	return s.repo.CreateReminder(ctx, &model.AdhkarReminder{
		UserID:           userID,
		CategoryID:       categoryID,
		Enabled:          enabled,
		Time:             timeStr,
		NotificationType: notifType,
	})
}

func (s *adhkarService) UpdateReminder(ctx context.Context, userID string, categoryID int, enabled bool, timeStr, notifType string) error {
	if userID == "" || categoryID <= 0 {
		return apperr.Validation("INVALID_REMINDER_REQUEST")
	}
	if notifType == "" {
		notifType = "normal"
	}
	return s.repo.UpdateReminder(ctx, &model.AdhkarReminder{
		UserID:           userID,
		CategoryID:       categoryID,
		Enabled:          enabled,
		Time:             timeStr,
		NotificationType: notifType,
	})
}

func (s *adhkarService) GetReminders(ctx context.Context, userID string) ([]model.AdhkarReminder, error) {
	if userID == "" {
		return nil, apperr.Validation("INVALID_USER")
	}
	return s.repo.GetReminders(ctx, userID)
}

func (s *adhkarService) GetReminder(ctx context.Context, userID string, categoryID int) (*model.AdhkarReminder, error) {
	if userID == "" || categoryID <= 0 {
		return nil, apperr.Validation("INVALID_REMINDER_REQUEST")
	}
	return s.repo.GetReminder(ctx, userID, categoryID)
}

func (s *adhkarService) DeleteReminder(ctx context.Context, userID string, categoryID int) error {
	if userID == "" || categoryID <= 0 {
		return apperr.Validation("INVALID_REMINDER_REQUEST")
	}
	return s.repo.DeleteReminder(ctx, userID, categoryID)
}
