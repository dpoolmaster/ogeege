COMPOSE = docker compose --project-directory . -f docker/docker-compose.yml

.PHONY: up upd down run migrate-up migrate-down test lint build templ

up:
	$(COMPOSE) up

upd:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

run:
	go run ./cmd/server

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down 1

test:
	go test ./...

lint:
	golangci-lint run ./...

build:
	go build ./...

templ:
	@if find . -name '*.templ' -print -quit | grep -q .; then \
		templ generate; \
	else \
		echo "Нет .templ-файлов — генерация пропущена"; \
	fi
