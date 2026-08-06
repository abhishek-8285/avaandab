.PHONY: build run test test-race lint fmt vet generate migrate-up migrate-down clean docker dev build-css

## Build CSS from Tailwind source
build-css:
	npx @tailwindcss/cli -i src/input.css -o internal/static/css/tailwind.css --minify

## Build the server binary
build:
	go build -o bin/mvtms ./cmd/server/

## Run the server
run:
	go run ./cmd/server/

## Run tests
test:
	go test -v ./...

## Run tests with race detector
test-race:
	go test -race -v ./...

## Run linter
lint:
	golangci-lint run

## Format code
fmt:
	go fmt ./...

## Vet code
vet:
	go vet ./...

## Generate sqlc code
generate:
	sqlc generate

## Run migrations
migrate-up:
	goose sqlite $(DATABASE_URL) up

migrate-down:
	goose sqlite $(DATABASE_URL) down

## Clean build artifacts
clean:
	rm -rf bin/ coverage.out *.db *.db-wal *.db-shm

## Run tests and build
ci: fmt vet test build

## Run development server
dev:
	air

## Build Docker image
docker:
	docker build -t mvtms:latest .