package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Eursukkul/booking-microservice/event-service/internal/models"
	"github.com/stretchr/testify/assert"
)

// --- Mock EventRepository ---

type mockEventRepo struct {
	createFn   func(ctx context.Context, event *models.Event) error
	findByIDFn func(ctx context.Context, id uint) (*models.Event, error)
	findAllFn  func(ctx context.Context) ([]models.Event, error)
}

func (m *mockEventRepo) Create(ctx context.Context, event *models.Event) error {
	return m.createFn(ctx, event)
}
func (m *mockEventRepo) FindByID(ctx context.Context, id uint) (*models.Event, error) {
	return m.findByIDFn(ctx, id)
}
func (m *mockEventRepo) FindAll(ctx context.Context) ([]models.Event, error) {
	return m.findAllFn(ctx)
}

// --- Tests ---

func sampleEvent() *models.Event {
	return &models.Event{
		Name:           "Golang Workshop Bangkok",
		MaxSeats:       50,
		WaitlistLimit:  5,
		Price:          2500,
		BookingStartAt: time.Date(2026, 2, 20, 17, 0, 0, 0, time.UTC),
		BookingEndAt:   time.Date(2026, 2, 25, 17, 0, 0, 0, time.UTC),
	}
}

func TestCreateEvent_Success(t *testing.T) {
	repo := &mockEventRepo{
		createFn: func(ctx context.Context, event *models.Event) error {
			event.ID = 1
			return nil
		},
	}

	svc := NewEventService(repo, nil) // nil publisher = skip RabbitMQ
	event := sampleEvent()

	err := svc.CreateEvent(context.Background(), event)

	assert.NoError(t, err)
	assert.Equal(t, uint(1), event.ID)
}

func TestCreateEvent_RepoError(t *testing.T) {
	repo := &mockEventRepo{
		createFn: func(ctx context.Context, event *models.Event) error {
			return errors.New("db connection failed")
		},
	}

	svc := NewEventService(repo, nil)
	event := sampleEvent()

	err := svc.CreateEvent(context.Background(), event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db connection failed")
}

func TestGetEvent_Success(t *testing.T) {
	expected := sampleEvent()
	expected.ID = 1

	repo := &mockEventRepo{
		findByIDFn: func(ctx context.Context, id uint) (*models.Event, error) {
			return expected, nil
		},
	}

	svc := NewEventService(repo, nil)
	event, err := svc.GetEvent(context.Background(), 1)

	assert.NoError(t, err)
	assert.Equal(t, "Golang Workshop Bangkok", event.Name)
	assert.Equal(t, 50, event.MaxSeats)
}

func TestGetEvent_NotFound(t *testing.T) {
	repo := &mockEventRepo{
		findByIDFn: func(ctx context.Context, id uint) (*models.Event, error) {
			return nil, errors.New("record not found")
		},
	}

	svc := NewEventService(repo, nil)
	event, err := svc.GetEvent(context.Background(), 999)

	assert.Error(t, err)
	assert.Nil(t, event)
}

func TestListEvents_Success(t *testing.T) {
	repo := &mockEventRepo{
		findAllFn: func(ctx context.Context) ([]models.Event, error) {
			return []models.Event{
				{ID: 1, Name: "Event A", MaxSeats: 50},
				{ID: 2, Name: "Event B", MaxSeats: 30},
			}, nil
		},
	}

	svc := NewEventService(repo, nil)
	events, err := svc.ListEvents(context.Background())

	assert.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "Event A", events[0].Name)
}

func TestListEvents_Empty(t *testing.T) {
	repo := &mockEventRepo{
		findAllFn: func(ctx context.Context) ([]models.Event, error) {
			return []models.Event{}, nil
		},
	}

	svc := NewEventService(repo, nil)
	events, err := svc.ListEvents(context.Background())

	assert.NoError(t, err)
	assert.Empty(t, events)
}
