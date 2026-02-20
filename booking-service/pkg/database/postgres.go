package database

import (
	"log"

	"github.com/Eursukkul/booking-microservice/booking-service/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPostgresDB(dsn string) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(&models.Event{}, &models.Booking{}); err != nil {
		log.Fatalf("failed to auto-migrate: %v", err)
	}

	// Partial unique index: prevents double-booking (same user + same event) unless cancelled
	db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_booking_active
		ON bookings (event_id, user_id)
		WHERE status <> 'cancelled'
	`)

	return db
}
