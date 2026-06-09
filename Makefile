.PHONY: build run templ docker-build docker-up docker-down

## Generate templ files
templ:
	templ generate

## Build the binary locally (runs templ generate first)
build: templ
	go build -o bin/server .

## Run locally (builds first)
run: build
	./bin/server

## Build the Docker image
docker-build:
	docker build -t esp32-rgb .

## Start with docker compose (builds image if needed)
docker-up:
	docker compose up --build

## Stop docker compose services
docker-down:
	docker compose down
