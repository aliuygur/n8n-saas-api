# n8n SaaS Platform

A platform for deploying and managing n8n workflow automation instances on Google Cloud Platform (GCP).

## Architecture

The platform consists of the following components:

### Frontend (React Router)
- Located in `frontend-remix/`
- Modern React application using React Router v7
- Dark theme UI for instol.cloud branding
- Runs on `http://localhost:5173` in development

### Backend Services (Encore.dev)

#### API Service (`internal/services/api/`)
- Main backend API that the frontend communicates with
- Handles authentication (Google OAuth)
- Manages instance operations by calling other services
- Runs on `http://localhost:4000`

#### Provisioning Service (`internal/services/provisioning/`)
- Manages n8n instance lifecycle (create, delete, monitor)
- Integrates with Google Kubernetes Engine (GKE)
- Handles subdomain validation and DNS configuration

#### Subscription Service (`internal/services/subscription/`)
- Manages user subscriptions and trials
- Enforces instance limits based on subscription tier
- Handles subscription lifecycle and billing

## Getting Started

### Prerequisites
- Go 1.21+
- Node.js 18+
- Encore CLI (`curl -L https://encore.dev/install.sh | bash`)
- Google Cloud Platform account with GKE enabled

### Development Setup

1. **Start the Encore backend:**
   ```bash
   encore run
   ```
   This starts all backend services on `http://localhost:4000`

2. **Start the React Router frontend:**
   ```bash
   cd frontend-remix
   npm install
   npm run dev
   ```
   This starts the frontend on `http://localhost:5173`

3. **Access the application:**
   - Open `http://localhost:5173` in your browser
   - Login with Google
   - Create and manage n8n instances

## Project Structure

```
n8n-host/
├── frontend-remix/          # React Router frontend application
│   ├── app/
│   │   ├── routes/         # Page routes
│   │   ├── app.css         # Global styles
│   │   └── root.tsx        # Root component
│   └── package.json
│
├── internal/
│   ├── auth/               # Shared authentication utilities
│   ├── db/                 # Database queries (sqlc generated)
│   └── services/
│       ├── api/            # Main API service (NEW)
│       │   ├── auth.go
│       │   ├── auth_google.go
│       │   ├── instances.go
│       │   ├── service.go
│       │   └── migrations/
│       ├── provisioning/   # Instance provisioning service
│       └── subscription/   # Subscription management service
│
├── k8s/                    # Kubernetes manifests
├── scripts/                # Utility scripts
└── encore.app              # Encore configuration
```

## API Documentation

See [API Service README](internal/services/api/README.md) for detailed API documentation.

## Key Features

- **One-Click Deployment**: Deploy n8n instances on GKE with a single click
- **Automatic SSL**: Let's Encrypt certificates automatically provisioned
- **Google OAuth**: Secure authentication with Google accounts
- **Instance Management**: Create, monitor, and delete instances
- **Subscription Management**: Trial accounts and paid subscriptions
- **Dark Theme UI**: Modern, clean interface

## Database

The platform uses PostgreSQL databases managed by Encore:

- `api`: User accounts and sessions
- `provisioning`: Instance metadata
- `subscription`: Subscription and billing data

## Environment Configuration

Configuration is managed through Encore secrets. Set these values:

```bash
encore secret set GoogleClientID
encore secret set GoogleClientSecret
encore secret set GoogleRedirectURL
```

## Deployment

### Backend Deployment
```bash
# Deploy to Encore Cloud
encore deploy

# Or deploy to your own infrastructure
encore build docker
```

### Frontend Deployment
```bash
cd frontend-remix
npm run build
# Deploy the build/ directory to your static hosting service
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

[Your License Here]
