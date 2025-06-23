# ZBZ Framework

A convention-over-configuration Go framework for building type-safe APIs with automatic CRUD generation, OpenAPI documentation, and structured error handling.

## Features

- **🚀 Automatic CRUD Generation** - Define models, get REST endpoints
- **📚 Auto-Generated Documentation** - OpenAPI specs with interactive docs
- **🛡️ Built-in Error Handling** - Consistent, structured error responses
- **🔒 Authentication Ready** - OIDC integration out of the box
- **📊 Observability** - Structured logging, metrics, and distributed tracing
- **⚡ Performance First** - Type-safe operations with minimal reflection
- **🎯 Opinionated** - Sensible defaults that just work

## Quick Start

### Prerequisites

- Go 1.23.1 or later
- PostgreSQL database
- Docker (optional, for local development)

### Installation

```bash
git clone <repository-url>
cd zbz
go mod download
```

### Basic Usage

1. **Define your models:**

```go
// internal/models/user.go
type User struct {
    zbz.Model  // Provides ID, CreatedAt, UpdatedAt
    Name  string `db:"name" json:"name" validate:"required" desc:"The user's full name"`
    Email string `db:"email" json:"email" validate:"required,email" desc:"User's email address"`
}
```

2. **Create your application:**

```go
// main.go
func main() {
    e := zbz.NewEngine()
    
    // Auto-generate CRUD endpoints for User
    userCore := zbz.NewCore[models.User]("A user in the system")
    e.Inject(userCore)
    
    e.Start() // Runs on :8080 by default
}
```

3. **That's it!** You now have:
   - `GET /user/{id}` - Get user by ID
   - `POST /user` - Create new user  
   - `PUT /user/{id}` - Update user
   - `DELETE /user/{id}` - Delete user
   - `GET /docs` - Interactive API documentation
   - `GET /openapi` - OpenAPI specification

## Development Setup

### Environment Variables

```bash
# Database
DATABASE_URL="postgres://user:password@localhost:5432/dbname?sslmode=disable"

# Authentication (optional)
AUTH_DOMAIN="your-auth0-domain.auth0.com"
AUTH_CLIENT_ID="your-auth0-client-id"
AUTH_CLIENT_SECRET="your-auth0-client-secret"

# Server
PORT="8080"
APP_TITLE="My API"
APP_DESCRIPTION="Description of my API"
APP_VERSION="1.0.0"
```

### Running the Example

```bash
# Start PostgreSQL (if using Docker)
docker run --name zbz-postgres -e POSTGRES_PASSWORD=password -e POSTGRES_DB=zbz -p 5432:5432 -d postgres:15

# Set environment variables
export DATABASE_URL="postgres://postgres:password@localhost:5432/zbz?sslmode=disable"

# Run the example application
go run ./cmd/example
```

### Development with Docker Compose

