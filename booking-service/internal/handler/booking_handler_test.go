package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Eursukkul/booking-microservice/booking-service/internal/dto"
	"github.com/Eursukkul/booking-microservice/booking-service/internal/models"
	"github.com/Eursukkul/booking-microservice/booking-service/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// --- Mock BookingService ---

type mockBookingService struct {
	createFn func(ctx context.Context, eventID uint, userID string) (*models.Booking, error)
	cancelFn func(ctx context.Context, bookingID uint) (*models.Booking, error)
	getFn    func(ctx context.Context, id uint) (*models.Booking, error)
	listFn   func(ctx context.Context, eventID uint, status *models.BookingStatus) ([]models.Booking, error)
}

func (m *mockBookingService) CreateBooking(ctx context.Context, eventID uint, userID string) (*models.Booking, error) {
	return m.createFn(ctx, eventID, userID)
}
func (m *mockBookingService) CancelBooking(ctx context.Context, bookingID uint) (*models.Booking, error) {
	return m.cancelFn(ctx, bookingID)
}
func (m *mockBookingService) GetBooking(ctx context.Context, id uint) (*models.Booking, error) {
	return m.getFn(ctx, id)
}
func (m *mockBookingService) ListBookings(ctx context.Context, eventID uint, status *models.BookingStatus) ([]models.Booking, error) {
	return m.listFn(ctx, eventID, status)
}

// --- Mock EventRepository ---

type mockEventRepo struct {
	findByIDFn func(ctx context.Context, id uint) (*models.Event, error)
}

func (m *mockEventRepo) FindByID(ctx context.Context, id uint) (*models.Event, error) {
	return m.findByIDFn(ctx, id)
}
func (m *mockEventRepo) FindByIDForUpdate(ctx context.Context, tx *gorm.DB, id uint) (*models.Event, error) {
	return m.findByIDFn(ctx, id)
}

// --- Mock BookingRepository ---

type mockBookingRepo struct {
	countFn func(ctx context.Context, tx *gorm.DB, eventID uint, status models.BookingStatus) (int64, error)
}

func (m *mockBookingRepo) Create(ctx context.Context, tx *gorm.DB, b *models.Booking) error { return nil }
func (m *mockBookingRepo) FindByID(ctx context.Context, id uint) (*models.Booking, error) {
	return nil, nil
}
func (m *mockBookingRepo) FindByEventID(ctx context.Context, eventID uint, status *models.BookingStatus) ([]models.Booking, error) {
	return nil, nil
}
func (m *mockBookingRepo) FindActiveByUserAndEvent(ctx context.Context, tx *gorm.DB, userID string, eventID uint) (*models.Booking, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockBookingRepo) CountByStatus(ctx context.Context, tx *gorm.DB, eventID uint, status models.BookingStatus) (int64, error) {
	if m.countFn != nil {
		return m.countFn(ctx, tx, eventID, status)
	}
	return 0, nil
}
func (m *mockBookingRepo) UpdateStatus(ctx context.Context, tx *gorm.DB, bookingID uint, status models.BookingStatus) error {
	return nil
}
func (m *mockBookingRepo) FindFirstWaitlisted(ctx context.Context, tx *gorm.DB, eventID uint) (*models.Booking, error) {
	return nil, gorm.ErrRecordNotFound
}
func (m *mockBookingRepo) GetDB() *gorm.DB { return nil }

// --- Tests ---

