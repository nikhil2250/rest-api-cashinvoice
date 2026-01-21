# Task Management REST API

A scalable RESTful Task Management Service built with Go, demonstrating API design, MongoDB persistence, concurrency, authentication, and clean architecture.

## Features

- **RESTful API**: Complete CRUD operations for task management
- **Authentication**: JWT-based authentication with secure password hashing
- **Authorization**: Role-based access control (User/Admin)
- **Pagination**: Efficient pagination for task listings
- **Filtering**: Filter tasks by status (pending, in_progress, completed)
- **Concurrency**: Background worker for auto-completing tasks using goroutines and channels
- **MongoDB**: NoSQL database with clean repository pattern
- **Docker**: Fully containerized with Docker Compose
- **Clean Architecture**: Separation of concerns with handlers, services, and repositories
- **Thread Safety**: Mutex-protected database operations
- **Graceful Shutdown**: Proper context-based shutdown handling

## Architecture

```
├── main.go                 # Application entry point with graceful shutdown
├── config.go              # Configuration management
├── models.go              # Data models and DTOs
├── database.go            # MongoDB connection and indexes
├── user_repository.go     # User data access layer
├── task_repository.go     # Task data access layer with pagination
├── auth_service.go        # Authentication logic and JWT handling
├── task_service.go        # Task business logic
├── auth_handler.go        # Authentication HTTP handlers
├── task_handler.go        # Task HTTP handlers with filtering
├── worker.go              # Background worker for auto-completion
├── utils.go               # Helper functions
├── go.mod                 # Go module dependencies
├── Dockerfile             # Application container
├── docker-compose.yml     # Docker services orchestration
├── Makefile               # Build and run commands
└── .env.example           # Environment variables template
```

## Prerequisites

- Docker & Docker Compose (recommended)
- OR Go 1.21+ and MongoDB 7.0+ (for local development)

## Quick Start with Docker

1. Clone the repository:
```bash
git clone https://github.com/yourusername/task-management-api.git
cd task-management-api
```

2. Create `.env` file:
```bash
cp .env.example .env
```

3. Start the application with Docker Compose:
```bash
docker-compose up -d
```

4. Check logs:
```bash
docker-compose logs -f app
```

The API will be available at `http://localhost:8080`

### Docker Commands

```bash
# Start services
docker-compose up -d

# Stop services
docker-compose down

# View logs
docker-compose logs -f

# Rebuild after code changes
docker-compose up -d --build

# Clean everything (including volumes)
docker-compose down -v
```

## Local Development Setup

1. Install dependencies:
```bash
go mod download
```

2. Start MongoDB:
```bash
docker run -d -p 27017:27017 \
  -e MONGO_INITDB_ROOT_USERNAME=admin \
  -e MONGO_INITDB_ROOT_PASSWORD=password123 \
  --name mongodb mongo:7.0
```

3. Create `.env` file:
```bash
cp .env.example .env
```

4. Run the application:
```bash
go run .
```

## API Endpoints

### Authentication

#### Register a new user
```http
POST /register
Content-Type: application/json

{
  "email": "user@example.com",
  "username": "johndoe",
  "password": "password123"
}
```

Response:
```json
{
  "id": "507f1f77bcf86cd799439011",
  "email": "user@example.com",
  "username": "johndoe",
  "role": "user",
  "created_at": "2024-01-21T10:00:00Z"
}
```

#### Login
```http
POST /login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "507f1f77bcf86cd799439011",
    "email": "user@example.com",
    "username": "johndoe",
    "role": "user",
    "created_at": "2024-01-21T10:00:00Z"
  }
}
```

### Tasks (Protected Routes)

All task endpoints require the `Authorization` header:
```
Authorization: Bearer <jwt-token>
```

#### Create a task
```http
POST /tasks
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "title": "Complete assignment",
  "description": "Finish the Go REST API",
  "status": "pending"
}
```

Response:
```json
{
  "id": "507f191e810c19729de860ea",
  "user_id": "507f1f77bcf86cd799439011",
  "title": "Complete assignment",
  "description": "Finish the Go REST API",
  "status": "pending",
  "created_at": "2024-01-21T10:00:00Z",
  "updated_at": "2024-01-21T10:00:00Z"
}
```

#### List all tasks (with pagination and filtering)
```http
GET /tasks?page=1&limit=10&status=pending
Authorization: Bearer <jwt-token>
```

Query Parameters:
- `page` (optional, default: 1) - Page number
- `limit` (optional, default: 10, max: 100) - Items per page
- `status` (optional) - Filter by status: `pending`, `in_progress`, or `completed`

