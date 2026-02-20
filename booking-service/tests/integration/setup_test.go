//go:build integration

package integration

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/Eursukkul/booking-microservice/booking-service/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv("TEST_DB_HOST", "localhost"),
		getEnv("TEST_DB_PORT", "5434"),
		getEnv("TEST_DB_USER", "postgres"),
		getEnv("TEST_DB_PASSWORD", "postgres"),
		getEnv("TEST_DB_NAME", "booking_test_db"),
	)

	var err error
	testDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("failed to connect to test database: %v", err)
	}

	// Drop and recreate tables for clean state
	testDB.Exec("DROP TABLE IF EXISTS bookings")
	testDB.Exec("DROP TABLE IF EXISTS events")

	if err := testDB.AutoMigrate(&models.Event{}, &models.Booking{}); err != nil {
		log.Fatalf("failed to auto-migrate test database: %v", err)
	}

	testDB.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_booking_active
		ON bookings (event_id, user_id)
		WHERE status <> 'cancelled'
	`)

	code := m.Run()

	testDB.Exec("DROP TABLE IF EXISTS bookings")
	testDB.Exec("DROP TABLE IF EXISTS events")

	os.Exit(code)
}

func cleanTables() {
	testDB.Exec("DELETE FROM bookings")
	testDB.Exec("DELETE FROM events")
	testDB.Exec("ALTER SEQUENCE IF EXISTS events_id_seq RESTART WITH 1")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
