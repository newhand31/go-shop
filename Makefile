# Shortcuts for the commands in README. Every target is a plain command you can also run by hand.
SQLC_VERSION := 1.31.1
DATABASE_URL ?= postgres://goshop:goshop@localhost:5432/goshop?sslmode=disable

.PHONY: db-up db-down run sqlc test test-integration

db-up:
	docker compose up -d --wait

db-down:
	docker compose down

run:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/api

# sqlc runs in Docker, so no local C toolchain is needed.
sqlc:
	docker run --rm -v "$(CURDIR):/src" -w /src sqlc/sqlc:$(SQLC_VERSION) generate

test:
	go vet ./...
	go test ./...

test-integration:
	go test -tags=integration ./...
