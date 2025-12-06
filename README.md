# Auction Service

Event-driven microservice for managing auction items with RabbitMQ integration and high-throughput concurrent processing.

## Architecture Overview

The auction service consists of two main components:

### 1. API Service (`cmd/api/`)
**Role:** HTTP REST API + Event Publisher

- Handles item and comment CRUD operations
- Manages categories and item metadata (attributes, images)
- Publishes domain events (items, comments) to RabbitMQ
- Used by external clients and services

**Events Published:**
- `item.created.v1` → When a new item is created
- `item.updated.v1` → When an item is updated
- `item.deleted.v1` → When an item is deleted
- `item.comment.created.v1` → When a comment is added to an item
- `item.comment.deleted.v1` → When a comment is deleted from an item
- `item.image.uploaded.v1` → When an image is uploaded to an item
- `item.image.deleted.v1` → When an image is deleted from an item

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
├── app/                        # Application layer (HTTP handlers)
│   ├── create_item_handler.go
│   ├── get_items_handler.go
│   ├── get_item_handler.go
│   ├── update_item_handler.go
│   ├── delete_item_handler.go
│   ├── get_categories_handler.go
│   ├── get_category_handler.go
│   ├── get_comments_handler.go
│   ├── create_comment_handler.go
│   ├── delete_comment_handler.go
│   ├── get_item_images_handler.go
│   ├── upload_item_image_handler.go
│   ├── delete_item_image_handler.go
│   └── repository.go
│
├── domain/                     # Domain entities
│   ├── item.go
│   ├── category.go
│   ├── item_category.go
│   ├── item_attribute.go
│   ├── item_comment.go
│   └── item_image.go
│
├── internal/
│   ├── middleware/             # HTTP middlewares
│   └── consumers/              # Event consumer handlers
│       └── bid_consumer.go     # Handles bid events
│
├── infra/
│   ├── postgres/               # Database layer
│   │   ├── migrations/
│   │   │   ├── 001_create_items.sql
│   │   │   ├── 002_create_categories.sql
│   │   │   ├── 003_create_item_categories.sql
│   │   │   ├── 004_create_item_attributes.sql
│   │   │   ├── 005_create_item_comments.sql
│   │   │   └── 006_create_item_images.sql
│   │   └── repository.go       # Connection pool tuning
│   └── rabbitmq/               # Message broker
│       ├── publisher.go        # Event publishing
│       └── consumer.go         # Concurrent event consuming
│
├── pkg/
│   ├── config/                 # Configuration management
│   ├── events/                 # Event schemas
│   ├── httperror/              # HTTP error handling
│   └── aws/                    # AWS S3 integration
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

### Public Endpoints (No Authentication Required)

**Items:**
- `GET /api/v1/items` - List all items with pagination and filtering
- `GET /api/v1/items/:id` - Get item details by ID
- `GET /api/v1/items/:id/comments` - Get all comments for an item
- `GET /api/v1/items/:id/images` - Get all images for an item

**Categories:**
- `GET /api/v1/categories` - List all categories
- `GET /api/v1/categories/:id` - Get category details by ID

### Private Endpoints (Require X-User-ID header)

**Items:**
- `POST /api/v1/items` - Create new auction item
- `PUT /api/v1/items/:id` - Update item details
- `DELETE /api/v1/items/:id` - Delete item

**Comments:**
- `POST /api/v1/items/:id/comments` - Add comment to an item
- `DELETE /api/v1/items/:itemId/comments/:commentId` - Delete a comment

**Images:**
- `POST /api/v1/items/:id/images` - Upload image to an item (multipart/form-data)
- `DELETE /api/v1/items/:itemId/images/:imageId` - Delete an image from an item

### API Examples

#### Create Item
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

#### Get Items with Filtering
```bash
# List all active items
curl -X GET "http://localhost:8081/api/v1/items?status=active"

# Get items with pagination
curl -X GET "http://localhost:8081/api/v1/items?page=1&limit=20"
```

#### Add Comment to Item
```bash
curl -X POST http://localhost:8081/api/v1/items/item-uuid/comments \
  -H "Content-Type: application/json" \
  -H "X-User-ID: user-123" \
  -d '{
    "content": "Is this item still available?"
  }'
```