func TestCreateBooking_Handler_Success(t *testing.T) {
	svc := &mockBookingService{
		createFn: func(ctx context.Context, eventID uint, userID string) (*models.Booking, error) {
			return &models.Booking{
				ID:        1,
				EventID:   eventID,
				UserID:    userID,
				Status:    models.StatusConfirmed,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	e := echo.New()
	body := `{"user_id":"user-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/1/bookings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := NewBookingHandler(svc, nil, nil)
	err := h.CreateBooking(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp dto.BookingResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, uint(1), resp.ID)
	assert.Equal(t, models.StatusConfirmed, resp.Status)
	assert.Equal(t, "user-1", resp.UserID)
}

func TestCreateBooking_Handler_Waitlisted(t *testing.T) {
	order := 1
	svc := &mockBookingService{
		createFn: func(ctx context.Context, eventID uint, userID string) (*models.Booking, error) {
			return &models.Booking{
				ID:            2,
				EventID:       eventID,
				UserID:        userID,
				Status:        models.StatusWaitlisted,
				WaitlistOrder: &order,
				CreatedAt:     time.Now(),
			}, nil
		},
	}

	e := echo.New()
	body := `{"user_id":"user-51"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/1/bookings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := NewBookingHandler(svc, nil, nil)
	err := h.CreateBooking(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp dto.BookingResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, models.StatusWaitlisted, resp.Status)
}

func TestCreateBooking_Handler_EmptyUserID(t *testing.T) {
	e := echo.New()
	body := `{"user_id":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/1/bookings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := NewBookingHandler(nil, nil, nil)
	err := h.CreateBooking(c)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
}

func TestCreateBooking_Handler_InvalidEventID(t *testing.T) {
	e := echo.New()
	body := `{"user_id":"user-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/abc/bookings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("abc")

	h := NewBookingHandler(nil, nil, nil)
	err := h.CreateBooking(c)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
}

func TestCreateBooking_Handler_AlreadyBooked(t *testing.T) {
	svc := &mockBookingService{
		createFn: func(ctx context.Context, eventID uint, userID string) (*models.Booking, error) {
			return nil, service.ErrAlreadyBooked
		},
	}

	e := echo.New()
	body := `{"user_id":"user-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/1/bookings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := NewBookingHandler(svc, nil, nil)
	err := h.CreateBooking(c)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusConflict, he.Code)
}

func TestCreateBooking_Handler_FullyBooked(t *testing.T) {
	svc := &mockBookingService{
		createFn: func(ctx context.Context, eventID uint, userID string) (*models.Booking, error) {
			return nil, service.ErrEventFullyBooked
		},
	}

	e := echo.New()
	body := `{"user_id":"user-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/1/bookings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := NewBookingHandler(svc, nil, nil)
	err := h.CreateBooking(c)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusConflict, he.Code)
}

func TestCreateBooking_Handler_EventNotFound(t *testing.T) {
	svc := &mockBookingService{
		createFn: func(ctx context.Context, eventID uint, userID string) (*models.Booking, error) {
			return nil, service.ErrEventNotFound
		},
	}

	e := echo.New()
	body := `{"user_id":"user-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/999/bookings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	h := NewBookingHandler(svc, nil, nil)
	err := h.CreateBooking(c)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusNotFound, he.Code)
}

func TestCreateBooking_Handler_BookingClosed(t *testing.T) {
	svc := &mockBookingService{
		createFn: func(ctx context.Context, eventID uint, userID string) (*models.Booking, error) {
			return nil, service.ErrBookingClosed
		},
	}

	e := echo.New()
	body := `{"user_id":"user-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/events/1/bookings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := NewBookingHandler(svc, nil, nil)
	err := h.CreateBooking(c)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
}

func TestCancelBooking_Handler_Success(t *testing.T) {
	svc := &mockBookingService{
		cancelFn: func(ctx context.Context, bookingID uint) (*models.Booking, error) {
			return &models.Booking{
				ID:      bookingID,
				EventID: 1,
				UserID:  "user-1",
				Status:  models.StatusCancelled,
			}, nil
		},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/bookings/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := NewBookingHandler(svc, nil, nil)
	err := h.CancelBooking(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.BookingResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, models.StatusCancelled, resp.Status)
}

func TestCancelBooking_Handler_NotFound(t *testing.T) {
	svc := &mockBookingService{
		cancelFn: func(ctx context.Context, bookingID uint) (*models.Booking, error) {
			return nil, service.ErrBookingNotFound
		},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/bookings/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	h := NewBookingHandler(svc, nil, nil)
	err := h.CancelBooking(c)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusNotFound, he.Code)
}

func TestGetBooking_Handler_Success(t *testing.T) {
	svc := &mockBookingService{
		getFn: func(ctx context.Context, id uint) (*models.Booking, error) {
			return &models.Booking{
				ID:      1,
				EventID: 1,
				UserID:  "user-1",
				Status:  models.StatusConfirmed,
			}, nil
		},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookings/1", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := NewBookingHandler(svc, nil, nil)
	err := h.GetBooking(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetBooking_Handler_NotFound(t *testing.T) {
	svc := &mockBookingService{
		getFn: func(ctx context.Context, id uint) (*models.Booking, error) {
			return nil, gorm.ErrRecordNotFound
		},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/bookings/999", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("999")

	h := NewBookingHandler(svc, nil, nil)
	err := h.GetBooking(c)

	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusNotFound, he.Code)
}

func TestListBookings_Handler_Success(t *testing.T) {
	svc := &mockBookingService{
		listFn: func(ctx context.Context, eventID uint, status *models.BookingStatus) ([]models.Booking, error) {
			return []models.Booking{
				{ID: 1, EventID: 1, UserID: "user-1", Status: models.StatusConfirmed},
				{ID: 2, EventID: 1, UserID: "user-2", Status: models.StatusConfirmed},
			}, nil
		},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/1/bookings", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := NewBookingHandler(svc, nil, nil)
	err := h.ListBookings(c)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp []dto.BookingResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
}

func TestListBookings_Handler_WithStatusFilter(t *testing.T) {
	var capturedStatus *models.BookingStatus
	svc := &mockBookingService{
		listFn: func(ctx context.Context, eventID uint, status *models.BookingStatus) ([]models.Booking, error) {
			capturedStatus = status
			return []models.Booking{}, nil
		},
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/1/bookings?status=confirmed", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("1")

	h := NewBookingHandler(svc, nil, nil)
	err := h.ListBookings(c)

	assert.NoError(t, err)
	assert.NotNil(t, capturedStatus)
	assert.Equal(t, models.StatusConfirmed, *capturedStatus)
}
