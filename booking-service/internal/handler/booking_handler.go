package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Eursukkul/booking-microservice/booking-service/internal/dto"
	"github.com/Eursukkul/booking-microservice/booking-service/internal/models"
	"github.com/Eursukkul/booking-microservice/booking-service/internal/repository"
	"github.com/Eursukkul/booking-microservice/booking-service/internal/service"
	"github.com/labstack/echo/v4"
)

type BookingHandler struct {
	svc       service.BookingService
	eventRepo repository.EventRepository
	bookRepo  repository.BookingRepository
}

func NewBookingHandler(svc service.BookingService, eventRepo repository.EventRepository, bookRepo repository.BookingRepository) *BookingHandler {
	return &BookingHandler{svc: svc, eventRepo: eventRepo, bookRepo: bookRepo}
}

func (h *BookingHandler) RegisterRoutes(e *echo.Echo) {
	events := e.Group("/api/v1/events")
	events.GET("/:id/status", h.GetEventStatus)
	events.POST("/:id/bookings", h.CreateBooking)
	events.GET("/:id/bookings", h.ListBookings)

	e.GET("/api/v1/bookings/:id", h.GetBooking)
	e.DELETE("/api/v1/bookings/:id", h.CancelBooking)
}

func (h *BookingHandler) CreateBooking(c echo.Context) error {
	eventID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	var req dto.CreateBookingRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.UserID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_id is required")
	}

	booking, err := h.svc.CreateBooking(c.Request().Context(), uint(eventID), req.UserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEventNotFound):
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		case errors.Is(err, service.ErrBookingClosed):
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrAlreadyBooked):
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		case errors.Is(err, service.ErrEventFullyBooked):
			return echo.NewHTTPError(http.StatusConflict, err.Error())
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	return c.JSON(http.StatusCreated, dto.ToBookingResponse(booking))
}

func (h *BookingHandler) CancelBooking(c echo.Context) error {
	bookingID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid booking id")
	}

	booking, err := h.svc.CancelBooking(c.Request().Context(), uint(bookingID))
	if err != nil {
		if errors.Is(err, service.ErrBookingNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, err.Error())
		}
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, dto.ToBookingResponse(booking))
}

func (h *BookingHandler) GetBooking(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid booking id")
	}

	booking, err := h.svc.GetBooking(c.Request().Context(), uint(id))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "booking not found")
	}

	return c.JSON(http.StatusOK, dto.ToBookingResponse(booking))
}

func (h *BookingHandler) ListBookings(c echo.Context) error {
	eventID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	var status *models.BookingStatus
	if s := c.QueryParam("status"); s != "" {
		bs := models.BookingStatus(s)
		status = &bs
	}

	bookings, err := h.svc.ListBookings(c.Request().Context(), uint(eventID), status)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resp := make([]dto.BookingResponse, len(bookings))
	for i, b := range bookings {
		resp[i] = dto.ToBookingResponse(&b)
	}

	return c.JSON(http.StatusOK, resp)
}

func (h *BookingHandler) GetEventStatus(c echo.Context) error {
	eventID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	event, err := h.eventRepo.FindByID(c.Request().Context(), uint(eventID))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "event not found")
	}

	ctx := c.Request().Context()
	confirmed, _ := h.bookRepo.CountByStatus(ctx, h.bookRepo.GetDB(), event.ID, models.StatusConfirmed)
	waitlisted, _ := h.bookRepo.CountByStatus(ctx, h.bookRepo.GetDB(), event.ID, models.StatusWaitlisted)

	return c.JSON(http.StatusOK, dto.EventStatusResponse{
		ID:             event.ID,
		Name:           event.Name,
		MaxSeats:       event.MaxSeats,
		WaitlistLimit:  event.WaitlistLimit,
		Price:          event.Price,
		BookingStartAt: event.BookingStartAt,
		BookingEndAt:   event.BookingEndAt,
		Confirmed:      confirmed,
		Waitlisted:     waitlisted,
		SeatsAvailable: event.MaxSeats - int(confirmed),
	})
}
