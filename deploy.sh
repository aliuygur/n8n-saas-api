#!/bin/bash#!/bin/bash



# Deploy n8n SaaS Infrastructure to Kubernetes# Deploy n8n in queue mode to Kubernetes

set -eset -e



echo "🚀 Deploying n8n SaaS Infrastructure..."echo "🚀 Deploying n8n in queue mode to Kubernetes..."



# Check if kubectl is available# Check if kubectl is available

if ! command -v kubectl &> /dev/null; thenif ! command -v kubectl &> /dev/null; then

    echo "❌ kubectl is not installed or not in PATH"    echo "❌ kubectl is not installed or not in PATH"

    exit 1    exit 1

fifi



# Check if we can connect to the cluster# Check if we can connect to the cluster

if ! kubectl cluster-info &> /dev/null; thenif ! kubectl cluster-info &> /dev/null; then

    echo "❌ Cannot connect to Kubernetes cluster"    echo "❌ Cannot connect to Kubernetes cluster"

    exit 1    exit 1

fifi



# Check for required environment variables# Apply manifests in order

if [ -z "$CLOUDFLARE_TUNNEL_TOKEN" ]; thenecho "📦 Creating namespace..."

    echo "❌ CLOUDFLARE_TUNNEL_TOKEN environment variable is required"kubectl apply -f k8s/namespace.yaml

    echo "   Get your token from Cloudflare Dashboard -> Zero Trust -> Networks -> Tunnels"

    exit 1echo "⚙️  Applying configuration..."

fikubectl apply -f k8s/configmap.yaml

kubectl apply -f k8s/secrets.yaml

echo "☁️  Deploying Cloudflare Tunnel..."

echo "💾 Creating persistent volumes..."

# Create tunnel secretkubectl apply -f k8s/pvc.yaml

kubectl create secret generic cloudflare-tunnel \

    --from-literal=token="$CLOUDFLARE_TUNNEL_TOKEN" \echo "🐘 Deploying PostgreSQL..."

    --dry-run=client -o yaml | kubectl apply -f -kubectl apply -f k8s/postgres.yaml



# Deploy tunnelecho "🔴 Deploying Redis..."

kubectl apply -f k8s/cloudflare-tunnel.yamlkubectl apply -f k8s/redis.yaml



# Wait for tunnel to be ready# Wait for dependencies to be ready

echo "⏳ Waiting for Cloudflare Tunnel to be ready..."echo "⏳ Waiting for PostgreSQL to be ready..."

kubectl wait --for=condition=ready pod -l app=cloudflare-tunnel --timeout=300skubectl wait --for=condition=ready pod -l app=postgres -n n8n --timeout=300s



echo "✅ Infrastructure deployment completed!"echo "⏳ Waiting for Redis to be ready..."

echo ""kubectl wait --for=condition=ready pod -l app=redis -n n8n --timeout=300s

echo "🎯 Next steps:"

echo "1. Configure your Go application environment variables:"echo "🎯 Deploying n8n main instance..."

echo "   export GKE_PROJECT_ID=your-project"kubectl apply -f k8s/n8n-main.yaml

echo "   export GKE_ZONE=your-zone"

echo "   export GKE_CLUSTER_NAME=your-cluster"echo "👷 Deploying n8n workers..."

echo ""kubectl apply -f k8s/n8n-worker.yaml

echo "2. Start the SaaS API server:"

echo "   encore run"echo "🌐 Setting up ingress..."

echo ""kubectl apply -f k8s/ingress.yaml

echo "3. Create n8n instances via API calls"

echo "   (All n8n resources are created programmatically)"echo "📈 Configuring auto-scaling..."

echo ""kubectl apply -f k8s/hpa.yaml

echo "4. For each customer domain, create DNS records at Cloudflare:"

echo "   CNAME: customer-domain.com -> tunnel-id.cfargotunnel.com"# Optional: Apply monitoring if Prometheus is available

echo ""if kubectl get crd servicemonitors.monitoring.coreos.com &> /dev/null; then

echo "📊 Monitor tunnel status:"    echo "📊 Setting up monitoring..."

echo "   kubectl logs -l app=cloudflare-tunnel -f"    kubectl apply -f k8s/monitoring.yaml
else
    echo "ℹ️  Skipping monitoring setup (Prometheus Operator not found)"
fi

echo "⏳ Waiting for n8n main instance to be ready..."
kubectl wait --for=condition=ready pod -l app=n8n-main -n n8n --timeout=300s

echo "✅ Deployment complete!"
echo ""
echo "📋 Deployment Status:"
kubectl get pods -n n8n
echo ""
echo "🔗 Services:"
kubectl get services -n n8n
echo ""
echo "🌍 Ingress:"
kubectl get ingress -n n8n

echo ""
echo "🎉 n8n is now running in queue mode!"
echo "👉 Access the UI at: http://$(kubectl get ingress n8n-ingress -n n8n -o jsonpath='{.spec.rules[0].host}')"
echo ""
echo "📊 Useful commands:"
echo "  View logs:     kubectl logs -f deployment/n8n-main -n n8n"
echo "  Scale workers: kubectl scale deployment n8n-worker -n n8n --replicas=X"
echo "  Port forward:  kubectl port-forward service/n8n-service -n n8n 8080:80"