#### Get Categories
```bash
curl -X GET http://localhost:8081/api/v1/categories
```

#### Upload Item Image
```bash
curl -X POST http://localhost:8081/api/v1/items/item-uuid/images \
  -H "X-User-ID: user-123" \
  -F "image=@/path/to/image.jpg"
```

#### Delete Item Image
```bash
curl -X DELETE http://localhost:8081/api/v1/items/item-uuid/images/image-uuid \
  -H "X-User-ID: user-123"
```

## Event Integration

### Publishing Events (API Service)

When items and comments are created/updated/deleted, the API service automatically publishes events:

#### Item Events

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
    "sellerId": "user-123",
    "startPrice": "100.00",
    "currentPrice": "100.00",
    "currencyCode": "USD"
  }
}
```

#### Comment Events

```json
// Publishes to: auction.item exchange
// Routing key: item.comment.created.v1
{
  "event": "item.comment.created",
  "version": "v1",
  "timestamp": "2024-01-15T10:00:00Z",
  "traceId": "trace-uuid",
  "correlationId": "correlation-uuid",
  "payload": {
    "id": "comment-uuid",
    "itemId": "item-uuid",
    "authorId": "user-123",
    "content": "Is this item still available?",
    "createdAt": "2024-01-15T10:00:00Z"
  }
}
```

#### Image Events

```json
// Publishes to: auction.item exchange
// Routing key: item.image.uploaded.v1
{
  "event": "item.image.uploaded",
  "version": "v1",
  "timestamp": "2024-01-15T10:00:00Z",
  "traceId": "trace-uuid",
  "correlationId": "correlation-uuid",
  "payload": {
    "id": "image-uuid",
    "itemId": "item-uuid",
    "imageUrl": "https://s3.amazonaws.com/bucket/items/item-uuid/image-uuid.jpg",
    "createdAt": "2024-01-15T10:00:00Z"
  }
}

