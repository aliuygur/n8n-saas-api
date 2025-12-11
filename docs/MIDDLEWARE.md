# Authentication Middleware

The application uses context-based authentication middleware for clean and consistent auth handling.

## Overview

Instead of manually checking authentication in every handler, we use middleware that:
1. Validates JWT tokens
2. Adds user information to request context
3. Handles unauthorized access appropriately

## Middleware Types

### 1. `requireAuth` - For Frontend Pages
Redirects unauthenticated users to `/login`:

```go
mux.HandleFunc("GET /dashboard", h.requireAuth(h.Dashboard))
```

### 2. `requireAuthAPI` - For API Endpoints
Returns `401 Unauthorized` for API endpoints:

```go
mux.HandleFunc("POST /api/create-instance", h.requireAuthAPI(h.CreateInstance))
```

### 3. `AuthMiddleware` - Standalone Middleware
Can be used with `http.Handler` for route groups:

```go
protected := http.NewServeMux()
protected.HandleFunc("/admin", h.AdminPanel)
mux.Handle("/admin/", h.AuthMiddleware(protected))
```

## Context Functions

### `MustGetUser(ctx)` - Retrieve User from Context
Use in handlers wrapped with auth middleware:

```go
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
    user := MustGetUser(r.Context())
    // user is guaranteed to exist
    instances, _ := h.listInstances(user.UserID)
}
```

**Panics if user not in context** - only use after auth middleware!

### `GetUser(ctx)` - Optional User Retrieval
Use when authentication is optional:

```go
func (h *Handler) OptionalFeature(w http.ResponseWriter, r *http.Request) {
    user := GetUser(r.Context())
    if user != nil {
        // Show personalized content
    } else {
        // Show generic content
    }
}
```

Returns `nil` if user not authenticated.

## Benefits

### Before (Without Middleware)
```go
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
    // Repetitive auth check in every handler
    user, err := h.GetUserFromRequest(r)
    if err != nil {
        http.Redirect(w, r, "/login", http.StatusSeeOther)
        return
    }

    // Handler logic...
}
```

### After (With Middleware)
```go
// Route registration
mux.HandleFunc("GET /dashboard", h.requireAuth(h.Dashboard))

// Handler is clean
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
    user := MustGetUser(r.Context())
    // Handler logic...
}
```

**Advantages:**
- ✅ **DRY**: No repetitive auth checks
- ✅ **Centralized**: Auth logic in one place
- ✅ **Type-safe**: Context key is unexported
- ✅ **Clear**: Route registration shows auth requirements
- ✅ **Testable**: Easy to mock context with user

## Route Organization

Routes are now organized by authentication requirement in [internal/handler/routes.go](../internal/handler/routes.go):

```go
// Public routes (no auth required)
mux.HandleFunc("GET /", h.Home)
mux.HandleFunc("GET /login", h.Login)

// Auth required - Frontend pages (redirects to login)
mux.HandleFunc("GET /dashboard", h.requireAuth(h.Dashboard))
mux.HandleFunc("GET /create-instance", h.requireAuth(h.CreateInstancePage))

// Auth required - API endpoints (returns 401)
mux.HandleFunc("POST /api/create-instance", h.requireAuthAPI(h.CreateInstance))
mux.HandleFunc("GET /api/auth/me", h.requireAuthAPI(h.GetAuthMe))

// Public webhooks (no auth)
mux.HandleFunc("POST /api/webhooks/polar", h.PolarWebhook)
```

## Security Notes

1. **Context Key**: Uses unexported `contextKey` type to prevent key collisions
2. **Panic Safety**: `MustGetUser` only panics in development - middleware always sets user
3. **JWT Validation**: Token signature and expiration validated in `GetUserFromRequest`
4. **Cookie Security**: JWT stored in HTTP-only, secure, SameSite cookies

## Testing

### Mock User in Tests
```go
func TestDashboard(t *testing.T) {
    req := httptest.NewRequest("GET", "/dashboard", nil)

    // Add mock user to context
    user := &JWTClaims{UserID: "test-123", Email: "test@example.com"}
    ctx := context.WithValue(req.Context(), userContextKey, user)
    req = req.WithContext(ctx)

    // Test handler
    rec := httptest.NewRecorder()
    handler.Dashboard(rec, req)

    // Assertions...
}
```

## Migration Notes

The following handlers were updated to use middleware:

**Frontend Pages:**
- `Dashboard` - Uses `MustGetUser(r.Context())`
- `CreateInstancePage` - Auth check removed (middleware handles)
- `ProvisioningPage` - Auth check removed (middleware handles)

**API Endpoints:**
- `GetAuthMe` - Uses `MustGetUser(r.Context())`
- `CreateInstance` - Uses `MustGetUser(r.Context())`
- `DeleteInstance` - Uses `MustGetUser(r.Context())`
- `CheckSubdomain` - Auth check removed (middleware handles)
- `GetProvisioningStatus` - Auth check removed (middleware handles)
- `DeleteModal` - Auth check removed (middleware handles)

All handlers now benefit from cleaner code and centralized authentication!
