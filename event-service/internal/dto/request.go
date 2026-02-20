package dto

import "time"

type CreateEventRequest struct {
	Name           string    `json:"name" validate:"required"`
	MaxSeats       int       `json:"max_seats" validate:"required,gt=0"`
	WaitlistLimit  int       `json:"waitlist_limit" validate:"gte=0"`
	Price          float64   `json:"price" validate:"gte=0"`
	BookingStartAt time.Time `json:"booking_start_at" validate:"required"`
	BookingEndAt   time.Time `json:"booking_end_at" validate:"required,gtfield=BookingStartAt"`
}
