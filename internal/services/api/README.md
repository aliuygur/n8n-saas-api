# API Service

The API service provides authentication and user management endpoints for the n8n-host frontend application.

## Features

- **Google OAuth Authentication**: Login with Google accounts
- **Session Management**: Secure session-based authentication
- **User Profile**: Retrieve current user information

## Endpoints

### Authentication

#### `GET /api/auth/google/login`
Initiates the Google OAuth flow.

**Response:**
```json
{
  "auth_url": "https://accounts.google.com/o/oauth2/auth?..."
}
```

#### `POST /api/auth/google/callback`
Handles the OAuth callback from Google.

**Request:**
```json
{
  "code": "oauth_code_from_google",
  "state": "state_token"
}
```

**Response:**
```json
{
  "session_token": "session_token_string",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "name": "John Doe",
    "picture": "https://..."
  },
  "expires_at": "2024-12-11T00:00:00Z"
}
```

#### `GET /api/auth/me`
Returns the current user's information.

**Headers:**
- `Authorization`: Session token

**Response:**
```json
{
  "user": {
    "id": 1,
    "email": "user@example.com",
    "name": "John Doe",
    "picture": "https://..."
  }
}
```

#### `POST /api/auth/logout`
Invalidates the current session.

**Headers:**
- `Authorization`: Session token

**Response:**
```json
{
  "success": true
}
```

### Health Check

#### `GET /api/health`
Returns the health status of the service.

**Response:**
```json
{
  "status": "ok",
  "service": "api"
}
```

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR NOT NULL,
    google_id VARCHAR NOT NULL,
    name VARCHAR NOT NULL,
    picture VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_login_at TIMESTAMP
);
```

### Sessions Table
```sql
CREATE TABLE sessions (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    token VARCHAR NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Configuration

The service requires the following configuration values (stored in Encore secrets or environment variables):

- `GoogleClientID`: Google OAuth client ID
- `GoogleClientSecret`: Google OAuth client secret
- `GoogleRedirectURL`: OAuth callback URL (e.g., `http://localhost:4000/api/auth/google/callback`)

### Setting up Google OAuth

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Google+ API
4. Go to "Credentials" and create OAuth 2.0 credentials
5. Add authorized redirect URIs:
   - Development: `http://localhost:4000/api/auth/google/callback`
   - Production: `https://yourdomain.com/api/auth/google/callback`
6. Copy the Client ID and Client Secret
7. Update the configuration in `service.go` or use Encore secrets

### Example: Setting Encore Secrets

```bash
encore secret set --type dev GoogleClientID
encore secret set --type dev GoogleClientSecret
encore secret set --type dev GoogleRedirectURL
```

## Usage with Frontend

1. **Initiate Login**: Call `GET /api/auth/google/login` to get the Google OAuth URL
2. **Redirect User**: Redirect the user to the `auth_url` returned
3. **Handle Callback**: Google redirects back to your frontend with a code
4. **Exchange Code**: Call `POST /api/auth/google/callback` with the code
5. **Store Token**: Save the `session_token` in localStorage or a cookie
6. **Authenticated Requests**: Include the token in the `Authorization` header for protected endpoints

## Development

### Running the Service

```bash
encore run
```

The API will be available at `http://localhost:4000`

### Database Migrations

Migrations are automatically applied by Encore when the service starts.

To create new migrations:
1. Add SQL files to `internal/services/api/migrations/`
2. Follow the naming convention: `{number}_{description}.up.sql` and `{number}_{description}.down.sql`

### Generating Database Code

After modifying SQL queries in `internal/db/queries/users.sql`:

```bash
sqlc generate
```

## Security Notes

- Session tokens are randomly generated 64-character strings
- Sessions expire after 7 days
- HTTPS should be used in production
- CSRF protection via state tokens in OAuth flow (TODO: implement state validation)
- Store Google Client Secret securely using Encore secrets
