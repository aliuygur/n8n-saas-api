# Deploy n8n in queue mode to Kubernetes

This directory contains Kubernetes manifests for deploying n8n in queue mode with the following components:

## Components

- **Main n8n instance**: Handles the UI and webhook endpoints
- **Worker nodes**: Process workflow executions from the queue
- **PostgreSQL**: Database for workflow data and execution history
- **Redis**: Queue backend for job management
- **Ingress**: External access configuration
- **HPA**: Horizontal Pod Autoscaler for worker scaling

## Quick Start

1. **Update configuration**:

   - Edit `configmap.yaml` to set your domain and environment variables
   - Update `secrets.yaml` with your database password and encryption key
   - Modify `ingress.yaml` to use your domain name

2. **Generate secrets** (recommended):

   ```bash
   # Generate a secure encryption key
   openssl rand -base64 32 | base64

   # Generate a secure database password
   openssl rand -base64 16 | base64
   ```

3. **Deploy Cloudflare Tunnel** (one-time setup):

   ```bash
   # Create tunnel token secret
   kubectl create secret generic cloudflare-tunnel \
     --from-literal=token=YOUR_TUNNEL_TOKEN

   # Deploy tunnel
   kubectl apply -f k8s/cloudflare-tunnel.yaml
   ```

4. **Run the SaaS API Server**:

   ```bash
   # Configure environment
   export GKE_PROJECT_ID=your-project
   export GKE_ZONE=us-central1-a
   export GKE_CLUSTER_NAME=n8n-cluster

   # Start the server
   encore run
   ```

5. **Create n8n instances via API**:

   ```bash
   # All n8n resources are created programmatically via Go client
   # No manual YAML deployment needed

   # Check deployment status for a customer
   kubectl get pods -n customer-{namespace}
   kubectl get services -n customer-{namespace}
   ```

## Configuration

### Environment Variables

Key configuration options in `configmap.yaml`:

- `N8N_HOST`: Your domain name
- `WEBHOOK_URL`: External webhook URL
- `EXECUTIONS_MODE`: Set to "queue" for queue mode

### Scaling

- Main instance: Single replica (stateful)
- Workers: Auto-scaling from 2-10 replicas based on CPU/memory usage
- Adjust HPA settings in `hpa.yaml` based on your needs

### Storage

- PostgreSQL: 10Gi persistent volume
- n8n data: 5Gi persistent volume
- Redis: Ephemeral storage (data in memory)

### Security

- Database credentials stored in Kubernetes secrets
- Encryption key for n8n data protection
- Network policies can be added for additional security

## Monitoring

If you have Prometheus Operator installed, the `monitoring.yaml` file includes ServiceMonitor resources for metrics collection.

## Customization

### Storage Classes

Update the `storageClassName` in `pvc.yaml` to match your cluster's available storage classes.

### Ingress Controller

Modify the ingress annotations in `ingress.yaml` based on your ingress controller (nginx, traefik, etc.).

### Resource Limits

Adjust CPU and memory limits in the deployment files based on your workload requirements.

### TLS/SSL

Uncomment and configure the TLS section in `ingress.yaml` if you want HTTPS access.

## Troubleshooting

1. **Check pod logs**:

   ```bash
   kubectl logs -n n8n deployment/n8n-main
   kubectl logs -n n8n deployment/n8n-worker
   ```

2. **Check service connectivity**:

   ```bash
   kubectl exec -n n8n deployment/n8n-main -- nc -zv postgres-service 5432
   kubectl exec -n n8n deployment/n8n-main -- nc -zv redis-service 6379
   ```

3. **Scale workers manually**:
   ```bash
   kubectl scale deployment n8n-worker -n n8n --replicas=5
   ```

## Production Considerations

- Use external managed databases (RDS, Cloud SQL) for production
- Implement proper backup strategies for PostgreSQL
- Configure resource quotas and limits
- Set up monitoring and alerting
- Use network policies for security
- Consider using Redis Cluster for high availability
- Implement proper logging aggregation
