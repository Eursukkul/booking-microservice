package repository

import (
	"context"

	"github.com/Eursukkul/booking-microservice/booking-service/internal/models"
	"gorm.io/gorm"
)

type BookingRepository interface {
	Create(ctx context.Context, tx *gorm.DB, booking *models.Booking) error
	FindByID(ctx context.Context, id uint) (*models.Booking, error)
	FindByEventID(ctx context.Context, eventID uint, status *models.BookingStatus) ([]models.Booking, error)
	FindActiveByUserAndEvent(ctx context.Context, tx *gorm.DB, userID string, eventID uint) (*models.Booking, error)
	CountByStatus(ctx context.Context, tx *gorm.DB, eventID uint, status models.BookingStatus) (int64, error)
	UpdateStatus(ctx context.Context, tx *gorm.DB, bookingID uint, status models.BookingStatus) error
	FindFirstWaitlisted(ctx context.Context, tx *gorm.DB, eventID uint) (*models.Booking, error)
	GetDB() *gorm.DB
}

type bookingRepository struct {
	db *gorm.DB
}

func NewBookingRepository(db *gorm.DB) BookingRepository {
	return &bookingRepository{db: db}
}

func (r *bookingRepository) GetDB() *gorm.DB {
	return r.db
}

func (r *bookingRepository) Create(ctx context.Context, tx *gorm.DB, booking *models.Booking) error {
	return tx.WithContext(ctx).Create(booking).Error
}

func (r *bookingRepository) FindByID(ctx context.Context, id uint) (*models.Booking, error) {
	var booking models.Booking
	if err := r.db.WithContext(ctx).First(&booking, id).Error; err != nil {
		return nil, err
	}
	return &booking, nil
}

func (r *bookingRepository) FindByEventID(ctx context.Context, eventID uint, status *models.BookingStatus) ([]models.Booking, error) {
	var bookings []models.Booking
	q := r.db.WithContext(ctx).Where("event_id = ?", eventID)
	if status != nil {
		q = q.Where("status = ?", *status)
	}
	if err := q.Order("id ASC").Find(&bookings).Error; err != nil {
		return nil, err
	}
	return bookings, nil
}

func (r *bookingRepository) FindActiveByUserAndEvent(ctx context.Context, tx *gorm.DB, userID string, eventID uint) (*models.Booking, error) {
	var booking models.Booking
	err := tx.WithContext(ctx).
		Where("user_id = ? AND event_id = ? AND status <> ?", userID, eventID, models.StatusCancelled).
		First(&booking).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

func (r *bookingRepository) CountByStatus(ctx context.Context, tx *gorm.DB, eventID uint, status models.BookingStatus) (int64, error) {
	var count int64
	err := tx.WithContext(ctx).
		Model(&models.Booking{}).
		Where("event_id = ? AND status = ?", eventID, status).
		Count(&count).Error
	return count, err
}

func (r *bookingRepository) UpdateStatus(ctx context.Context, tx *gorm.DB, bookingID uint, status models.BookingStatus) error {
	return tx.WithContext(ctx).
		Model(&models.Booking{}).
		Where("id = ?", bookingID).
		Update("status", status).Error
}

// FindFirstWaitlisted returns the earliest waitlisted booking for promotion.
func (r *bookingRepository) FindFirstWaitlisted(ctx context.Context, tx *gorm.DB, eventID uint) (*models.Booking, error) {
	var booking models.Booking
	err := tx.WithContext(ctx).
		Where("event_id = ? AND status = ?", eventID, models.StatusWaitlisted).
		Order("waitlist_order ASC, id ASC").
		First(&booking).Error
	if err != nil {
		return nil, err
	}
	return &booking, nil
}
