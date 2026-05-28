FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o raportichka .

FROM alpine:3.20

RUN apk add --no-cache python3 py3-openpyxl

WORKDIR /app

COPY --from=builder /app/raportichka .
COPY --chown=65534:65534 pages ./pages
COPY --chown=65534:65534 static ./static
COPY --chown=65534:65534 vedomost ./vedomost
COPY --chown=65534:65534 config ./config

RUN mkdir -p downloads/schedule downloads/substitutions

USER 65534

EXPOSE 8800

CMD ["./raportichka"]