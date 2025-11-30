FROM golang:1.25.4 AS dev

WORKDIR /app

RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

CMD ["air", "-c", ".air.toml"]

FROM golang:1.25.4 AS builder

WORKDIR /src

# Download dependencies first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build the API service binary
FROM builder AS builder-api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /auction-api ./cmd/api/main.go

# Build the Worker service binary
FROM builder AS builder-worker
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /auction-worker ./cmd/worker/main.go

# API Service Runner
FROM gcr.io/distroless/base-debian12:nonroot AS api

WORKDIR /app

COPY --from=builder-api /auction-api /usr/local/bin/auction-api

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/auction-api"]

# Worker Service Runner
FROM gcr.io/distroless/base-debian12:nonroot AS worker

WORKDIR /app

COPY --from=builder-worker /auction-worker /usr/local/bin/auction-worker

ENTRYPOINT ["/usr/local/bin/auction-worker"]
