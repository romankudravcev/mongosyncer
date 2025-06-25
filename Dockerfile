# syntax=docker/dockerfile:1
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o mongosyncer main.go

FROM alpine:3.19
WORKDIR /app
RUN apk add --no-cache curl
COPY --from=builder /app/mongosyncer /app/mongosyncer
RUN chmod +x /app/mongosyncer
ENTRYPOINT ["/app/mongosyncer"]
