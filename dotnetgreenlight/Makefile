# Settings live in src/Greenlight.Api/appsettings{,.Development,.Production}.json.
# The launch profiles (Properties/launchSettings.json) run in Development and only override
# the connection string via the ConnectionStrings__Greenlight environment variable.
PROJECT     := src/Greenlight.Api
PUBLISH_DIR := ./bin/publish
# Launch profiles set applicationUrl; the published app gets the same URL via ASPNETCORE_URLS.
PUBLISHED_URL ?= http://localhost:5133

## help: print this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run/homelab: run the API (Development) against the homelab PostgreSQL ("homelab" profile)
run/homelab:
	dotnet run --project $(PROJECT) --launch-profile homelab

## run/locally: run the API (Development) against the local PostgreSQL ("local" profile)
run/locally:
	dotnet run --project $(PROJECT) --launch-profile local

## otel/up: start Grafana LGTM - OTLP ports 4317/4318, Grafana UI on port 3000
otel/up:
	docker compose up -d

## otel/down: stop Grafana LGTM
otel/down:
	docker compose down

## db/up: apply the shared Goose migrations (same schema and GOOSE_* settings as the Go app)
db/up:
	cd .. && goose up

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: restore packages and format code
tidy:
	dotnet restore
	dotnet format

## audit: verify formatting, build with warnings as errors, run all tests
audit:
	dotnet format --verify-no-changes
	dotnet build --no-incremental
	dotnet test --solution Greenlight.slnx --no-build

## test/unit: run unit tests (no Docker required)
test/unit:
	dotnet test --project tests/Greenlight.Api.UnitTests

## test/integration: run integration tests (requires Docker)
test/integration:
	dotnet test --project tests/Greenlight.Api.IntegrationTests

## test/all: run unit and integration tests (requires Docker)
test/all:
	dotnet test --solution Greenlight.slnx

# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build/api: build the API in Release configuration
build/api:
	dotnet build $(PROJECT) -c Release

## publish: publish the API (Release, framework-dependent) to ./bin/publish
publish:
	dotnet publish $(PROJECT) -c Release -o $(PUBLISH_DIR)

## run/published: run the published API in Production (appsettings.Production.json)
run/published:
	cd $(PUBLISH_DIR) && ASPNETCORE_ENVIRONMENT=Production ASPNETCORE_URLS=$(PUBLISHED_URL) dotnet Greenlight.Api.dll

.PHONY: help run/homelab run/locally otel/up otel/down db/up tidy audit test/unit test/integration test/all build/api publish run/published
