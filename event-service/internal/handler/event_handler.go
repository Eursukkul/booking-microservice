package handler

import (
	"net/http"
	"strconv"

	"github.com/Eursukkul/booking-microservice/event-service/internal/dto"
	"github.com/Eursukkul/booking-microservice/event-service/internal/models"
	"github.com/Eursukkul/booking-microservice/event-service/internal/service"
	"github.com/labstack/echo/v4"
)

type EventHandler struct {
	svc service.EventService
}

func NewEventHandler(svc service.EventService) *EventHandler {
	return &EventHandler{svc: svc}
}

func (h *EventHandler) RegisterRoutes(g *echo.Group) {
	g.POST("", h.CreateEvent)
	g.GET("", h.ListEvents)
	g.GET("/:id", h.GetEvent)
}

func (h *EventHandler) CreateEvent(c echo.Context) error {
	var req dto.CreateEventRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name == "" || req.MaxSeats <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "name and max_seats (>0) are required")
	}
	if !req.BookingEndAt.After(req.BookingStartAt) {
		return echo.NewHTTPError(http.StatusBadRequest, "booking_end_at must be after booking_start_at")
	}

	event := &models.Event{
		Name:           req.Name,
		MaxSeats:       req.MaxSeats,
		WaitlistLimit:  req.WaitlistLimit,
		Price:          req.Price,
		BookingStartAt: req.BookingStartAt,
		BookingEndAt:   req.BookingEndAt,
	}

	if err := h.svc.CreateEvent(c.Request().Context(), event); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, dto.ToEventResponse(event))
}

func (h *EventHandler) GetEvent(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid event id")
	}

	event, err := h.svc.GetEvent(c.Request().Context(), uint(id))
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "event not found")
	}

	return c.JSON(http.StatusOK, dto.ToEventResponse(event))
}

func (h *EventHandler) ListEvents(c echo.Context) error {
	events, err := h.svc.ListEvents(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resp := make([]dto.EventResponse, len(events))
	for i, e := range events {
		resp[i] = dto.ToEventResponse(&e)
	}

	return c.JSON(http.StatusOK, resp)
}
