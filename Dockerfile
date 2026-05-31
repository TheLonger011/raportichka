FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o raportichka ./cmd/server

FROM alpine:3.20 AS migrate-cli
RUN apk add --no-cache curl
ARG MIGRATE_VERSION=v4.17.0
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-amd64.tar.gz | tar xvz && \
    mv migrate /usr/local/bin/migrate && \
    chmod +x /usr/local/bin/migrate

FROM alpine:3.20

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    python3 \
    py3-pip \
    && pip3 install --no-cache-dir openpyxl \
    && apk del py3-pip

COPY --from=migrate-cli /usr/local/bin/migrate /usr/local/bin/migrate

WORKDIR /app

COPY --from=builder /app/raportichka .

COPY --chown=65534:65534 pages ./pages
COPY --chown=65534:65534 static ./static
COPY --chown=65534:65534 vedomost ./vedomost
COPY --chown=65534:65534 config ./config
COPY --chown=65534:65534 migrations ./migrations

RUN mkdir -p downloads/schedule downloads/substitutions && \
    chown -R 65534:65534 downloads

USER 65534

EXPOSE 8800

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8800/api/groups-public || exit 1

CMD migrate -path /app/migrations -database "$STORAGE_PATH" up && ./raportichka