FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/api ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache tzdata
WORKDIR /app
COPY --from=build /app/bin/api /app/api
EXPOSE 8080
ENV TZ=Africa/Kinshasa
CMD ["/app/api"]
