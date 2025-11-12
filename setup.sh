#!/bin/bash

# Setup script for n8n SaaS API development environment

set -e

echo "🚀 Setting up n8n SaaS API development environment..."

# Check prerequisites
echo "📋 Checking prerequisites..."

if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21 or later."
    exit 1
fi

if ! command -v encore &> /dev/null; then
    echo "📦 Installing Encore CLI..."
    go install encr.dev/cmd/encore@latest
fi

if ! command -v sqlc &> /dev/null; then
    echo "📦 Installing SQLC..."
    go install github.com/kyleconroy/sqlc/cmd/sqlc@latest
fi

if ! command -v psql &> /dev/null; then
    echo "⚠️  PostgreSQL client not found. Please install PostgreSQL."
fi

# Setup environment
echo "⚙️  Setting up environment..."

if [ ! -f .env ]; then
    cp .env.example .env
    echo "📝 Created .env file. Please update it with your configuration."
fi

# Generate database code
echo "🗄️  Generating database code..."
if command -v sqlc &> /dev/null; then
    sqlc generate
    echo "✅ Database code generated successfully"
else
    echo "⚠️  SQLC not found, skipping code generation"
fi

# Initialize Encore database
echo "🗄️  Setting up database..."
if command -v encore &> /dev/null; then
    encore db migrate
    echo "✅ Database migrations applied"
else
    echo "⚠️  Encore CLI not found, skipping database setup"
fi

# Install Go dependencies
echo "📦 Installing Go dependencies..."
go mod tidy

echo ""
echo "✅ Setup complete!"
echo ""
echo "🚀 To start the development server:"
echo "   encore run"
echo ""
echo "📚 API documentation will be available at:"
echo "   http://localhost:9400"
echo ""
echo "🔧 Before running, make sure to:"
echo "   1. Update .env with your GCP credentials"
echo "   2. Create a GKE Autopilot cluster"
echo "   3. Set up service account permissions"