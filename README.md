# Auction Service

Event-driven microservice for managing auction items with RabbitMQ integration and high-throughput concurrent processing.

## Architecture Overview

The auction service consists of two main components:

### 1. API Service (`cmd/api/`)
**Role:** HTTP REST API + Event Publisher

- Handles item CRUD operations
- Publishes item events (created, updated, deleted) to RabbitMQ
- Used by external clients and services

**Events Published:**
- `item.created.v1` → When a new item is created
- `item.updated.v1` → When an item is updated
- `item.deleted.v1` → When an item is deleted

### 2. Worker Service (`cmd/worker/`)
**Role:** Event Consumer + Item State Manager

- Consumes events from other services (bid service, payment service, etc.)
- Updates item state based on external events
- **Runs with 3 replicas** (horizontal scaling)
- **60 concurrent workers** (3 replicas × 20 workers each)
- **45 total DB connections** (3 replicas × 15 connections each)

**Events Consumed:**
- `bid.placed.v1` → Updates item current price
- `bid.won.v1` → Marks item as sold and sets buyer

## Project Structure

```
auction/
├── cmd/
│   ├── api/                    # HTTP API Service (Publisher)
│   │   └── main.go
│   └── worker/                 # Event Worker Service (Consumer)
│       └── main.go
│
├── app/                        # Application layer
│   └── item/                   # Item HTTP handlers
│
├── domain/                     # Domain entities
│   └── item.go
│
├── internal/
│   ├── middleware/             # HTTP middlewares
│   └── consumers/              # Event consumer handlers
│       └── bid_consumer.go     # Handles bid events
│
├── infra/
│   ├── postgres/               # Database layer
│   │   ├── migrations/
│   │   └── repository.go       # Connection pool tuning
│   └── rabbitmq/               # Message broker
│       ├── publisher.go        # Event publishing
│       └── consumer.go         # Concurrent event consuming
│
├── pkg/
│   ├── config/                 # Configuration management
│   ├── events/                 # Event schemas
│   └── httperror/              # HTTP error handling
│
├── docker-compose.yaml         # Multi-service setup with replicas
└── Dockerfile                  # Multi-stage build (api + worker)
```

## Event Flow Diagram

```
┌───────────────────────────────────────────────────────────┐
│                    Auction Service                        │
├───────────────────────────────────────────────────────────┤
│                                                           │
│  ┌─────────────┐                  ┌─────────────┐         │
│  │ API Service │                  │   Worker    │         │
│  │ (Publisher) │                  │  (Consumer) │         │
│  │             │                  │  x3 replicas│         │
│  │             │                  │  60 workers │         │
│  └──────┬──────┘                  └──────▲──────┘         │
│         │                                │                │
│         │ Publishes item.*               │ Consumes       │
│         │                                │ bid.*          │
└─────────┼────────────────────────────────┼────────────────┘
          │                                │
          ▼                                │
    ┌─────────────┐                        │
    │  RabbitMQ   │                        │
    ├─────────────┤                        │
    │ Exchange:   │                        │
    │ auction.item│──────┐                 │
    │             │      │                 │
    │ Exchange:   │      │                 │
    │ auction.bid │──────┼─────────────────┘
    └─────────────┘      │
                         │
         ┌───────────────┼────────────────┐
         │               │                │
         ▼               ▼                ▼
    ┌─────────┐   ┌─────────────┐   ┌──────────┐
    │ Payment │   │Notification │   │PostgreSQL│
    │ Service │   │  Service    │   │   DB     │
    └─────────┘   └─────────────┘   └──────────┘
                                     45/100 conns
```

## Performance & Scalability

### Worker Concurrency Configuration

The worker service is optimized for high-throughput event processing:

```go
// Per replica configuration (cmd/worker/main.go)
ConsumerConfig{
    PrefetchCount:  10,  // Prefetch 10 messages from RabbitMQ
    WorkerPoolSize: 20,  // Process up to 20 messages concurrently
}

// Docker deployment (docker-compose.yaml)
deploy:
    replicas: 3          // 3 worker instances

// Total capacity
// - 60 concurrent workers (3 × 20)
// - 30 messages prefetched (3 × 10)
```

### Database Connection Pool

Tuned for concurrent access (infra/postgres/repository.go):

