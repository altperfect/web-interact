ifneq (,$(wildcard .env))
include .env
export
endif

COMPOSE ?= docker compose

APP_PORT ?= 18080
APP_SECRET ?= 0123456789abcdef0123456789abcdef
PUBLIC_BASE_URL ?= http://127.0.0.1:$(APP_PORT)
COOKIE_SECURE ?= false
POSTGRES_USER ?= webhook
POSTGRES_PASSWORD ?= webhook
POSTGRES_DB ?= webhook
RETENTION_DAYS ?= 30
MAX_BODY_BYTES ?= 2097152

export APP_PORT APP_SECRET PUBLIC_BASE_URL COOKIE_SECURE
export POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB
export RETENTION_DAYS MAX_BODY_BYTES

.DEFAULT_GOAL := deploy

.PHONY: help build deploy up down restart ps logs test frontend-build

help:
	@printf '%s\n' \
		'Targets:' \
		'  make deploy          Build and run the app container in Docker' \
		'  make build           Build the Docker image' \
		'  make up              Start existing Docker containers' \
		'  make down            Stop and remove Docker containers' \
		'  make restart         Rebuild and restart the app container' \
		'  make ps              Show Docker Compose status' \
		'  make logs            Follow app logs' \
		'  make test            Run Go tests and frontend build'

build:
	$(COMPOSE) build app

deploy:
	$(COMPOSE) up -d --build app

up:
	$(COMPOSE) up -d

down:
	$(COMPOSE) down

restart: deploy

ps:
	$(COMPOSE) ps

logs:
	$(COMPOSE) logs -f app

test:
	GOCACHE=/tmp/w8nc-gocache GOMODCACHE=/tmp/w8nc-gomodcache go test ./...
	$(MAKE) frontend-build

frontend-build:
	npm --prefix frontend run build
