APP_NAME := adaptive-routing-backend
COMPOSE_FILE := deployments/local/docker-compose.yml

.PHONY: build run dry-run test race vet lint fmt up down

build:
	go build -o bin/api ./cmd/bin

run:
	go run ./cmd/bin -config env.yaml

dry-run:
	go run ./cmd/bin -config env.yaml -dry-run

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

lint:
	docker run --rm -v "$(CURDIR):/app" -w /app golangci/golangci-lint:v2.1.6 golangci-lint run

fmt:
	gofmt -w cmd configs delivery internal pkg

up:
	docker compose -f $(COMPOSE_FILE) up --build

down:
	docker compose -f $(COMPOSE_FILE) down
