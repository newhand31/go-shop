# Multi-stage build: the toolchain stays in the first stage, only the binary ships.
# Build (from the repo root):  docker build -t go-shop .

ARG GO_VERSION=1.27.0

# ---- build stage: Go toolchain, source, module cache. None of this reaches the final image.
FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src

# go.mod and go.sum first: this layer is rebuilt only when dependencies change, not on every source edit.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0: a fully static binary with no libc, which is the only kind that runs on scratch.
# -trimpath removes local build paths from the binary; -s -w drop the symbol and debug tables.
# Migrations are embedded (db/migrations.go), so the binary needs no files next to it.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---- final stage: literally nothing but the binary.
FROM scratch

# TLS roots, so the binary can call HTTPS APIs later. scratch has no certificate store of its own.
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/api /api

# scratch has no /etc/passwd, so the user is a bare number: not root, and nothing to look up.
USER 65532:65532
EXPOSE 8080

# No HEALTHCHECK on purpose: scratch has no shell and no curl to run one with. Let the platform probe GET /healthz.
# Exec form (a JSON array) makes /api PID 1, so docker stop's SIGTERM reaches the Go program. The shell form
# ("ENTRYPOINT /api") would start /bin/sh as PID 1, which scratch does not have, and the signal would go nowhere useful.
ENTRYPOINT ["/api"]
