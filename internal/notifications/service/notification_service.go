package service

import (
	"context"
	"time"

	apperr "github.com/ilmnafi/backend/internal/errors"
	"github.com/ilmnafi/backend/internal/notifications/model"
)

type NotificationService interface {
	Create(ctx context.Context, userID, notifType, title, body string, data map[string]interface{}) (*model.Notification, error)
	List(ctx context.Context, userID string, limit int) ([]model.Notification, error)
	MarkRead(ctx context.Context, userID string, notifID int) error
}

type notificationService struct{}

func NewNotificationService() NotificationService {
	return &notificationService{}
}

func (s *notificationService) Create(ctx context.Context, userID, notifType, title, body string, data map[string]interface{}) (*model.Notification, error) {
	if userID == "" {
		return nil, apperr.Validation("INVALID_USER")
	}
	return &model.Notification{
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		Body:      body,
		Data:      data,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *notificationService) List(ctx context.Context, userID string, limit int) ([]model.Notification, error) {
	if userID == "" {
		return nil, apperr.Validation("INVALID_USER")
	}
	return nil, nil
}

func (s *notificationService) MarkRead(ctx context.Context, userID string, notifID int) error {
	if userID == "" || notifID <= 0 {
		return apperr.Validation("INVALID_REQUEST")
	}
	return nil
}
