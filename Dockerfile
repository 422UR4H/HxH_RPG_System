# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

# Slim, postgres-only goose CLI pinned to the version in go.mod.
RUN go install -tags='no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb no_turso no_starrocks' \
    github.com/pressly/goose/v3/cmd/goose@v3.27.1

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/game ./cmd/game

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -u 10001 app
WORKDIR /app

COPY --from=builder /out/api /out/game ./
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY migrations ./migrations
COPY scripts/start-api.sh ./start-api.sh
RUN chmod +x ./start-api.sh

USER app
# Default: REST API (applies migrations first). The game service overrides this with ./game.
CMD ["./start-api.sh"]
