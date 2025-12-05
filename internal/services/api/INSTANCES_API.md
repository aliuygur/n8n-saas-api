# Instance CRUD API Endpoints

This document describes the CRUD API endpoints for managing n8n instances in the API service.

## Overview

The API service exposes public endpoints that require authentication. These endpoints call the provisioning service internally to perform operations on n8n instances.

## Authentication

All endpoints require authentication via the session token obtained from the `/auth/google/callback` endpoint. Include the token in the `Authorization` header:

```
Authorization: <session_token>
```

## Endpoints

### 1. Create Instance

Create a new n8n instance for the authenticated user.

**Endpoint:** `POST /api/instances`

**Request Body:**
```json
{
  "subdomain": "my-instance"
}
```

**Response:**
```json
{
  "instance_id": 1,
  "status": "deploying",
  "domain": "https://my-instance.instol.cloud"
}
```

**Error Responses:**
- `400 Bad Request`: Invalid subdomain format or subdomain already taken
- `401 Unauthorized`: Invalid or missing session token
- `500 Internal Server Error`: Failed to create instance

**Example:**
```bash
curl -X POST http://localhost:8080/api/instances \
  -H "Authorization: <session_token>" \
  -H "Content-Type: application/json" \
  -d '{"subdomain": "my-instance"}'
```

### 2. Get Instance

Retrieve details of a specific instance by ID.

**Endpoint:** `GET /api/instances/:id`

**Path Parameters:**
- `id` (int): The instance ID

**Response:**
```json
{
  "id": 1,
  "status": "deployed",
  "domain": "https://my-instance.instol.cloud",
  "namespace": "n8n-user123-abc12345",
  "service_url": "n8n-main.n8n-user123-abc12345.svc.cluster.local",
  "created_at": "2025-12-04T10:30:00Z",
  "deployed_at": "2025-12-04T10:35:00Z",
  "details": "{\"cpu\":\"150m\",\"memory\":\"512Mi\"}"
}
```

**Error Responses:**
- `401 Unauthorized`: Invalid or missing session token
- `404 Not Found`: Instance not found
- `500 Internal Server Error`: Failed to retrieve instance

**Example:**
```bash
curl -X GET http://localhost:8080/api/instances/1 \
  -H "Authorization: <session_token>"
```

### 3. List Instances

Retrieve all instances for the authenticated user.

**Endpoint:** `GET /api/instances`

**Query Parameters:**
- `limit` (int, optional): Maximum number of instances to return (default: 50)
- `offset` (int, optional): Number of instances to skip (default: 0)

**Response:**
```json
{
  "instances": [
    {
      "id": 1,
      "status": "deployed",
      "domain": "https://my-instance.instol.cloud",
      "namespace": "n8n-user123-abc12345",
      "service_url": "n8n-main.n8n-user123-abc12345.svc.cluster.local",
      "created_at": "2025-12-04T10:30:00Z",
      "deployed_at": "2025-12-04T10:35:00Z"
    },
    {
      "id": 2,
      "status": "deploying",
      "domain": "https://another-instance.instol.cloud",
      "namespace": "n8n-user123-xyz98765",
      "service_url": "n8n-main.n8n-user123-xyz98765.svc.cluster.local",
      "created_at": "2025-12-04T11:00:00Z"
    }
  ]
}
```

**Error Responses:**
- `401 Unauthorized`: Invalid or missing session token
- `500 Internal Server Error`: Failed to retrieve instances

**Example:**
```bash
curl -X GET "http://localhost:8080/api/instances?limit=10&offset=0" \
  -H "Authorization: <session_token>"
```

### 4. Delete Instance

Delete an existing instance.

**Endpoint:** `DELETE /api/instances/:id`

**Path Parameters:**
- `id` (int): The instance ID

**Response:**
```json
{
  "message": "Instance successfully deleted"
}
```

**Error Responses:**
- `401 Unauthorized`: Invalid or missing session token
- `404 Not Found`: Instance not found
- `500 Internal Server Error`: Failed to delete instance

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/instances/1 \
  -H "Authorization: <session_token>"
```

## Status Values

Instances can have the following status values:

- `creating`: Instance record created, deployment starting
- `deploying`: GKE resources being created
- `deployed`: Instance fully deployed and accessible
- `failed`: Deployment or operation failed
- `deleting`: Instance being deleted
- `deleted`: Instance has been deleted

## Error Handling

All endpoints return appropriate HTTP status codes and error messages:

- `200 OK`: Successful request
- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Authentication required or invalid session
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server-side error

Error response format:
```json
{
  "code": "unauthenticated",
  "message": "invalid or expired session"
}
```

## Service Architecture

The API service acts as a gateway between the frontend and the provisioning service:

1. **API Service** (`/internal/services/api/instances.go`)
   - Handles HTTP requests
   - Validates authentication
   - Calls provisioning service
   - Formats responses for frontend

2. **Provisioning Service** (`/internal/services/provisioning/`)
   - Manages GKE deployments
   - Manages Cloudflare DNS
   - Interacts with database
   - Handles actual instance lifecycle

## Testing

You can test the endpoints using the Encore development UI at http://localhost:9400 when running `encore run`, or use tools like curl or Postman.

First, authenticate to get a session token:

```bash
# 1. Get Google login URL
curl http://localhost:8080/auth/google/login

# 2. Visit the auth_url in a browser and complete OAuth flow

# 3. Use the session_token from the callback response in subsequent requests
```

Then use the token to manage instances:

```bash
# Create an instance
curl -X POST http://localhost:8080/api/instances \
  -H "Authorization: <your_session_token>" \
  -H "Content-Type: application/json" \
  -d '{"subdomain": "test-instance"}'

# List instances
curl -X GET http://localhost:8080/api/instances \
  -H "Authorization: <your_session_token>"

# Get instance
curl -X GET http://localhost:8080/api/instances/1 \
  -H "Authorization: <your_session_token>"

# Delete instance
curl -X DELETE http://localhost:8080/api/instances/1 \
  -H "Authorization: <your_session_token>"
```
