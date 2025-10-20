# syntax=docker/dockerfile:1.6
FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o mesa-ya ./cmd/server

FROM gcr.io/distroless/base-debian12
WORKDIR /srv
COPY --from=build /app/mesa-ya .
COPY .env .env
EXPOSE 8080
ENTRYPOINT ["/srv/mesa-ya"]