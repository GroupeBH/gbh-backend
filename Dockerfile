# Multi-stage Dockerfile for development and production

# Base stage for dependencies
FROM golang:1.22-alpine AS base
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

# Development stage with hot reload
FROM base AS dev
RUN apk add --no-cache git
RUN go install github.com/cosmtrek/air@latest
COPY . .
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

# Production build stage
FROM base AS build
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/bin/api ./cmd/api

# Production runtime stage
FROM alpine:3.20 AS prod
RUN apk add --no-cache tzdata
WORKDIR /app
COPY --from=build /app/bin/api /app/api
EXPOSE 8080
ENV TZ=Africa/Kinshasa
CMD ["/app/api"]
