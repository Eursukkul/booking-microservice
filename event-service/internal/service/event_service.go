package service

import (
	"context"
	"fmt"

	"github.com/Eursukkul/booking-microservice/event-service/internal/models"
	"github.com/Eursukkul/booking-microservice/event-service/internal/repository"
	"github.com/Eursukkul/booking-microservice/event-service/pkg/rabbitmq"
)

type EventService interface {
	CreateEvent(ctx context.Context, event *models.Event) error
	GetEvent(ctx context.Context, id uint) (*models.Event, error)
	ListEvents(ctx context.Context) ([]models.Event, error)
}

type eventService struct {
	repo      repository.EventRepository
	publisher *rabbitmq.Publisher
}

func NewEventService(repo repository.EventRepository, publisher *rabbitmq.Publisher) EventService {
	return &eventService{repo: repo, publisher: publisher}
}

func (s *eventService) CreateEvent(ctx context.Context, event *models.Event) error {
	if err := s.repo.Create(ctx, event); err != nil {
		return fmt.Errorf("create event: %w", err)
	}

	// Publish event.created to RabbitMQ so booking-service can sync
	if s.publisher != nil {
		_ = s.publisher.Publish("event.created", event)
	}

	return nil
}

func (s *eventService) GetEvent(ctx context.Context, id uint) (*models.Event, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *eventService) ListEvents(ctx context.Context) ([]models.Event, error) {
	return s.repo.FindAll(ctx)
}