```go
db.SetMaxOpenConns(15)                    // Max 15 connections per replica
db.SetMaxIdleConns(8)                     // Keep 8 idle in pool
db.SetConnMaxLifetime(5 * time.Minute)    // Recycle every 5 min
db.SetConnMaxIdleTime(2 * time.Minute)    // Close idle after 2 min

// Total DB connections: 3 replicas × 15 = 45 connections
// Safe margin below PostgreSQL default max_connections=100
```

### Monitoring

Connection pool stats are logged every 30 seconds:

```
INFO  Connection pool stats
  max_open=15
  open=8
  in_use=5
  idle=3
  wait_count=0        ← Should stay 0 (workers never block)
  wait_duration_ms=0
```

**Alert if:**
- `wait_count > 0` → Workers are waiting for DB connections, increase MaxOpenConns
- `in_use / max_open > 0.8` → Pool is >80% full, potential bottleneck

## Running the Service

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- PostgreSQL
- RabbitMQ

### Development Mode

```bash
# Start all services (API + 3 Worker replicas + Postgres)
docker-compose --profile dev up

# Or start individually
docker-compose --profile dev up auction-api-dev
docker-compose --profile dev up auction-worker-dev

# Check running workers
docker-compose ps | grep worker
```

### Production Mode

```bash
# Build and start all services
docker-compose up -d

# Services:
# - auction-api: http://localhost:8081
# - auction-worker: 3 replicas (background workers)
# - auction-postgres: localhost:5433

# View worker logs
docker-compose logs -f auction-worker

# Monitor connection pool
docker-compose logs -f auction-worker | grep "pool stats"
```

### Local Development (Without Docker)

#### 1. Start dependencies
```bash
docker-compose up auction-postgres
```

#### 2. Run API service
```bash
POSTGRES_HOST=localhost \
POSTGRES_PORT=5433 \
POSTGRES_DATABASE=auction \
POSTGRES_USERNAME=postgres \
POSTGRES_PASSWORD=postgres \
RABBITMQ_URL=amqp://guest:guest@localhost:5672/ \
SERVICE_NAME=auction \
PORT=8080 \
go run cmd/api/main.go
```

#### 3. Run Worker service
```bash
POSTGRES_HOST=localhost \
POSTGRES_PORT=5433 \
POSTGRES_DATABASE=auction \
POSTGRES_USERNAME=postgres \
POSTGRES_PASSWORD=postgres \
RABBITMQ_URL=amqp://guest:guest@localhost:5672/ \
SERVICE_NAME=auction \
go run cmd/worker/main.go
```

## API Endpoints

### Public Endpoints
- `GET /api/v1/items` - List all items
- `GET /api/v1/items/:id` - Get item by ID

### Private Endpoints (Require X-User-ID header)
- `POST /api/v1/items` - Create new item
- `PUT /api/v1/items/:id` - Update item
- `DELETE /api/v1/items/:id` - Delete item

### Example: Create Item

```bash
curl -X POST http://localhost:8081/api/v1/items \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user-123" \
  -d '{
    "name": "Vintage Camera",
    "description": "Rare 1960s camera",
    "currencyCode": "USD",
    "startPrice": "100.00",
    "bidIncrement": "5.00",
    "startDate": "2024-01-15T10:00:00Z",
    "endDate": "2024-01-20T10:00:00Z",
    "status": "active"
  }'
```

## Event Integration

### Publishing Events (API Service)

When items are created/updated/deleted, the API service automatically publishes events:

```json
// Publishes to: auction.item exchange
// Routing key: item.created.v1
{
  "event": "item.created",
  "version": "v1",
  "timestamp": "2024-01-15T10:00:00Z",
  "traceId": "trace-uuid",
  "correlationId": "correlation-uuid",
  "payload": {
    "id": "item-uuid",
    "name": "Vintage Camera",
    "sellerId": "user-123"
  }
}
```

### Consuming Events (Worker Service)

The worker service listens for events from other services:

```json
// Consumes from: auction.bid exchange
// Routing keys: bid.*.v1 (all bid events)

// Example: bid.placed.v1
{
  "event": "bid.placed",
  "version": "v1",
  "payload": {
    "bidId": "bid-uuid",
    "itemId": "item-uuid",
    "userId": "user-uuid",
    "amount": "150.00",
    "currencyCode": "USD"
  }
}
```

