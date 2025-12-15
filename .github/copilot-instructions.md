# Copilot Instructions for n8n SaaS Platform

Welcome to the n8n SaaS Platform codebase! This project is designed to be Copilot-friendly and follows clear Go standards and conventions. Below are guidelines and tips to help Copilot (and contributors) work effectively in this repository.

## Project Standards

- **Language:** Go 1.24+
- **Dependency Management:** Go modules (`go.mod`)
- **Code Structure:**
  - `cmd/` — Application entry points
  - `internal/` — Main application logic, organized by domain
  - `pkg/` — Reusable packages/utilities
  - `migrations/` — SQL migration scripts
  - `k8s/` — Kubernetes manifests
  - `scripts/` — Helper scripts
- **Database:** PostgreSQL, with SQLC for type-safe queries
- **Templates:** [templ](https://templ.guide/) for HTML rendering
- **Logging:** Standard Go `log/slog`
- **Configuration:** Environment variables via `.env` and `github.com/joho/godotenv`

## Copilot Usage Tips

- **Follow Go idioms:** Use clear, concise function and variable names. Organize code by domain.
- **Use existing patterns:** Refer to files in `internal/handler` for handler patterns, and `internal/services` for service logic.
- **Database access:** Use the generated code in `internal/db` for all database operations. Avoid raw SQL in handlers.
- **Templates:** Use the `templ` package for HTML. See `internal/handler/components` for examples.
- **Configuration:** All config should be loaded via `internal/config/config.go`.
- **Testing:** Place tests alongside implementation files, using Go's standard testing package.
- **Error handling:** Use the `internal/apperrs` package for custom error types and handling.
- **Logging:** Use the logger from `internal/applogs`.

## Adding New Features

1. Add new handlers in `internal/handler`.
2. Add new services in `internal/services`.
3. Add new database queries in `internal/db/queries/*.sql` and run `make sqlc`.
4. Add new templates in `internal/handler/components` and run `make templ`.
5. Update documentation in `README.md` as needed.

## Dependencies

All dependencies are managed in `go.mod`. Use `go mod tidy` to clean up unused dependencies.

## Code Generation

- Run `make sqlc` to generate database code from SQL files.
- Run `make templ` to generate HTML templates.

## Testing & Formatting

- Run `make test` to execute all tests.
- Run `make fmt` to format code.

## Documentation

- See `README.md` for setup, architecture, and API documentation.
- See `MIGRATION.md` for migration details.

---

**Copilot, please adhere to these standards and patterns when generating code for this project.**
