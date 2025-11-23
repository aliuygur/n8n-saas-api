#!/bin/bash

# N8N SaaS Deployment Script
# This script deploys Cloudflare tunnel for N8N SaaS platform

set -e

echo "🚀 Starting Cloudflare Tunnel Deployment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed or not in PATH"
    exit 1
fi

# Check if we can connect to the cluster
if ! kubectl cluster-info &> /dev/null; then
    print_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
    exit 1
fi

print_status "Connected to Kubernetes cluster:"
kubectl cluster-info --context=$(kubectl config current-context) | head -1

# Deploy Cloudflare Tunnel
print_status "Deploying Cloudflare Tunnel..."
# Check if Cloudflare tunnel token is provided
if [[ -z "$CLOUDFLARE_TUNNEL_TOKEN" ]]; then
    print_warning "CLOUDFLARE_TUNNEL_TOKEN environment variable not set."
    print_status "Skipping Cloudflare tunnel deployment."
    print_status "To deploy later, set the token and run:"
    echo "  kubectl create secret generic cloudflare-tunnel --from-literal=token=YOUR_TOKEN -n cloudflare-tunnel"
    echo "  kubectl apply -f cloudflare-tunnel.yaml"
else
    print_status "Creating Cloudflare tunnel secret..."
    kubectl create namespace cloudflare-tunnel --dry-run=client -o yaml | kubectl apply -f -
    kubectl create secret generic cloudflare-tunnel \
        --from-literal=token="$CLOUDFLARE_TUNNEL_TOKEN" \
        -n cloudflare-tunnel \
        --dry-run=client -o yaml | kubectl apply -f -
    
    kubectl apply -f cloudflare-tunnel.yaml
    print_success "Cloudflare tunnel deployment created"
    
    # Wait for tunnel to be ready
    print_status "Waiting for Cloudflare tunnel to be ready..."
    kubectl wait --for=condition=Available deployment/cloudflare-tunnel -n cloudflare-tunnel --timeout=120s
    
    # Display deployment status
    print_status "Checking Cloudflare tunnel status..."
    echo ""
    echo "=== CLOUDFLARE TUNNEL STATUS ==="
    kubectl get all -n cloudflare-tunnel
fi

print_success "✅ N8N SaaS infrastructure deployment completed!"

echo ""
echo "📋 Architecture Overview:"
echo "• N8N SaaS uses standard mode (not queue mode)"
echo "• Each customer gets their own isolated N8N instance"
echo "• Each instance uses SQLite for complete data isolation"
echo "• No shared PostgreSQL or Redis needed"
echo "• Each customer instance deployed in separate namespace"
echo ""
echo "🚀 To deploy customer instances:"
echo "Use the Go API to deploy individual N8N instances using n8n-instance.yaml template"
echo ""
echo "📊 Deployed Components:"
if [[ -n "$CLOUDFLARE_TUNNEL_TOKEN" ]]; then
    echo "   ✅ Cloudflare Tunnel: cloudflare-tunnel namespace"
else
    echo "   ⏸️  Cloudflare Tunnel: Ready to deploy (token needed)"
fi