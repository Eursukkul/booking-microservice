package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Eursukkul/booking-microservice/event-service/internal/dto"
	"github.com/Eursukkul/booking-microservice/event-service/internal/models"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// --- Mock EventService ---

type mockEventService struct {
	createFn func(ctx context.Context, event *models.Event) error
	getFn    func(ctx context.Context, id uint) (*models.Event, error)
	listFn   func(ctx context.Context) ([]models.Event, error)
}

func (m *mockEventService) CreateEvent(ctx context.Context, event *models.Event) error {
	return m.createFn(ctx, event)
}
func (m *mockEventService) GetEvent(ctx context.Context, id uint) (*models.Event, error) {
	return m.getFn(ctx, id)
}
func (m *mockEventService) ListEvents(ctx context.Context) ([]models.Event, error) {
	return m.listFn(ctx)
}

// --- Tests ---

func TestCreateEvent_Handler_Success(t *testing.T) {
	svc := &mockEventService{
		createFn: func(ctx context.Context, event *models.Event) error {
			event.ID = 1
			event.CreatedAt = time.Now()
			return nil
		},
	}

	e := echo.New()
	body := `{"name":"Golang Workshop","max_seats":50,"waitlist_limit":5,"price":2500,"booking_start_at":"2026-02-20T17:00:00Z","booking_end_at":"2026-02-25T17:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := NewEventHandler(svc)
	err := h.CreateEvent(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp dto.EventResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, uint(1), resp.ID)
	assert.Equal(t, "Golang Workshop", resp.Name)
	assert.Equal(t, 50, resp.MaxSeats)
}

func TestCreateEvent_Handler_BadRequest_EmptyName(t *testing.T) {
	e := echo.New()
	body := `{"name":"","max_seats":50,"booking_start_at":"2026-02-20T17:00:00Z","booking_end_at":"2026-02-25T17:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := NewEventHandler(&mockEventService{})
	err := h.CreateEvent(c)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
}

func TestCreateEvent_Handler_BadRequest_InvalidDates(t *testing.T) {
	e := echo.New()
	body := `{"name":"Test","max_seats":50,"booking_start_at":"2026-02-25T17:00:00Z","booking_end_at":"2026-02-20T17:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := NewEventHandler(&mockEventService{})
	err := h.CreateEvent(c)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
}

func TestGetEvent_Handler_Success(t *testing.T) {
	svc := &mockEventService{
		getFn: func(ctx context.Context, id uint) (*models.Event, error) {
			return &models.Event{ID: 1, Name: "Test Event", MaxSeats: 50}, nil
		},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := NewEventHandler(svc)
	err := h.GetEvent(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.EventResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Test Event", resp.Name)
}

func TestGetEvent_Handler_NotFound(t *testing.T) {
	svc := &mockEventService{
		getFn: func(ctx context.Context, id uint) (*models.Event, error) {
			return nil, errors.New("not found")
		},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	h := NewEventHandler(svc)
	err := h.GetEvent(c)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusNotFound, he.Code)
}

func TestGetEvent_Handler_InvalidID(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/abc", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	h := NewEventHandler(&mockEventService{})
	err := h.GetEvent(c)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
}

func TestListEvents_Handler_Success(t *testing.T) {
	svc := &mockEventService{
		listFn: func(ctx context.Context) ([]models.Event, error) {
			return []models.Event{
				{ID: 1, Name: "Event A"},
				{ID: 2, Name: "Event B"},
			}, nil
		},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := NewEventHandler(svc)
	err := h.ListEvents(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []dto.EventResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
}

func TestListEvents_Handler_Error(t *testing.T) {
	svc := &mockEventService{
		listFn: func(ctx context.Context) ([]models.Event, error) {
			return nil, errors.New("db error")
		},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := NewEventHandler(svc)
	err := h.ListEvents(c)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusInternalServerError, he.Code)
}
