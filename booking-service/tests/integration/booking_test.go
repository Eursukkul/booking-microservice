//go:build integration

package integration

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Eursukkul/booking-microservice/booking-service/internal/models"
	"github.com/Eursukkul/booking-microservice/booking-service/internal/repository"
	"github.com/Eursukkul/booking-microservice/booking-service/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var eventIDCounter uint = 0

func nextEventID() uint {
	eventIDCounter++
	return eventIDCounter
}

func createTestEvent(t *testing.T, name string, maxSeats, waitlist int, price float64) *models.Event {
	t.Helper()
	event := &models.Event{
		ID:             nextEventID(),
		Name:           name,
		MaxSeats:       maxSeats,
		WaitlistLimit:  waitlist,
		Price:          price,
		BookingStartAt: time.Now().Add(-1 * time.Hour),
		BookingEndAt:   time.Now().Add(1 * time.Hour),
	}
	require.NoError(t, testDB.Create(event).Error)
	return event
}

func newBookingService() service.BookingService {
	eventRepo := repository.NewEventRepository(testDB)
	bookingRepo := repository.NewBookingRepository(testDB)
	return service.NewBookingService(bookingRepo, eventRepo)
}

// Test: 60 users book "Golang Workshop Bangkok" concurrently
// → exactly 50 confirmed, 5 waitlisted, 5 rejected
func TestConcurrentBooking(t *testing.T) {
	cleanTables()
	event := createTestEvent(t, "Golang Workshop Bangkok", 50, 5, 2500)
	svc := newBookingService()

	totalUsers := 60
	var wg sync.WaitGroup
	results := make(chan *models.Booking, totalUsers)
	errs := make(chan error, totalUsers)

	wg.Add(totalUsers)
	for i := 0; i < totalUsers; i++ {
		go func(userIdx int) {
			defer wg.Done()
			userID := fmt.Sprintf("user-%03d", userIdx)
			booking, err := svc.CreateBooking(t.Context(), event.ID, userID)
			if err != nil {
				errs <- err
				return
			}
			results <- booking
		}(i)
	}
	wg.Wait()
	close(results)
	close(errs)

	var confirmed, waitlisted int
	for b := range results {
		switch b.Status {
		case models.StatusConfirmed:
			confirmed++
		case models.StatusWaitlisted:
			waitlisted++
		}
	}

	rejectedCount := 0
	for range errs {
		rejectedCount++
	}

	assert.Equal(t, 50, confirmed, "should have exactly 50 confirmed bookings")
	assert.Equal(t, 5, waitlisted, "should have exactly 5 waitlisted bookings")
	assert.Equal(t, 5, rejectedCount, "should reject 5 users (fully booked)")

	// Verify DB counts
	var dbConfirmed, dbWaitlisted int64
	testDB.Model(&models.Booking{}).Where("event_id = ? AND status = ?", event.ID, models.StatusConfirmed).Count(&dbConfirmed)
	testDB.Model(&models.Booking{}).Where("event_id = ? AND status = ?", event.ID, models.StatusWaitlisted).Count(&dbWaitlisted)
	assert.Equal(t, int64(50), dbConfirmed)
	assert.Equal(t, int64(5), dbWaitlisted)
}

// Test: same user books twice → second attempt rejected (double-booking prevention)
func TestDoubleBookingPrevention(t *testing.T) {
	cleanTables()
	event := createTestEvent(t, "Golang Workshop Bangkok", 50, 5, 2500)
	svc := newBookingService()

	booking1, err := svc.CreateBooking(t.Context(), event.ID, "user-duplicate")
	require.NoError(t, err)
	assert.Equal(t, models.StatusConfirmed, booking1.Status)

	booking2, err := svc.CreateBooking(t.Context(), event.ID, "user-duplicate")
	assert.ErrorIs(t, err, service.ErrAlreadyBooked)
	assert.Nil(t, booking2)
}

