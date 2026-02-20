package models

import "time"

type BookingStatus string

const (
	StatusConfirmed  BookingStatus = "confirmed"
	StatusWaitlisted BookingStatus = "waitlisted"
	StatusCancelled  BookingStatus = "cancelled"
)

type Booking struct {
	ID            uint          `gorm:"primaryKey" json:"id"`
	EventID       uint          `gorm:"not null" json:"event_id"`
	UserID        string        `gorm:"not null" json:"user_id"`
	Status        BookingStatus `gorm:"type:varchar(20);not null;default:'confirmed'" json:"status"`
	WaitlistOrder *int          `json:"waitlist_order,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`

	Event *Event `gorm:"foreignKey:EventID" json:"event,omitempty"`
}
