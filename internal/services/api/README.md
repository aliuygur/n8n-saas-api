# API Service

This is the backend API service for the n8n-host platform. It acts as an intermediary between the React Router frontend and other backend services like provisioning and subscription.

## Architecture

The API service provides RESTful endpoints for:

- **Authentication**: Google OAuth login, logout, session management
- **Instance Management**: Create, list, get, and delete n8n instances
- **User Management**: Get current user information

## Endpoints

### Authentication

- `GET /auth/google` - Initiate Google OAuth login flow
- `GET /auth/google/callback` - Handle Google OAuth callback
- `POST /api/auth/logout` - Logout user and clear session
- `GET /api/auth/me` - Get current authenticated user (requires auth)

### Instance Management

- `POST /api/instances` - Create a new n8n instance (requires auth)
  - Request: `{ "subdomain": "myworkflow" }`
  - Response: `{ "id": "...", "subdomain": "...", "domain": "...", "status": "...", "created_at": "..." }`

- `GET /api/instances` - List all instances for the authenticated user (requires auth)
  - Response: `{ "instances": [...] }`

- `GET /api/instances/:id` - Get a specific instance (requires auth)
  - Response: `{ "id": "...", "subdomain": "...", "domain": "...", "status": "...", "created_at": "..." }`

- `DELETE /api/instances/:id` - Delete an instance (requires auth)
  - Response: `{ "success": true }`

## Service Communication

The API service communicates with:

- **Provisioning Service**: For creating, listing, and deleting n8n instances
- **Subscription Service**: For managing user subscriptions, trials, and instance limits

## Database

The API service has its own database (`api`) with the following tables:

- `users`: User accounts from Google OAuth
- `sessions`: User session tokens for authentication

## Authentication Flow

1. User clicks "Login with Google" on frontend (React Router)
2. Frontend redirects to `/auth/google`
3. API service redirects to Google OAuth
4. Google redirects back to `/auth/google/callback`
5. API service:
   - Exchanges code for user info
   - Creates/updates user in database
   - Creates session token
   - Sets session cookie
   - Redirects to frontend dashboard

6. Frontend makes authenticated requests with session cookie
7. API service validates session via `AuthHandler`
8. Authenticated requests can access protected endpoints

## CORS Configuration

The API is configured to accept requests from the React Router frontend running on `http://localhost:5173` (Vite dev server).

## Environment Variables

Configuration is loaded from Encore secrets (TODO):
- `GoogleClientID`: Google OAuth client ID
- `GoogleClientSecret`: Google OAuth client secret
- `GoogleRedirectURL`: OAuth callback URL

## Running the Service

```bash
# Start Encore services (from project root)
encore run

# The API service will be available at http://localhost:4000
```

## Frontend Integration

The React Router frontend (in `frontend-remix/`) makes requests to this API service. Update the base URL in the frontend to point to the correct port (4000 for Encore services).

## Development

When developing:
1. Run the Encore backend: `encore run`
2. Run the React Router frontend: `cd frontend-remix && npm run dev`
3. Frontend will be at `http://localhost:5173`
4. Backend API will be at `http://localhost:4000`
