package main

import (
	"log"

	"github.com/Eursukkul/booking-microservice/event-service/config"
	"github.com/Eursukkul/booking-microservice/event-service/internal/handler"
	"github.com/Eursukkul/booking-microservice/event-service/internal/middleware"
	"github.com/Eursukkul/booking-microservice/event-service/internal/repository"
	"github.com/Eursukkul/booking-microservice/event-service/internal/service"
	"github.com/Eursukkul/booking-microservice/event-service/pkg/database"
	"github.com/Eursukkul/booking-microservice/event-service/pkg/rabbitmq"
	"github.com/labstack/echo/v4"
	echoMw "github.com/labstack/echo/v4/middleware"
)

func main() {
	cfg := config.Load()

	db := database.NewPostgresDB(cfg.DSN())

	publisher, err := rabbitmq.NewPublisher(cfg.RabbitURL)
	if err != nil {
		log.Fatalf("failed to connect to RabbitMQ: %v", err)
	}
	defer publisher.Close()

	repo := repository.NewEventRepository(db)
	svc := service.NewEventService(repo, publisher)

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
		return c.JSON(200, map[string]string{"status": "ok", "service": "event-service"})
	})

	api := e.Group("/api/v1/events")
	handler.NewEventHandler(svc).RegisterRoutes(api)

	log.Printf("Event Service starting on :%s", cfg.ServerPort)
	e.Logger.Fatal(e.Start(":" + cfg.ServerPort))
}
