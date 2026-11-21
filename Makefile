## help: print this help message
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'
## run/homelab: will run api that connects to homelab server database
run/homelab:
	go run ./cmd/api --port=4000 --env=development --db-dsn=postgres://postgres:postgres@192.168.1.60/greenlight?sslmode=disable --db-max-open-conns=50 --db-max-idle-time=2h30m --limiter-requests=2 --limiter-window=5s --limiter-enabled=false --auth-enabled=true '--cors-trusted-origins=http://localhost:9000 http://localhost:9001' 

## run/locally: will run api that connects to default(localhost) database
run/locally:
	go run ./cmd/api --port=4000 --env=development --db-dsn=postgres://postgres:a-secure-password@localhost/greenlight?sslmode=disable --db-max-open-conns=50 --db-max-idle-time=2h30m --limiter-requests=2 --limiter-window=5s --limiter-enabled=false --auth-enabled=true '--cors-trusted-origins=http://localhost:9000 http://localhost:9001'

## db/up: migrate database which is hosted on GOOSE_DBSTRING in .env 
db/up:
	goose up

## db/migrations/new name=$1: creates a new database migration
db/migrations/new:
	goose create ${name} sql
