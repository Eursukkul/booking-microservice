package dto

import (
	"time"

	"github.com/Eursukkul/booking-microservice/booking-service/internal/models"
)

type BookingResponse struct {
	ID            uint                 `json:"id"`
	EventID       uint                 `json:"event_id"`
	UserID        string               `json:"user_id"`
	Status        models.BookingStatus `json:"status"`
	WaitlistOrder *int                 `json:"waitlist_order,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
}

type EventStatusResponse struct {
	ID             uint      `json:"id"`
	Name           string    `json:"name"`
	MaxSeats       int       `json:"max_seats"`
	WaitlistLimit  int       `json:"waitlist_limit"`
	Price          float64   `json:"price"`
	BookingStartAt time.Time `json:"booking_start_at"`
	BookingEndAt   time.Time `json:"booking_end_at"`
	Confirmed      int64     `json:"confirmed_count"`
	Waitlisted     int64     `json:"waitlisted_count"`
	SeatsAvailable int       `json:"seats_available"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

func ToBookingResponse(b *models.Booking) BookingResponse {
	return BookingResponse{
		ID:            b.ID,
		EventID:       b.EventID,
		UserID:        b.UserID,
		Status:        b.Status,
		WaitlistOrder: b.WaitlistOrder,
		CreatedAt:     b.CreatedAt,
	}
}