// Test: same user double-books concurrently → only one succeeds
func TestConcurrentDoubleBooking(t *testing.T) {
	cleanTables()
	event := createTestEvent(t, "Golang Workshop Bangkok", 50, 5, 2500)
	svc := newBookingService()

	attempts := 10
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	wg.Add(attempts)
	for i := 0; i < attempts; i++ {
		go func() {
			defer wg.Done()
			_, err := svc.CreateBooking(t.Context(), event.ID, "user-same")
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, successCount, "only one concurrent booking should succeed for same user")

	var count int64
	testDB.Model(&models.Booking{}).
		Where("event_id = ? AND user_id = ? AND status <> ?", event.ID, "user-same", models.StatusCancelled).
		Count(&count)
	assert.Equal(t, int64(1), count, "DB should have exactly 1 active booking")
}

// Test: cancel confirmed booking → first waitlisted user auto-promoted
func TestCancelAndWaitlistPromotion(t *testing.T) {
	cleanTables()
	event := createTestEvent(t, "Golang Workshop Bangkok", 50, 5, 2500)
	svc := newBookingService()

	// Fill all 50 seats
	var confirmedBookings []*models.Booking
	for i := 0; i < 50; i++ {
		b, err := svc.CreateBooking(t.Context(), event.ID, fmt.Sprintf("user-%03d", i))
		require.NoError(t, err)
		assert.Equal(t, models.StatusConfirmed, b.Status)
		confirmedBookings = append(confirmedBookings, b)
	}

	// Add 3 waitlisted users
	var waitlistedBookings []*models.Booking
	for i := 50; i < 53; i++ {
		b, err := svc.CreateBooking(t.Context(), event.ID, fmt.Sprintf("user-%03d", i))
		require.NoError(t, err)
		assert.Equal(t, models.StatusWaitlisted, b.Status)
		waitlistedBookings = append(waitlistedBookings, b)
	}

	// Cancel the first confirmed booking
	cancelled, err := svc.CancelBooking(t.Context(), confirmedBookings[0].ID)
	require.NoError(t, err)
	assert.Equal(t, models.StatusCancelled, cancelled.Status)

	// First waitlisted (user-050) should be promoted to confirmed
	var promoted models.Booking
	testDB.First(&promoted, waitlistedBookings[0].ID)
	assert.Equal(t, models.StatusConfirmed, promoted.Status, "first waitlisted should be promoted")

	// Second waitlisted still waiting
	var stillWaiting models.Booking
	testDB.First(&stillWaiting, waitlistedBookings[1].ID)
	assert.Equal(t, models.StatusWaitlisted, stillWaiting.Status, "second waitlisted should remain")

	// DB should still have exactly 50 confirmed
	var dbConfirmed int64
	testDB.Model(&models.Booking{}).Where("event_id = ? AND status = ?", event.ID, models.StatusConfirmed).Count(&dbConfirmed)
	assert.Equal(t, int64(50), dbConfirmed, "should still have 50 confirmed after cancel+promote")
}

// Test: booking outside time window → rejected
func TestBookingWindowValidation(t *testing.T) {
	cleanTables()
	svc := newBookingService()

	// Past event
	pastEvent := &models.Event{
		ID:             nextEventID(),
		Name:           "Past Event",
		MaxSeats:       50,
		WaitlistLimit:  5,
		Price:          2500,
		BookingStartAt: time.Now().Add(-48 * time.Hour),
		BookingEndAt:   time.Now().Add(-24 * time.Hour),
	}
	require.NoError(t, testDB.Create(pastEvent).Error)

	_, err := svc.CreateBooking(t.Context(), pastEvent.ID, "user-late")
	assert.ErrorIs(t, err, service.ErrBookingClosed)

	// Future event
	futureEvent := &models.Event{
		ID:             nextEventID(),
		Name:           "Future Event",
		MaxSeats:       50,
		WaitlistLimit:  5,
		Price:          2500,
		BookingStartAt: time.Now().Add(24 * time.Hour),
		BookingEndAt:   time.Now().Add(48 * time.Hour),
	}
	require.NoError(t, testDB.Create(futureEvent).Error)

	_, err = svc.CreateBooking(t.Context(), futureEvent.ID, "user-early")
	assert.ErrorIs(t, err, service.ErrBookingClosed)
}

// Test: booking non-existent event → event not found
func TestBookingEventNotFound(t *testing.T) {
	cleanTables()
	svc := newBookingService()

	_, err := svc.CreateBooking(t.Context(), 99999, "user-1")
	assert.ErrorIs(t, err, service.ErrEventNotFound)
}
