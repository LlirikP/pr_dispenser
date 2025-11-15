FROM golang:1.22 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/pressly/goose/cmd/goose@v2.7.0

COPY . .

RUN go build -o serv ./cmd/service

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=builder /app/serv .
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY internal/sql /app/internal/sql

ENV PORT=8080

EXPOSE 8080

CMD ["./serv"]
