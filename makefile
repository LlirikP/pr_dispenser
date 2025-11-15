APP_NAME=pr_dispenser

.PHONY: run build tidy lint sqlc migrate-up migrate-down

run:
	go run ./cmd/service

build:
	go build -o bin/$(APP_NAME) ./cmd/service

tidy:
	go mod tidy

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...
	
sqlc:
	sqlc generate

migrate-up:
	goose -dir migrations postgres "$(DB_URL)" up
