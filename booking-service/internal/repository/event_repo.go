package repository

import (
	"context"

	"github.com/Eursukkul/booking-microservice/booking-service/internal/models"
	"gorm.io/gorm"
)

type EventRepository interface {
	FindByID(ctx context.Context, id uint) (*models.Event, error)
	FindByIDForUpdate(ctx context.Context, tx *gorm.DB, id uint) (*models.Event, error)
}

type eventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) FindByID(ctx context.Context, id uint) (*models.Event, error) {
	var event models.Event
	if err := r.db.WithContext(ctx).First(&event, id).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

// FindByIDForUpdate acquires a row-level lock on the event within the given transaction.
func (r *eventRepository) FindByIDForUpdate(ctx context.Context, tx *gorm.DB, id uint) (*models.Event, error) {
	var event models.Event
	if err := tx.WithContext(ctx).
		Set("gorm:query_option", "FOR UPDATE").
		First(&event, id).Error; err != nil {
		return nil, err
	}
	return &event, nil
}