## Database Migrations

Migrations are automatically applied on startup via Docker `docker-entrypoint-initdb.d`.

Location: `infra/postgres/migrations/`

Current migrations:
- `001_create_items.sql` - Creates items table with triggers and indexes

## Configuration

Environment variables (`.env`):

```env
# Server
PORT=8080
HOST_PORT=8081

# Database
POSTGRES_HOST=auction-postgres
POSTGRES_PORT=5432
POSTGRES_DATABASE=auction
POSTGRES_USERNAME=postgres
POSTGRES_PASSWORD=postgres

# Message Broker
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
SERVICE_NAME=auction
```

## Monitoring

### RabbitMQ Management UI
- URL: http://localhost:15672
- Credentials: guest/guest

### Health Checks
- Postgres: `docker-compose ps` (should show healthy)
- API: `curl http://localhost:8081/api/v1/items`
- Worker: Check logs for "Worker service started successfully"

### Connection Pool Monitoring
```bash
# Watch pool stats in real-time
docker-compose logs -f auction-worker | grep "pool stats"

# Check PostgreSQL active connections
docker exec -it auction-auction-auction-postgres-1 \
  psql -U postgres -d auction \
  -c "SELECT count(*) FROM pg_stat_activity WHERE datname='auction';"
```

## Production Considerations

### Scalability
- **API Service**: Scale horizontally (multiple instances behind load balancer)
- **Worker Service**: Currently 3 replicas with 20 workers each = 60 concurrent processors
  - Scale up: Increase `WorkerPoolSize` (more workers per replica)
  - Scale out: Increase `replicas` (more instances)
- **Database**: Connection pool tuned for 45 concurrent connections
  - If scaling workers, adjust `MaxOpenConns` proportionally
- **RabbitMQ**: Cluster for high availability

### Reliability
- **Dead Letter Queues**: Automatically created for failed event processing
- **Idempotency**: Event handlers should be idempotent
- **Graceful Shutdown**: Both services handle SIGTERM properly
- **Message Acknowledgment**: Manual ACK after successful processing
- **Retry Logic**: RabbitMQ handles redelivery of unacknowledged messages

### Observability
- **Tracing**: All events include `traceId` and `correlationId`
- **Logging**: Structured logging with Zap
- **Metrics**: Connection pool stats, consumer lag monitoring
- **Health**: Connection pool `wait_count` indicates bottlenecks

### Concurrency Pattern

The worker uses a **semaphore pattern** for controlled concurrency:

```go
// infra/rabbitmq/consumer.go
semaphore := make(chan struct{}, workerPoolSize)  // Buffer = pool size

// Acquire slot (blocks if pool full)
semaphore <- struct{}{}

// Process in goroutine
go func(msg amqp.Delivery) {
    defer func() { <-semaphore }()  // Release slot when done
    processMessage(msg)
}(msg)
```

Benefits:
- **Controlled parallelism**: Never exceed `workerPoolSize` concurrent goroutines
- **Memory safety**: Prevents unbounded goroutine creation
- **Backpressure**: Blocks RabbitMQ consumption when pool is full

## Performance Tuning Guide

### Increasing Throughput

**If you need to process more events per second:**

1. **Increase WorkerPoolSize** (more concurrency per replica):
   ```go
   // cmd/worker/main.go
   WorkerPoolSize: 50  // From 20 to 50
   ```
   - Also increase `MaxOpenConns` to at least `WorkerPoolSize`

2. **Increase Replicas** (more instances):
   ```yaml
   # docker-compose.yaml
   deploy:
     replicas: 5  # From 3 to 5
   ```
   - Total workers: 5 × 50 = 250 concurrent processors
   - Total DB conns: 5 × 50 = 250 (may need to increase PostgreSQL `max_connections`)

3. **Increase PrefetchCount** (larger buffer):
   ```go
   PrefetchCount: 50  // From 10 to 50
   ```
   - Only helps if message processing is fast (< 100ms)

### Connection Pool Tuning

**Formula:**
```
MaxOpenConns ≥ WorkerPoolSize
Total DB Connections = Replicas × MaxOpenConns
Total DB Connections < PostgreSQL max_connections
```

**Example scaling:**
```
5 replicas × 50 workers × 15 conns = 750 potential connections
→ Need to set PostgreSQL max_connections ≥ 800
```
