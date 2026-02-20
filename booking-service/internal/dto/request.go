package dto

type CreateBookingRequest struct {
	UserID string `json:"user_id" validate:"required"`
}
