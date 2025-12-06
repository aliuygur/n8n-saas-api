# JWT Authentication Implementation

The authentication system has been updated to use JWT tokens instead of session-based cookies.

## Changes Made

### Backend (API Service)

#### 1. **Service Configuration** (`service.go`)
- Added `jwtSecret` field to the Service struct
- JWT secret is loaded from config (currently hardcoded, should use Encore secrets in production)

#### 2. **Authentication Handler** (`auth.go`)
- Updated `AuthHandler` to validate JWT tokens from `Authorization` header
- Expects format: `Authorization: Bearer <jwt_token>`
- Parses and validates JWT claims (user_id, email, expiration)
- Returns user information from JWT claims

#### 3. **Google OAuth Callback** (`auth_google.go`)
- Generates JWT token after successful Google authentication
- JWT includes:
  - `user_id`: User's unique identifier
  - `email`: User's email address
  - `exp`: Token expiration (7 days)
  - `iat`: Issued at timestamp
  - `iss`: Issuer (instol.cloud)
- Redirects to frontend with token in URL: `/auth/callback?token=<jwt>`
- **No longer uses cookies**

#### 4. **Logout Endpoint** (`auth.go`)
- Changed from raw HTTP endpoint to standard Encore endpoint
- Returns simple success response
- Client is responsible for discarding the token

### Frontend (React Router)

#### 1. **Auth Utilities** (`app/utils/auth.ts`)
- `getToken()`: Retrieve JWT from localStorage
- `setToken()`: Store JWT in localStorage
- `removeToken()`: Remove JWT from localStorage
- `isAuthenticated()`: Check if user has a token
- `logout()`: Call logout API and clear token

#### 2. **Auth Callback Route** (`app/routes/auth.callback.tsx`)
- New route to handle OAuth redirect
- Extracts JWT token from URL query parameter
- Stores token in localStorage
- Redirects to dashboard

#### 3. **Protected Routes**
- **Dashboard** (`dashboard.tsx`):
  - Checks authentication on mount
  - Includes JWT in `Authorization` header for API calls
  - Shows logout button
  - Redirects to login if unauthorized (401)

- **Create Instance** (`create-instance.tsx`):
  - Checks authentication on mount
  - Includes JWT in `Authorization` header for API calls

## Authentication Flow

1. **User clicks "Login with Google"**
   - Frontend redirects to `/auth/google`

2. **Backend initiates OAuth flow**
   - Redirects to Google OAuth consent screen

3. **Google redirects back to callback**
   - Backend receives OAuth code at `/auth/google/callback`

4. **Backend processes authentication**
   - Exchanges code for user info
   - Creates/updates user in database
   - Generates JWT token with user claims
   - Redirects to frontend: `http://localhost:5173/auth/callback?token=<jwt>`

5. **Frontend processes token**
   - Auth callback route extracts token from URL
   - Stores token in localStorage
   - Redirects to dashboard

6. **Authenticated requests**
   - Frontend includes token in header: `Authorization: Bearer <jwt>`
   - Backend validates token and extracts user info
   - Request proceeds with authenticated user context

## API Request Format

All authenticated requests must include the JWT token:

```javascript
const token = localStorage.getItem('jwt_token');
const response = await fetch('http://localhost:4000/api/instances', {
  headers: {
    'Authorization': `Bearer ${token}`,
  },
});
```

## Security Considerations

### Current Implementation
- JWT secret is hardcoded (should use Encore secrets)
- Token stored in localStorage (vulnerable to XSS)
- No token refresh mechanism
- 7-day token expiration

### Production Recommendations
1. **Use Encore Secrets** for JWT secret
2. **Consider shorter expiration** (e.g., 1 hour) with refresh tokens
3. **Implement token refresh** mechanism
4. **Add HTTPS** for all production traffic
5. **Consider httpOnly cookies** as alternative (with CSRF protection)
6. **Add rate limiting** for auth endpoints
7. **Implement token revocation** if needed

## Migration from Session-Based Auth

### Removed
- Session cookie storage
- Database session table queries
- Session validation in auth handler

### Added
- JWT token generation and validation
- localStorage token management
- Authorization header support
- Auth callback route for token handling

## Testing

1. Start backend: `encore run`
2. Start frontend: `cd frontend-remix && npm run dev`
3. Navigate to `http://localhost:5173`
4. Click "Login with Google"
5. Complete OAuth flow
6. Verify redirect to `/auth/callback` with token
7. Verify redirect to `/dashboard` with token stored
8. Verify API calls include `Authorization` header
9. Test logout clears token and redirects to login

## Troubleshooting

### "No authorization header provided"
- Ensure token is stored in localStorage as `jwt_token`
- Check that fetch requests include `Authorization` header

### "Invalid or expired token"
- Token may have expired (7 days)
- Token may be malformed
- JWT secret mismatch between creation and validation

### "Unauthorized" (401) responses
- Token is missing or invalid
- Frontend should redirect to login
