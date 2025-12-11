# n8n SaaS Platform - Standard Go Version

A managed n8n hosting platform built with standard Go, featuring Google OAuth authentication, GKE-based provisioning, and Polar subscription management.

## 🚀 Quick Start

### Prerequisites

- Go 1.24+
- PostgreSQL database
- Google Cloud Platform account (for GKE)
- Cloudflare account (for DNS/tunneling)
- Polar account (for subscriptions)

### Setup

1. **Clone and install dependencies**
   ```bash
   go mod download
   ```

2. **Configure environment**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

3. **Run database migrations**
   ```bash
   psql $DATABASE_URL < migrations/001_init.up.sql
   ```

4. **Build and run**
   ```bash
   make build
   make run
   ```

   Or directly:
   ```bash
   go run ./cmd/server
   ```

The server will start on `http://localhost:8080` by default.

## 📁 Project Structure

```
.
├── cmd/
│   └── server/          # Main application entry point
├── internal/
│   ├── auth/            # Authentication utilities
│   ├── cloudflare/      # Cloudflare client
│   ├── config/          # Configuration management
│   ├── db/              # SQLC generated database code
│   ├── gke/             # Google Kubernetes Engine client
│   ├── handler/         # HTTP handlers (main business logic)
│   └── services/        # Old Encore services (being phased out)
├── migrations/          # Database migrations
├── pkg/
│   └── domainutils/     # Domain validation utilities
└── Makefile             # Build and development commands
```

## 🛠 Development

### Available Make Commands

```bash
make build      # Build the application
make run        # Build and run
make dev        # Run with live reload (requires air)
make clean      # Clean build artifacts
make sqlc       # Generate SQLC code
make templ      # Generate templ templates
make test       # Run tests
make fmt        # Format code
make tidy       # Tidy dependencies
```

### Environment Variables

See [.env.example](.env.example) for all required configuration. Key variables:

- `DATABASE_URL` - PostgreSQL connection string
- `JWT_SECRET` - Secret for JWT token signing
- `GOOGLE_CLIENT_ID` / `GOOGLE_CLIENT_SECRET` - Google OAuth
- `GCP_*` - Google Cloud Platform configuration
- `CLOUDFLARE_*` - Cloudflare API configuration
- `POLAR_*` - Polar subscription configuration

## 🔄 API Routes

### Authentication
- `GET /login` - Login page
- `GET /auth/google` - Initiate Google OAuth
- `GET /auth/google/callback` - OAuth callback
- `GET /api/auth/logout` - Logout
- `GET /api/auth/me` - Get current user

### Frontend
- `GET /` - Home page
- `GET /dashboard` - User dashboard
- `GET /create-instance` - Create instance page
- `GET /provisioning` - Provisioning status page

### Instance Management
- `POST /api/create-instance` - Create new instance
- `POST /api/check-subdomain` - Check subdomain availability
- `DELETE /instances/{id}` - Delete instance
- `GET /api/delete-modal/{id}` - Get delete confirmation modal
- `GET /api/provisioning-status` - Get provisioning status (HTMX polling)

### Webhooks
- `POST /api/webhooks/polar` - Polar subscription webhooks

### Static Files
- `/static/*` - Static assets (CSS, images, etc.)

## 🏗 Architecture

### Handler-Based Architecture

The application uses a handler-based architecture where all HTTP endpoints are managed by the `internal/handler` package:

```go
// Handler holds all dependencies
type Handler struct {
    db             *db.Queries
    gke            *gke.Client
    cloudflare     *cloudflare.Client
    polarClient    *polargo.Polar
    oauth2Config   *oauth2.Config
    jwtSecret      []byte
    config         *config.Config
    logger         *slog.Logger
}
```

### Key Components

- **Auth**: JWT-based authentication with Google OAuth
- **Database**: SQLC for type-safe SQL queries
- **Logging**: Standard library `log/slog` for structured logging
- **Provisioning**: GKE-based instance deployment
- **Subscriptions**: Polar for payment processing
- **Templates**: templ for type-safe HTML templates

## 🔐 Authentication Flow

1. User clicks "Login with Google"
2. Redirected to Google OAuth
3. Callback creates/updates user in database
4. JWT token issued and stored in HTTP-only cookie
5. Subsequent requests validated via JWT middleware

## 🚢 Deployment

### Building for Production

```bash
# Build optimized binary
CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/server ./cmd/server

# Run
./bin/server
```

### Docker (Example)

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
COPY --from=builder /app/internal/services/frontend/static ./internal/services/frontend/static
CMD ["./server"]
```

## 📊 Database Schema

The application uses a unified PostgreSQL schema with the following tables:

- `users` - User accounts
- `sessions` - Session management
- `instances` - n8n instance records
- `subscriptions` - Polar subscriptions
- `checkout_sessions` - Polar checkout tracking

See [migrations/001_init.up.sql](migrations/001_init.up.sql) for full schema.

## 🧪 Testing

```bash
# Run all tests
make test

# Run with coverage
go test -v -cover ./...

# Run specific package
go test ./internal/handler
```

## 📚 Documentation

- [MIGRATION.md](MIGRATION.md) - Detailed migration guide from Encore
- [MIGRATION_STATUS.md](MIGRATION_STATUS.md) - Current migration progress
- [.env.example](.env.example) - Environment variable reference

## 🐛 Troubleshooting

### Build Errors

If you encounter build errors related to Encore:
```bash
go mod tidy
go clean -modcache
go mod download
```

### Database Connection

Ensure your DATABASE_URL is correctly formatted:
```
postgresql://user:password@localhost:5432/database?sslmode=disable
```

### Missing Templates

If templ templates are missing:
```bash
make templ
```

## 📝 License

[Your License Here]

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and formatting
5. Submit a pull request

## 📧 Support

For issues and questions, please open a GitHub issue.
