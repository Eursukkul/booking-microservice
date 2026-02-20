package dto

import (
	"time"

	"github.com/Eursukkul/booking-microservice/event-service/internal/models"
)

type EventResponse struct {
	ID             uint      `json:"id"`
	Name           string    `json:"name"`
	MaxSeats       int       `json:"max_seats"`
	WaitlistLimit  int       `json:"waitlist_limit"`
	Price          float64   `json:"price"`
	BookingStartAt time.Time `json:"booking_start_at"`
	BookingEndAt   time.Time `json:"booking_end_at"`
	CreatedAt      time.Time `json:"created_at"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func ToEventResponse(e *models.Event) EventResponse {
	return EventResponse{
		ID:             e.ID,
		Name:           e.Name,
		MaxSeats:       e.MaxSeats,
		WaitlistLimit:  e.WaitlistLimit,
		Price:          e.Price,
		BookingStartAt: e.BookingStartAt,
		BookingEndAt:   e.BookingEndAt,
		CreatedAt:      e.CreatedAt,
	}
}
