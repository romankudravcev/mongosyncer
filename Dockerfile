# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mongosyncer main.go

FROM debian:bookworm-slim
WORKDIR /app
RUN apt-get update && apt-get install -y curl tar && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/mongosyncer /app/mongosyncer
RUN chmod +x /app/mongosyncer
ENTRYPOINT ["/app/mongosyncer"]
