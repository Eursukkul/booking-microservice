package main

import (
	"log"

	"github.com/Eursukkul/booking-microservice/booking-service/config"
	"github.com/Eursukkul/booking-microservice/booking-service/internal/consumer"
	"github.com/Eursukkul/booking-microservice/booking-service/internal/handler"
	"github.com/Eursukkul/booking-microservice/booking-service/internal/middleware"
	"github.com/Eursukkul/booking-microservice/booking-service/internal/repository"
	"github.com/Eursukkul/booking-microservice/booking-service/internal/service"
	"github.com/Eursukkul/booking-microservice/booking-service/pkg/database"
	"github.com/Eursukkul/booking-microservice/booking-service/pkg/rabbitmq"
	"github.com/labstack/echo/v4"
	echoMw "github.com/labstack/echo/v4/middleware"
)

func main() {
	cfg := config.Load()

	db := database.NewPostgresDB(cfg.DSN())

	// RabbitMQ consumer: sync events from Event Service
	mqConsumer, err := rabbitmq.NewConsumer(cfg.RabbitURL)
	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %v", err)
	}
	defer mqConsumer.Close()

	msgs, err := mqConsumer.Consume()
	if err != nil {
		log.Fatalf("failed to start consuming: %v", err)
	}

	eventConsumer := consumer.NewEventConsumer(db)
	eventConsumer.Start(msgs)

	// Repositories
	eventRepo := repository.NewEventRepository(db)
	bookingRepo := repository.NewBookingRepository(db)

	// Service
	bookingSvc := service.NewBookingService(bookingRepo, eventRepo)

	// Echo
	e := echo.New()
	e.HTTPErrorHandler = middleware.ErrorHandler
	e.Use(echoMw.RequestLoggerWithConfig(echoMw.RequestLoggerConfig{
		LogStatus: true,
		LogURI:    true,
		LogMethod: true,
		LogValuesFunc: func(c echo.Context, v echoMw.RequestLoggerValues) error {
			log.Printf("%s %s %d", v.Method, v.URI, v.Status)
			return nil
		},
	}))
	e.Use(echoMw.Recover())

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok", "service": "booking-service"})
	})

	handler.NewBookingHandler(bookingSvc, eventRepo, bookingRepo).RegisterRoutes(e)

	log.Printf("Booking Service starting on :%s", cfg.ServerPort)
	e.Logger.Fatal(e.Start(":" + cfg.ServerPort))
}
