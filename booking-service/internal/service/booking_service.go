package service

import (
	"context"
	"errors"
	"time"

	"github.com/Eursukkul/booking-microservice/booking-service/internal/models"
	"github.com/Eursukkul/booking-microservice/booking-service/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrEventNotFound    = errors.New("event not found")
	ErrBookingNotFound  = errors.New("booking not found")
	ErrBookingClosed    = errors.New("booking is not open")
	ErrAlreadyBooked    = errors.New("user already has an active booking for this event")
	ErrEventFullyBooked = errors.New("event is fully booked (seats + waitlist)")
)

type BookingService interface {
	CreateBooking(ctx context.Context, eventID uint, userID string) (*models.Booking, error)
	CancelBooking(ctx context.Context, bookingID uint) (*models.Booking, error)
	GetBooking(ctx context.Context, id uint) (*models.Booking, error)
	ListBookings(ctx context.Context, eventID uint, status *models.BookingStatus) ([]models.Booking, error)
}

type bookingService struct {
	bookingRepo repository.BookingRepository
	eventRepo   repository.EventRepository
}

func NewBookingService(bookingRepo repository.BookingRepository, eventRepo repository.EventRepository) BookingService {
	return &bookingService{
		bookingRepo: bookingRepo,
		eventRepo:   eventRepo,
	}
}

func (s *bookingService) CreateBooking(ctx context.Context, eventID uint, userID string) (*models.Booking, error) {
	var result *models.Booking

	err := s.bookingRepo.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Lock the event row — serializes concurrent booking attempts
		event, err := s.eventRepo.FindByIDForUpdate(ctx, tx, eventID)
		if err != nil {
			return ErrEventNotFound
		}

		// 2. Check booking window
		now := time.Now()
		if now.Before(event.BookingStartAt) || now.After(event.BookingEndAt) {
			return ErrBookingClosed
		}

		// 3. Check double-booking
		_, err = s.bookingRepo.FindActiveByUserAndEvent(ctx, tx, userID, eventID)
		if err == nil {
			return ErrAlreadyBooked
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// 4. Count confirmed seats
		confirmedCount, err := s.bookingRepo.CountByStatus(ctx, tx, eventID, models.StatusConfirmed)
		if err != nil {
			return err
		}

		// 5. Determine status
		if int(confirmedCount) < event.MaxSeats {
			// Seat available → confirmed
			booking := &models.Booking{
				EventID: eventID,
				UserID:  userID,
				Status:  models.StatusConfirmed,
			}
			if err := s.bookingRepo.Create(ctx, tx, booking); err != nil {
				return err
			}
			result = booking
			return nil
		}

		// 6. Seats full → try waitlist
		waitlistedCount, err := s.bookingRepo.CountByStatus(ctx, tx, eventID, models.StatusWaitlisted)
		if err != nil {
			return err
		}

		if int(waitlistedCount) < event.WaitlistLimit {
			order := int(waitlistedCount) + 1
			booking := &models.Booking{
				EventID:       eventID,
				UserID:        userID,
				Status:        models.StatusWaitlisted,
				WaitlistOrder: &order,
			}
			if err := s.bookingRepo.Create(ctx, tx, booking); err != nil {
				return err
			}
			result = booking
			return nil
		}

		// 7. Everything full
		return ErrEventFullyBooked
	})

	return result, err
}

func (s *bookingService) CancelBooking(ctx context.Context, bookingID uint) (*models.Booking, error) {
	var result *models.Booking

	err := s.bookingRepo.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Find the booking
		booking, err := s.bookingRepo.FindByID(ctx, bookingID)
		if err != nil {
			return ErrBookingNotFound
		}

		if booking.Status == models.StatusCancelled {
			return errors.New("booking is already cancelled")
		}

		wasPreviouslyConfirmed := booking.Status == models.StatusConfirmed

		// Lock the event row to safely promote waitlisted users
		_, err = s.eventRepo.FindByIDForUpdate(ctx, tx, booking.EventID)
		if err != nil {
			return err
		}

		// Cancel the booking
		if err := s.bookingRepo.UpdateStatus(ctx, tx, bookingID, models.StatusCancelled); err != nil {
			return err
		}

		booking.Status = models.StatusCancelled
		result = booking

		// If a confirmed booking was cancelled, promote first waitlisted
		if wasPreviouslyConfirmed {
			waitlisted, err := s.bookingRepo.FindFirstWaitlisted(ctx, tx, booking.EventID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil // no one to promote
				}
				return err
			}
			if err := s.bookingRepo.UpdateStatus(ctx, tx, waitlisted.ID, models.StatusConfirmed); err != nil {
				return err
			}
		}

		return nil
	})

	return result, err
}

func (s *bookingService) GetBooking(ctx context.Context, id uint) (*models.Booking, error) {
	return s.bookingRepo.FindByID(ctx, id)
}

func (s *bookingService) ListBookings(ctx context.Context, eventID uint, status *models.BookingStatus) ([]models.Booking, error) {
	return s.bookingRepo.FindByEventID(ctx, eventID, status)
}
