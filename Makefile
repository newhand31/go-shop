# Shortcuts for the commands in README. Every target is a plain command you can also run by hand.
SQLC_VERSION := 1.31.1
DATABASE_URL ?= postgres://goshop:goshop@localhost:5432/goshop?sslmode=disable
REDIS_URL ?= redis://localhost:6379/0

OAPI_CODEGEN_VERSION := v2.8.2

.PHONY: db-up db-down run sqlc api test test-integration

db-up:
	docker compose up -d --wait

db-down:
	docker compose down

run:
	DATABASE_URL="$(DATABASE_URL)" REDIS_URL="$(REDIS_URL)" go run ./cmd/api

# sqlc runs in Docker, so no local C toolchain is needed.
sqlc:
	docker run --rm -v "$(CURDIR):/src" -w /src sqlc/sqlc:$(SQLC_VERSION) generate

# Contract-first: edit api/openapi.yaml, then regenerate internal/api/api.gen.go.
api:
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION) -config oapi-codegen.yaml api/openapi.yaml

test:
	go vet ./...
	go test ./...

test-integration:
	go test -tags=integration ./...
