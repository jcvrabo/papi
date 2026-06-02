.PHONY: build test lint migrate-up migrate-down generate run docker-build

BINARY=papi
BUILD_DIR=bin

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/papi

test:
	go test ./...

lint:
	golangci-lint run ./...

migrate-up:
	migrate -path internal/store/migrations -database "$$DATABASE_URL" up

migrate-down:
	migrate -path internal/store/migrations -database "$$DATABASE_URL" down

generate:
	oapi-codegen --config api/openapi/cfg.yaml api/openapi/v1.yaml

run:
	go run ./cmd/papi

docker-build:
	docker build -t papi .
