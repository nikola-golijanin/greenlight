# Include variables from .env
include .env

## help: print this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run/homelab: will run api that connects to homelab server database
run/homelab:
	go run ./cmd/api --port=4000 --env=development --db-dsn=${GREENLIGHT_DB_DSN_HOMELAB} --db-max-open-conns=50 --db-max-idle-time=2h30m --limiter-requests=2 --limiter-window=5s --limiter-enabled=false --auth-enabled=true '--cors-trusted-origins=http://localhost:9000 http://localhost:9001' 

## run/locally: will run api that connects to default(localhost) database
run/locally:
	go run ./cmd/api --port=4000 --env=development --db-dsn=${GREENLIGHT_DB_DSN_LOCAL} --db-max-open-conns=50 --db-max-idle-time=2h30m --limiter-requests=2 --limiter-window=5s --limiter-enabled=false --auth-enabled=true '--cors-trusted-origins=http://localhost:9000 http://localhost:9001'

## db/up: migrate database which is hosted on GOOSE_DBSTRING in .env 
db/up:
	goose up

## db/migrations/new name=$1: creates a new database migration
db/migrations/new:
	goose create ${name} sql

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: tidy module dependencies, and format and modernize all .go files
tidy:
	go mod tidy
	go mod verify
# 	go mod vendor
	go fix ./...
	go fmt ./...

## audit: run quality control checks
audit:
	go mod tidy -diff
	go mod verify
	go vet ./...
	go tool staticcheck ./...
	go test -race -vet=off ./...

## test/unit: run unit tests (no Docker required)
test/unit:
	go test -race ./internal/...

## test/integration: run integration tests (requires Docker)
test/integration:
	go test -race -count=1 ./cmd/api/...

## test/all: run unit and integration tests (requires Docker)
test/all:
	go test -race -count=1 ./...

# ==================================================================================== #
# BUILD
# ==================================================================================== #
## build/api: build the cmd/api application
build/api:
	go build -ldflags='-s' -o=./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/api ./cmd/api
