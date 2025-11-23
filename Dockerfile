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

# Build the auction service binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /auction ./main.go

FROM gcr.io/distroless/base-debian12:nonroot AS runner

WORKDIR /app

COPY --from=builder /auction /usr/local/bin/auction
COPY --from=builder /src/config ./config

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/auction"]