// Routing key: item.image.deleted.v1
{
  "event": "item.image.deleted",
  "version": "v1",
  "timestamp": "2024-01-15T10:00:00Z",
  "traceId": "trace-uuid",
  "correlationId": "correlation-uuid",
  "payload": {
    "id": "image-uuid",
    "itemId": "item-uuid",
    "imageUrl": "https://s3.amazonaws.com/bucket/items/item-uuid/image-uuid.jpg",
    "deletedAt": "2024-01-15T10:00:00Z"
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

### Event Use Cases

**Item Events** are consumed by:
- **Payment Service** → Process payments when items are sold
- **Notification Service** → Notify users about item updates
- **Analytics Service** → Track item creation and sales metrics

**Comment Events** are consumed by:
- **Notification Service** → Notify item sellers when new questions are asked
- **Moderation Service** → Review comments for inappropriate content
- **Analytics Service** → Track user engagement and activity

**Image Events** are consumed by:
- **CDN Service** → Cache and optimize images for delivery
- **Image Processing Service** → Generate thumbnails and resize images
- **Moderation Service** → Scan images for inappropriate content
- **Analytics Service** → Track media uploads and storage metrics

## Domain Model

The auction service supports a rich domain model for marketplace items:

### Core Entities

**Item** - The main auction item with pricing and timing information
- Basic info: name, description, seller ID
- Pricing: start price, current price, bid increment, currency
- Timing: start date, end date
- Status: active, sold, expired, cancelled
- Buyer info: set when item is sold

**Category** - Hierarchical classification system for items
- Supports nested categories (parent-child relationships)
- Items can belong to multiple categories

**ItemAttribute** - Custom key-value metadata for items
- Flexible schema for item-specific properties
- Examples: brand, condition, size, color, year

**ItemComment** - User discussions and questions about items
- User-generated content
- Timestamped for chronological ordering
- Supports moderation via deletion

**ItemImage** - Photo gallery for items
- Multiple images per item
- Display order support
- Image URLs for external storage (AWS S3 / MinIO)
- Supports PNG and JPEG/JPG formats
- Maximum file size: 5MB per image

### Relationships

```
Item (1) ←→ (N) ItemCategory ←→ (1) Category
Item (1) ←→ (N) ItemAttribute
Item (1) ←→ (N) ItemComment
Item (1) ←→ (N) ItemImage
```

## Database Migrations

Migrations are automatically applied on startup via Docker `docker-entrypoint-initdb.d`.

Location: `infra/postgres/migrations/`

Current migrations:
- `001_create_items.sql` - Creates items table with triggers and indexes
- `002_create_categories.sql` - Creates categories table for item classification
- `003_create_item_categories.sql` - Creates many-to-many relationship between items and categories
- `004_create_item_attributes.sql` - Creates item attributes table for custom metadata
- `005_create_item_comments.sql` - Creates comments table for user discussions
- `006_create_item_images.sql` - Creates images table for item photos

## Image Storage (AWS S3 / MinIO)

The auction service supports uploading and managing item images using AWS S3 or MinIO (S3-compatible storage).

### Features

- **Multipart Form Upload**: Upload images via `multipart/form-data` with field name `image`
- **File Validation**:
  - Supported formats: PNG, JPEG, JPG
  - Maximum file size: 5MB
  - Content-Type validation
- **Authorization**: Only item sellers can upload/delete images for their items
- **Event-Driven**: Publishes `item.image.uploaded.v1` and `item.image.deleted.v1` events
- **URL Construction**: Automatic image URL generation for both AWS S3 and MinIO

### Storage Structure

Images are stored in the following S3 key pattern:
```
items/{itemId}/{uuid}.{extension}

Example:
items/abc123-def456/78910-xyz123.jpg
```

### Using MinIO (Development)

MinIO is an S3-compatible object storage server ideal for local development:

```bash
# Start MinIO with Docker
docker run -d \
  --name minio \
  -p 9000:9000 \
  -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"

# Access MinIO Console: http://localhost:9001
# Credentials: minioadmin / minioadmin
```

Environment configuration for MinIO:
```env
AWS_ENDPOINT=http://localhost:9000
AWS_ACCESS_KEY=minioadmin
AWS_SECRET_KEY=minioadmin
AWS_BUCKET=auction-images
```

### Using AWS S3 (Production)

For production deployment with AWS S3:

1. Create an S3 bucket (e.g., `auction-images-prod`)
2. Configure bucket policy for public read access (optional)
3. Create IAM user with S3 permissions (`s3:PutObject`, `s3:GetObject`, `s3:DeleteObject`)
4. Set environment variables:

```env
AWS_ENDPOINT=                              # Leave empty for AWS S3
AWS_ACCESS_KEY=AKIA...                     # IAM access key
AWS_SECRET_KEY=wJal...                     # IAM secret key
AWS_DEFAULT_REGION=eu-central-1            # Your AWS region
AWS_BUCKET=auction-images-prod             # Your bucket name
```

### Image URL Format

**MinIO/Custom Endpoint:**
```
http(s)://{AWS_ENDPOINT}/{AWS_BUCKET}/{key}
Example: http://localhost:9000/auction-images/items/abc123/image.jpg
```

**AWS S3:**
```
https://{bucket}.s3.{region}.amazonaws.com/{key}
Example: https://auction-images.s3.eu-central-1.amazonaws.com/items/abc123/image.jpg
```

### Error Handling

The image upload handler includes comprehensive error handling:

- **Missing file**: Returns `400 Bad Request` if `image` field is missing
- **File too large**: Returns `400 Bad Request` if file exceeds 5MB
- **Invalid format**: Returns `400 Bad Request` if not PNG/JPEG/JPG
- **Unauthorized**: Returns `403 Forbidden` if user is not the item seller
- **Storage failure**: Returns `500 Internal Server Error` if S3 upload fails
- **Rollback**: Automatically deletes S3 object if database save fails

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

# AWS S3 / MinIO (for image storage)
AWS_ENDPOINT=http://localhost:9000          # MinIO endpoint (leave empty for AWS S3)
AWS_ACCESS_KEY=minioadmin                   # S3 access key
AWS_SECRET_KEY=minioadmin                   # S3 secret key
AWS_DEFAULT_REGION=eu-central-1             # AWS region
AWS_BUCKET=auction-images                   # Bucket name
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

---

> **Note**: This documentation was generated and maintained with assistance from Claude Code AI.
