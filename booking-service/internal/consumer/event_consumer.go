package consumer

import (
	"encoding/json"
	"log"

	"github.com/Eursukkul/booking-microservice/booking-service/internal/models"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EventConsumer struct {
	db *gorm.DB
}

func NewEventConsumer(db *gorm.DB) *EventConsumer {
	return &EventConsumer{db: db}
}

// Start listens for messages and upserts events into the local booking DB.
func (ec *EventConsumer) Start(msgs <-chan amqp.Delivery) {
	go func() {
		for msg := range msgs {
			ec.handleMessage(msg)
		}
		log.Println("[EventConsumer] channel closed, stopping consumer")
	}()
}

func (ec *EventConsumer) handleMessage(msg amqp.Delivery) {
	var event models.Event
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("[EventConsumer] failed to unmarshal: %v", err)
		msg.Nack(false, false)
		return
	}

	// Upsert: insert or update on conflict (same ID from Event Service)
	result := ec.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "max_seats", "waitlist_limit", "price", "booking_start_at", "booking_end_at", "updated_at"}),
	}).Create(&event)

	if result.Error != nil {
		log.Printf("[EventConsumer] failed to upsert event %d: %v", event.ID, result.Error)
		msg.Nack(false, true) // requeue
		return
	}

	log.Printf("[EventConsumer] synced event %d: %s", event.ID, event.Name)
	msg.Ack(false)
}
