.PHONY: build run test test-integration docker-up docker-down docker-logs clean

BIN_NAME=server

build:
	@echo "Building mini-crm server..."
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN_NAME) ./cmd/server

run: build
	@echo "Running mini-crm server locally..."
	./$(BIN_NAME)

test:
	@echo "Running unit tests..."
	go test -v -race ./internal/...

test-integration:
	@echo "Running SQL report integration test..."
	go test -v -count=1 ./tests/integration/...

docker-up:
	@echo "Starting MySQL, Redis, and Mini-CRM app with Docker Compose..."
	docker compose up --build -d

docker-down:
	@echo "Stopping Docker Compose services..."
	docker compose down

docker-logs:
	docker compose logs -f

clean:
	@echo "Cleaning up build artifacts..."
	rm -f $(BIN_NAME)
