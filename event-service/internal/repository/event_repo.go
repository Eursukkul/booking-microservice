package repository

import (
	"context"

	"github.com/Eursukkul/booking-microservice/event-service/internal/models"
	"gorm.io/gorm"
)

type EventRepository interface {
	Create(ctx context.Context, event *models.Event) error
	FindByID(ctx context.Context, id uint) (*models.Event, error)
	FindAll(ctx context.Context) ([]models.Event, error)
}

type eventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) Create(ctx context.Context, event *models.Event) error {
	return r.db.WithContext(ctx).Create(event).Error
}

func (r *eventRepository) FindByID(ctx context.Context, id uint) (*models.Event, error) {
	var event models.Event
	if err := r.db.WithContext(ctx).First(&event, id).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func (r *eventRepository) FindAll(ctx context.Context) ([]models.Event, error) {
	var events []models.Event
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}
