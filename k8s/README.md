# N8N SaaS Platform - Kubernetes Deployment

This directory contains the Kubernetes manifests for deploying the N8N SaaS platform with complete customer isolation.

## Architecture Overview

The new simplified architecture provides complete isolation between customers:

- **Per-Customer N8N**: Each customer gets completely isolated N8N instance with SQLite database
- **No Shared Infrastructure**: No shared PostgreSQL, Redis, or workers between customers
- **Standard Mode**: N8N runs in standard mode (not queue mode) for simplicity
- **Cloudflare Tunnel**: Single tunnel for routing to all customer instances
- **Complete Isolation**: Each customer namespace is fully independent

## Deployment Files

### Infrastructure Components

- `n8n-instance.yaml` - N8N instance template with SQLite (used per customer)
- `cloudflare-tunnel.yaml` - Cloudflare tunnel for external access
- `deploy-infrastructure.sh` - Simple deployment script for Cloudflare tunnel

## Quick Deployment

```bash
# Deploy infrastructure (only Cloudflare tunnel)
export CLOUDFLARE_TUNNEL_TOKEN="your-token-here"
./deploy-infrastructure.sh
```

## Manual Deployment

```bash
# 1. Deploy Cloudflare tunnel (optional, when you have token)
kubectl create namespace cloudflare-tunnel
kubectl create secret generic cloudflare-tunnel --from-literal=token=YOUR_TOKEN -n cloudflare-tunnel
kubectl apply -f cloudflare-tunnel.yaml
```

## Configuration

### Secrets to Update

Before deployment, update these secrets with your actual values:

````bash
# PostgreSQL admin password
kubectl patch secret postgresql-secret -n n8n-postgresql -p='{"data":{"postgres-password":"YOUR_BASE64_PASSWORD"}}'

## Customer Instance Deployment

Customer N8N instances are deployed programmatically via the Go API using the `n8n-instance.yaml` template.

### Template Variables

The `n8n-instance.yaml` template uses these variables:

- `{{NAMESPACE}}` - Customer namespace (e.g., `n8n-customer-123`)
- `{{ENCRYPTION_KEY}}` - N8N encryption key for the customer
- `{{SUBDOMAIN}}` - Customer subdomain (e.g., `customer123`)
- `{{DOMAIN}}` - Your base domain (e.g., `yourdomain.com`)
- `{{IP_NAME}}` - GCP static IP name for the customer

### Example Customer Deployment

```bash
# Deploy customer instance (normally done via API)
export NAMESPACE="n8n-customer-123"
export ENCRYPTION_KEY="your-encryption-key"
export SUBDOMAIN="customer123"
export DOMAIN="yourdomain.com"
export IP_NAME="customer123-ip"

# Replace template variables and apply
envsubst < n8n-instance.yaml | kubectl apply -f -
````

## Monitoring and Scaling

### Check Status

```bash
# Cloudflare tunnel status (if deployed)
kubectl get all -n cloudflare-tunnel

# Customer instance status
kubectl get all -n n8n-customer-123

# All customer namespaces
kubectl get namespaces | grep n8n-customer
kubectl describe deployment redis -n n8n-redis
kubectl describe deployment n8n-worker -n n8n-worker

# Resource usage across all namespaces
kubectl top pods -A | grep -E "(n8n-postgresql|n8n-redis|n8n-worker|cloudflare-tunnel)"
```

### View Logs

```bash
# Customer instance logs
kubectl logs -f deployment/n8n -n n8n-customer-123

# Cloudflare tunnel logs (if deployed)
kubectl logs -f deployment/cloudflare-tunnel -n cloudflare-tunnel
```

## Storage

### Customer N8N Storage

Each customer instance gets:

- **Size**: 1Gi persistent volume for SQLite database and N8N data
- **Class**: premium-rwo (GKE SSD persistent disk)
- **Access**: ReadWriteOnce
- **Location**: `/home/node/.n8n` (contains SQLite database and workflows)

## Security

### Complete Isolation

- Each customer in separate namespace
- SQLite database per customer (no shared data)
- Dedicated ingress and SSL certificate per customer
- No cross-customer communication possible

### Container Security

- Non-root user execution
- Resource limits enforced
- Health checks for reliability

## Customer Management

### Deploy New Customer

```bash
# Example deployment via Go API
POST /provisioning/customers
{
  "name": "Customer Name",
  "subdomain": "customer123"
}
```

The API will:

1. Create customer record in SaaS platform database
2. Generate encryption key
3. Create customer namespace
4. Deploy N8N instance using template
5. Create DNS records

### Remove Customer

```bash
# Delete entire customer namespace (removes everything)
kubectl delete namespace n8n-customer-123
```

## Troubleshooting

### Common Issues

1. **N8N instance connection issues**

   ```bash
   kubectl exec -it deployment/n8n -n n8n-customer-123 -- ls -la /home/node/.n8n/
   ```

2. **SQLite database issues**

   ```bash
   kubectl exec -it deployment/n8n -n n8n-customer-123 -- sqlite3 /home/node/.n8n/database.sqlite ".tables"
   ```

3. **Storage issues**

   ```bash
   kubectl get pvc -n n8n-customer-123
   kubectl describe pvc n8n-data -n n8n-customer-123
   ```

4. **SSL certificate issues**

   ```bash
   kubectl get managedcertificate -n n8n-customer-123
   kubectl describe managedcertificate customer123-ssl-cert -n n8n-customer-123
   ```

### Resource Limits

Each N8N instance:

- **CPU**: 100m request, 500m limit
- **Memory**: 256Mi request, 512Mi limit
- **Storage**: 1Gi persistent volume

## Architecture Benefits

This simplified architecture provides:

1. **Complete Isolation**: No shared resources, no data mixing risks
2. **Simple Deployment**: Single template for all customers
3. **Easy Scaling**: Each customer scales independently
4. **Cost Effective**: Minimal resource usage per customer
5. **Reliable**: Customer issues don't affect others

Benefits over shared infrastructure:

- **Perfect Security**: Complete data isolation with SQLite
- **Simplified Operations**: No complex shared infrastructure to manage
- **Better Reliability**: Customer failures don't cascade
- **Easier Debugging**: Each customer instance is independent
