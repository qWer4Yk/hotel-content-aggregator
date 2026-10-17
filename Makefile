.PHONY: all build run test test-race docker-up docker-down clean

all: test-race build

build:
	go build -o bin/hotel-server ./cmd/server/main.go

run:
	go run ./cmd/server/main.go

test:
	go test -v ./...

test-race:
	go test -v -race ./...

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v

clean:
	rm -rf bin/