Response:
```json
{
  "tasks": [
    {
      "id": "507f191e810c19729de860ea",
      "user_id": "507f1f77bcf86cd799439011",
      "title": "Complete assignment",
      "description": "Finish the Go REST API",
      "status": "pending",
      "created_at": "2024-01-21T10:00:00Z",
      "updated_at": "2024-01-21T10:00:00Z"
    }
  ],
  "page": 1,
  "limit": 10,
  "total_count": 25,
  "total_pages": 3
}
```

#### Get a specific task
```http
GET /tasks/{id}
Authorization: Bearer <jwt-token>
```

#### Delete a task
```http
DELETE /tasks/{id}
Authorization: Bearer <jwt-token>
```

Response:
```json
{
  "message": "task deleted successfully"
}
```

#### Health Check
```http
GET /health
```

Response:
```json
{
  "status": "healthy"
}
```

## Task Status Values

- `pending` - Task is pending
- `in_progress` - Task is in progress
- `completed` - Task is completed

## Authorization Rules

- **Regular Users**: Can only access and manage their own tasks
- **Admin Users**: Can access and manage all tasks

## Background Worker

The application includes a background worker that automatically marks tasks as completed after a configurable time period (default: 10 minutes).

**Features:**
- Runs in separate goroutines with proper context handling
- Uses channels for task queue management
- Thread-safe database access with RWMutex
- Only auto-completes tasks in `pending` or `in_progress` status
- Respects manually completed or deleted tasks
- Configurable via `AUTO_COMPLETE_MINUTES` environment variable
- Gracefully shuts down with the application

**How it works:**
1. Worker checks for eligible tasks every minute
2. Tasks older than the threshold are queued via channels
3. Multiple worker goroutines (3) process the queue concurrently
4. Task status is updated to `completed` and persisted to MongoDB
5. Worker stops gracefully when application receives shutdown signal

## Graceful Shutdown

The application implements comprehensive graceful shutdown:

1. **Signal Handling**: Listens for SIGINT and SIGTERM signals
2. **Context Cancellation**: Cancels background worker context
3. **HTTP Server Shutdown**: Gracefully stops accepting new requests with 10s timeout
4. **Database Cleanup**: Properly closes MongoDB connections with 5s timeout
5. **Channel Cleanup**: Closes worker channels to stop goroutines
6. **Resource Release**: Ensures all resources are properly released

```go
// Shutdown sequence:
// 1. Receive signal (Ctrl+C or kill)
// 2. Cancel worker context
// 3. Shutdown HTTP server (10s timeout)
// 4. Close database connection (5s timeout)
// 5. Exit gracefully
```

## Pagination & Filtering

### Pagination
- Default page size: 10 items
- Maximum page size: 100 items
- Returns metadata: `page`, `limit`, `total_count`, `total_pages`
- Zero-based page numbering starts at 1

### Filtering
- Filter by task status: `?status=pending`, `?status=in_progress`, `?status=completed`
- Invalid status values return 400 error with message
- Filtering works with pagination

### Examples:

```bash
# Get first page with default limit (10)
curl -H "Authorization: Bearer TOKEN" \
  http://localhost:8080/tasks

# Get second page with 20 items per page
curl -H "Authorization: Bearer TOKEN" \
  "http://localhost:8080/tasks?page=2&limit=20"

# Get all pending tasks
curl -H "Authorization: Bearer TOKEN" \
  "http://localhost:8080/tasks?status=pending"

# Get completed tasks with pagination
curl -H "Authorization: Bearer TOKEN" \
  "http://localhost:8080/tasks?status=completed&page=1&limit=5"
```

## Error Handling

The API returns consistent JSON error responses:

```json
{
  "error": "Bad Request",
  "message": "Specific error message"
}
```

### HTTP Status Codes

- `200 OK` - Successful request
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Missing or invalid authentication
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server error

## MongoDB Collections

### Users Collection
```javascript
{
  _id: ObjectId,
  email: String (unique, indexed),
  username: String,
  password: String (hashed),
  role: String, // "user" or "admin"
  created_at: Date
}
```

### Tasks Collection
```javascript
{
  _id: ObjectId,
  user_id: ObjectId (indexed),
  title: String,
  description: String,
  status: String (indexed), // "pending", "in_progress", "completed"
  created_at: Date (indexed, descending),
  updated_at: Date
}
```

## Security Features

- Password hashing using bcrypt (cost factor 10)
- JWT-based authentication with 24-hour expiry
- Role-based authorization (User/Admin)
- Protected routes with middleware
- NoSQL injection prevention through MongoDB driver
- Context-based request handling
- Secure environment variable configuration

## Concurrency & Thread Safety

