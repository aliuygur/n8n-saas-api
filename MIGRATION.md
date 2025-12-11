# Migration from Encore to Standard Go

This document outlines the migration from Encore framework to standard Go with native HTTP routing.

## Completed Steps

### 1. Project Structure
- ✅ Created unified `/migrations` directory (merged from 3 service-specific migration folders)
- ✅ Created `/cmd/server` for main application entry point
- ✅ Created `/internal/config` for configuration management
- ✅ Created `/internal/handler` for unified HTTP handlers

### 2. Configuration
- ✅ Updated `.env.example` with all required environment variables
- ✅ Created `config` package with godotenv support
- ✅ Updated `.gitignore` for standard Go project

### 3. Database
- ✅ Consolidated all migrations into single directory
- ✅ Updated `sqlc.yaml` to use unified migrations
- ✅ Regenerated SQLC code
- ✅ Removed Encore's `sqldb` package dependency

### 4. Dependencies
- ✅ Added `github.com/joho/godotenv` for environment variable management
- ✅ Kept `github.com/jackc/pgx/v5` for PostgreSQL driver
- ✅ Removed `encore.dev` dependency (will need to remove remaining usages)

### 5. Application Structure
- ✅ Created `cmd/server/main.go` with:
  - Structured logging using `slog`
  - Database connection handling
  - HTTP server setup
  - Graceful shutdown
- ✅ Created `internal/handler/handler.go` with unified handler struct
- ✅ Created `internal/handler/routes.go` with all route definitions
- ✅ Created `internal/handler/auth.go` with authentication handlers

## Remaining Steps

### 1. Create Handler Methods
Need to create the following handler files in `/internal/handler/`:
- `frontend.go` - Dashboard, home page, provisioning page handlers
- `instances.go` - Instance creation, deletion, subdomain check handlers
- `provisioning.go` - Provisioning logic (create, delete, get, list instances)
- `subscription.go` - Polar checkout and webhook handlers

### 2. Update Internal Packages
Replace Encore-specific code in:
- `/internal/auth/auth.go` - Remove `encore.dev/beta/auth`, implement custom auth middleware
- `/internal/services/*` - Remove `encore:api` annotations and `encore.dev/rlog`
- Replace `rlog` with standard `slog` throughout the codebase
- Remove `encore.dev/types/uuid` - use `github.com/google/uuid` instead
- Remove `encore.dev/beta/errs` - use standard error handling
- Remove `encore.CurrentRequest().PathParams` - use `http.Request.PathValue()` (Go 1.22+)

### 3. Service Method Refactoring
Since we're merging services, convert Encore "private API" calls to direct method calls:
- `provisioning.CreateInstance()` -> `handler.createInstanceInternal()`
- `provisioning.DeleteInstance()` -> `handler.deleteInstanceInternal()`
- `provisioning.CheckSubdomainExists()` -> `handler.checkSubdomainExistsInternal()`
- `subscription.CreateCheckout()` -> `handler.createCheckoutInternal()`
- etc.

### 4. Database Initialization
Since Encore handled database migrations automatically, you'll need to either:
- Use a migration tool like `golang-migrate/migrate`
- Or manually run migrations before starting the server

### 5. Environment Configuration
Copy `.env.example` to `.env` and fill in actual values:
```bash
cp .env.example .env
# Edit .env with your actual configuration
```

### 6. Build and Run
```bash
# Build the application
go build -o bin/server ./cmd/server

# Run the application
./bin/server
```

## API Routes Mapping

### Encore -> Standard HTTP

| Encore Path | HTTP Method | New Path | Handler Method |
|------------|-------------|----------|----------------|
| `/login` | GET | `/login` | `Login` |
| `/api/auth/logout` | GET | `/api/auth/logout` | `Logout` |
| `/auth/google` | GET | `/auth/google` | `HandleGoogleLogin` |
| `/auth/google/callback` | GET | `/auth/google/callback` | `HandleGoogleCallback` |
| `/api/auth/me` | GET | `/api/auth/me` | `GetAuthMe` |
| `/dashboard` | GET | `/dashboard` | `Dashboard` |
| `/create-instance` | GET | `/create-instance` | `CreateInstancePage` |
| `/provisioning` | GET | `/provisioning` | `ProvisioningPage` |
| `/api/create-instance` | POST | `/api/create-instance` | `CreateInstance` |
| `/api/check-subdomain` | POST | `/api/check-subdomain` | `CheckSubdomain` |
| `/instances/:id` | DELETE | `/instances/{id}` | `DeleteInstance` |
| `/api/delete-modal/:id` | GET | `/api/delete-modal/{id}` | `DeleteModal` |
| `/api/provisioning-status` | GET | `/api/provisioning-status` | `GetProvisioningStatus` |
| `/api/webhooks/polar` | POST | `/api/webhooks/polar` | `PolarWebhook` |
| `/api/checkout-callback` | GET | ~~Removed~~ | Unused - Polar redirects directly to `/provisioning` |
| `/static/*path` | GET | `/static/` | Static file server |
| `/!fallback` | GET | `/` | `Home` (catch-all) |

## Key Changes

### Authentication
- **Before**: Used `encore.dev/beta/auth` with automatic auth middleware
- **After**: Custom JWT validation in `GetUserFromRequest()` method

### Logging
- **Before**: Used `encore.dev/rlog`
- **After**: Standard library `log/slog` with structured logging

### Path Parameters
- **Before**: `encore.CurrentRequest().PathParams.Get("id")`
- **After**: `r.PathValue("id")` (requires Go 1.22+)

### Error Handling
- **Before**: `encore.dev/beta/errs` with error codes
- **After**: Standard Go errors with HTTP status codes

### Database
- **Before**: `encore.dev/storage/sqldb` with automatic migrations
- **After**: Standard `database/sql` with manual migration management

## Notes

- The migration maintains the same database schema
- SQLC-generated code remains unchanged
- All business logic is preserved
- Static files, templates, and assets are unchanged
- GKE, Cloudflare, and Polar integrations remain the same
