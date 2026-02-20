.PHONY: docker-up docker-down docker-up-all run-event run-booking test test-integration

# Infrastructure only (DBs + RabbitMQ)
docker-up:
	docker-compose up -d event-db booking-db booking-test-db rabbitmq

docker-down:
	docker-compose down -v

# Infrastructure + both services
docker-up-all:
	docker-compose up -d --build

# Run services locally (requires docker-up for DBs + RabbitMQ)
run-event:
	cd event-service && go run main.go

run-booking:
	cd booking-service && go run main.go

# Unit tests
test-event:
	cd event-service && go test ./internal/... -v -count=1

test-booking:
	cd booking-service && go test ./internal/... -v -count=1

test:
	$(MAKE) test-event
	$(MAKE) test-booking

# Integration tests
test-integration:
	cd booking-service && go test ./tests/integration/... -v -count=1 -tags=integration
