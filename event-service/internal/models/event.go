package models

import "time"

type Event struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Name           string    `gorm:"not null" json:"name"`
	MaxSeats       int       `gorm:"not null" json:"max_seats"`
	WaitlistLimit  int       `gorm:"not null" json:"waitlist_limit"`
	Price          float64   `gorm:"not null" json:"price"`
	BookingStartAt time.Time `gorm:"not null" json:"booking_start_at"`
	BookingEndAt   time.Time `gorm:"not null" json:"booking_end_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