```bash
# Start all services (app + database + observability stack)
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

## Project Structure

```
zbz/
├── cmd/
│   └── example/           # Example application
│       ├── main.go
│       └── internal/
│           └── models/    # Example models
├── lib/                   # Framework core
│   ├── core.go           # CRUD operations
│   ├── engine.go         # Service orchestration
│   ├── http.go           # HTTP server
│   ├── database.go       # Database layer
│   ├── error.go          # Error handling
│   ├── auth.go           # Authentication
│   ├── docs.go           # OpenAPI generation
│   ├── logger.go         # Structured logging
│   ├── model.go          # Base model types
│   ├── validate.go       # Validation
│   ├── macros/           # SQL query templates
│   └── templates/        # HTML templates
├── docker-compose.yaml   # Local development stack
├── Dockerfile.dev        # Development container
├── go.mod
├── go.sum
├── README.md
└── CLAUDE.md             # Development context
```

## Technology Stack

### Core Dependencies
- **[Gin](https://github.com/gin-gonic/gin)** - HTTP web framework
- **[SQLx](https://github.com/jmoiron/sqlx)** - SQL toolkit with enhanced database/sql
- **[PostgreSQL Driver](https://github.com/lib/pq)** - Pure Go Postgres driver
- **[Zap](https://github.com/uber-go/zap)** - Structured logging
- **[Validator](https://github.com/go-playground/validator)** - Struct validation

### Observability
- **[OpenTelemetry](https://opentelemetry.io/)** - Distributed tracing
- **[Prometheus](https://prometheus.io/)** - Metrics collection
- **[Loki](https://grafana.com/oss/loki/)** - Log aggregation
- **[Tempo](https://grafana.com/oss/tempo/)** - Trace storage

### Authentication
- **[OIDC](https://github.com/coreos/go-oidc)** - OpenID Connect integration
- **[OAuth2](https://golang.org/x/oauth2)** - OAuth 2.0 client

## API Documentation

Once running, visit:
- **Interactive Docs**: http://localhost:8080/docs
- **OpenAPI Spec**: http://localhost:8080/openapi  
- **Health Check**: http://localhost:8080/health

## Model Definition

Models use struct tags to define behavior:

```go
type Product struct {
    zbz.Model
    Name        string  `db:"name" json:"name" validate:"required" desc:"Product name"`
    Price       float64 `db:"price" json:"price" validate:"required,gt=0" desc:"Price in USD"`
    Description string  `db:"description" json:"description" desc:"Product description"`
    CategoryID  string  `db:"category_id" json:"categoryId" validate:"required,uuid" desc:"Category reference"`
}
```

### Struct Tags
- `db:"field_name"` - Database column name
- `json:"fieldName"` - JSON field name  
- `validate:"rules"` - Validation rules
- `desc:"description"` - Field description for docs
- `edit:"permission"` - Edit permissions (future feature)
- `ex:"example"` - Example value for docs

## Error Handling

The framework provides automatic, structured error responses:

```json
{
  "error": "Not Found",
  "message": "The requested resource was not found", 
  "code": "Not Found"
}
```

### Custom Error Messages

```go
func MyHandler(ctx *gin.Context) {
    // Custom error message
    ctx.Set("error_message", "User with that email already exists")
    ctx.Status(http.StatusConflict)
    // Framework automatically adds structured error response
}
```

## Advanced Usage

### Custom Error Responses

```go
// Customize default error responses
engine := zbz.NewEngine()
http := engine.HTTP() // Get HTTP service

customError := &zbz.Error{
    Status: http.StatusNotFound,
    Name: "CustomNotFound", 
    Description: "Custom not found error",
    Response: zbz.ErrorResponse{
        Error: "Resource Missing",
        Message: "The thing you're looking for isn't here",
        Code: "Not Found",
    },
}

http.SetErrorResponse(http.StatusNotFound, customError)
```

### Database Queries

The framework uses SQL macros for type-safe queries:

```sql
-- internal/macros/find_users_by_email.sqlx
-- @embed email The email to search for
SELECT {{columns}} FROM users WHERE email = :email;
```

## Monitoring

### Logs
- **Format**: Structured JSON (production) / Console (development)
- **Levels**: Debug, Info, Warn, Error, Fatal
- **Fields**: Strongly typed with zap fields

### Metrics
- Automatic HTTP request metrics via Prometheus
- Custom business metrics can be added

### Tracing  
- Automatic request tracing via OpenTelemetry
- Integrates with Jaeger, Tempo, and other trace collectors

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Update tests and documentation
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Development Guidelines

- Follow Go best practices and conventions
- Write tests for new functionality  
- Update documentation for user-facing changes
- Use structured logging with typed fields
- Maintain service independence in architecture

## License

[License Type] - see LICENSE file for details

## Support

- **Issues**: [GitHub Issues](link-to-issues)
- **Discussions**: [GitHub Discussions](link-to-discussions)
- **Documentation**: [Full Documentation](link-to-docs)

---

Built with ❤️ for developers who want to focus on business logic, not boilerplate.