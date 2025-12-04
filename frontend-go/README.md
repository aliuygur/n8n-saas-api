# Frontend Go - N8N SaaS Frontend

A modern web frontend for the N8N SaaS platform built with Go and [templ](https://templ.guide/).

## Features

- 🚀 Type-safe HTML templating with templ
- ⚡ Dynamic UI updates with HTMX (no JavaScript frameworks needed)
- 🎨 Styled with Tailwind CSS
- 📱 Responsive design
- 🔌 Direct integration with Encore backend API
- ✅ Compile-time template validation

## Tech Stack

- **Backend**: Go 1.23+
- **Router**: Chi
- **Frontend**: HTMX + Tailwind CSS
- **Templates**: templ (type-safe Go templating)

## Project Structure

```
frontend-go/
├── main.go              # Main server and routing
├── api_client.go        # Client for Encore backend API
├── handlers/            # HTTP handlers
│   ├── home.go
│   ├── auth.go
│   ├── dashboard.go
│   └── types.go
├── views/               # Templ components
│   ├── layouts/         # Base layouts
│   │   └── base.templ
│   ├── components/      # Reusable components
│   │   ├── navbar.templ
│   │   ├── footer.templ
│   │   └── modal.templ
│   └── pages/           # Page components
│       ├── home.templ
│       ├── login.templ
│       ├── register.templ
│       └── dashboard.templ
└── static/              # Static assets (CSS, JS, images)
```

## Getting Started

### Prerequisites

- Go 1.23 or higher
- templ CLI: `go install github.com/a-h/templ/cmd/templ@latest`
- Encore backend running on `http://localhost:4000`

### Installation

1. Install dependencies:

```bash
cd frontend-go
go mod download
```

2. Generate templ files:

```bash
templ generate
```

3. Run the server:

```bash
go run .
```

The server will start on `http://localhost:8080`

### Development Workflow

When you modify `.templ` files:

1. Run `templ generate` to regenerate Go code
2. Restart the server

For auto-regeneration during development:
```bash
# Terminal 1: Watch for templ changes
templ generate --watch

# Terminal 2: Run the server
go run .
```

### Environment Variables

- `PORT` - Server port (default: 8080)
- `API_BASE_URL` - Encore backend URL (default: http://localhost:4000)

## How It Works

### Templ Components

Templ provides type-safe, compiled HTML templates:

```go
// Component definition
templ Dashboard(user *views.User) {
    @layouts.Base("Dashboard", user) {
        <div>Welcome { user.Email }</div>
    }
}

// Usage in handler
pages.Dashboard(user).Render(r.Context(), w)
```

Benefits:
- **Type Safety**: Props are type-checked at compile time
- **IDE Support**: Full autocomplete and refactoring
- **Performance**: Pre-compiled, no runtime parsing
- **Composition**: Easy to nest and reuse components

### HTMX Integration

This frontend uses HTMX to provide dynamic functionality without writing JavaScript:

- **Create Instance**: Form submission with `hx-post` triggers instance creation
- **List Instances**: Auto-loads on page with `hx-get` and `hx-trigger="load"`
- **Delete Instance**: Button with `hx-delete` and confirmation dialog
- **Loading States**: Automatic indicators with `hx-indicator`

## API Endpoints

### Pages

- `GET /` - Landing page
- `GET /login` - Login page
- `GET /register` - Register page
- `GET /dashboard` - Dashboard page (protected)

### Auth Endpoints

- `POST /auth/login` - User login
- `POST /auth/register` - User registration
- `POST /logout` - User logout

### HTMX API Endpoints

- `POST /api/instances` - Create new instance (protected)
- `GET /api/instances` - List all instances (protected)
- `DELETE /api/instances/{id}` - Delete instance (protected)

## Demo User

Email: `demo@instol.cloud`
Password: `demo123`

## Development

Run with auto-reload using Air:

```bash
go install github.com/cosmtrek/air@latest
air
```

## Deployment

Build for production:

```bash
go build -o n8n-frontend
./n8n-frontend
```

## License

MIT