- **Goroutines**: Background worker runs in separate goroutines
- **Channels**: Task queue implemented with buffered channels (capacity: 100)
- **Mutex**: Repository operations protected with RWMutex for concurrent reads
- **Context**: Proper context propagation for graceful shutdown
- **Worker Pool**: 3 concurrent workers processing tasks
- **Non-blocking**: Background processing doesn't block API requests

## Configuration

All configuration is managed through environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8080` |
| `MONGODB_URI` | MongoDB connection string | `mongodb://admin:password123@localhost:27017` |
| `MONGODB_DATABASE` | Database name | `taskdb` |
| `JWT_SECRET` | JWT signing secret | `your-secret-key-change-in-production` |
| `AUTO_COMPLETE_MINUTES` | Auto-completion delay | `10` |

## Testing the API

### Using cURL

1. **Register a user:**
```bash
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","username":"testuser","password":"password123"}'
```

2. **Login:**
```bash
curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'
```

3. **Create a task:**
```bash
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{"title":"Test Task","description":"Testing the API","status":"pending"}'
```

4. **List tasks with pagination:**
```bash
curl -X GET "http://localhost:8080/tasks?page=1&limit=5" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

5. **Filter tasks by status:**
```bash
curl -X GET "http://localhost:8080/tasks?status=completed" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

6. **Get a specific task:**
```bash
curl -X GET http://localhost:8080/tasks/TASK_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

7. **Delete a task:**
```bash
curl -X DELETE http://localhost:8080/tasks/TASK_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Using Postman

1. Import the following collection settings:
   - Base URL: `http://localhost:8080`
   - Add Authorization header: `Bearer {{token}}`
2. Create environment variable `token` after login
3. Test all endpoints with different scenarios

## Makefile Commands

```bash
make help           # Show available commands
make deps           # Download dependencies
make build          # Build the binary
make run            # Run locally
make docker-build   # Build Docker images
make docker-up      # Start with Docker Compose
make docker-down    # Stop containers
make docker-logs    # View container logs
make clean          # Clean build artifacts and volumes
```

## Project Structure Principles

- **Clean Architecture**: Clear separation between layers
- **Repository Pattern**: Abstraction of data access logic
- **Dependency Injection**: Services and repositories are injected
- **Idiomatic Go**: Follows Go best practices and conventions
- **Error Handling**: Consistent error handling throughout
- **Context Usage**: Proper context propagation for cancellation
- **Graceful Degradation**: Handles failures gracefully

## Performance Considerations

- Connection pooling for MongoDB
- Indexed queries for efficient filtering
- Pagination to limit memory usage
- Buffered channels for worker queue
- Concurrent request handling
- Read/Write mutex for optimal concurrency

## Production Checklist

- [ ] Change `JWT_SECRET` to a strong random value
- [ ] Use environment-specific MongoDB credentials
- [ ] Enable MongoDB authentication
- [ ] Set up MongoDB replica set for production
- [ ] Configure proper logging
- [ ] Add rate limiting middleware
- [ ] Set up monitoring and alerting
- [ ] Enable HTTPS/TLS
- [ ] Configure CORS properly
- [ ] Add input sanitization
- [ ] Set up backup strategy
- [ ] Configure resource limits

## Implemented Bonus Features

✅ **Pagination & Filtering**: Full pagination support with status filtering  
✅ **Dockerfile**: Multi-stage Docker build for optimized image  
✅ **Graceful Shutdown**: Context-based shutdown for all components  
✅ **Docker Compose**: Complete orchestration with MongoDB  
✅ **Makefile**: Convenient build and run commands  

## Future Enhancements (Not Yet Implemented)

- Unit and integration tests
- Swagger/OpenAPI documentation
- Advanced logging with structured logger (zerolog/zap)
- Metrics and monitoring (Prometheus)
- Rate limiting middleware
- Task update endpoint (PATCH/PUT)
- Task search with text indexing
- Soft delete functionality
- Task assignment to multiple users
- Task priority field
- Task due dates and reminders

## Troubleshooting

### MongoDB Connection Issues
```bash
# Check MongoDB is running
docker ps | grep mongo

# Check MongoDB logs
docker logs task-mongodb

# Test connection
docker exec -it task-mongodb mongosh -u admin -p password123
```

### Application Not Starting
```bash
# Check application logs
docker logs task-api

# Verify environment variables
docker exec task-api env | grep MONGODB
```

### Port Already in Use
```bash
# Change port in .env file
PORT=8081

# Or stop conflicting service
lsof -ti:8080 | xargs kill -9
```

## License

MIT License

## Author

Task Management API - Golang Assignment Implementation