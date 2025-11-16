.PHONY: help run build test docker-build docker-run compose-up compose-down compose-logs clean migrate

run:
	go run ./cmd/app/main.go

build:
	go build -o bin/app ./cmd/app

test:
	go test -v ./...

lint:
	golangci-lint run

docker-build:
	docker build -t avito-backend-test:local .

up:
	docker-compose up --build

down:
	docker-compose down

logs:
	docker compose logs -f app

clean:
	rm -rf bin/

docker-clean:
	docker-compose down -v

migrate-up:
	migrate -path ./migrations -database "postgres://user:pass@localhost:5432/service?sslmode=disable" up

migrate-down:
	migrate -path ./migrations -database "postgres://user:pass@localhost:5432/service?sslmode=disable" down