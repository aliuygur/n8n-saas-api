# Ranx Kubernetes Configuration

## Structure

```
k8s/app/
├── cluster-rbac.yaml          # Cluster-wide RBAC (apply once, shared across environments)
├── base/                      # Base resources (namespaced)
│   ├── kustomization.yaml
│   └── app.yaml              # Deployment, Service, ServiceAccount, Namespace
└── overlays/
    ├── prod/                 # Production overlay
    │   ├── kustomization.yaml
    │   └── .env.prod
    └── stage/                # Staging overlay
        ├── kustomization.yaml
        └── .env.stage
```

## Deployment

### 1. Apply Cluster-Wide RBAC (once)

```bash
kubectl apply -f k8s/app/cluster-rbac.yaml
```

This creates a single ClusterRole and ClusterRoleBinding that grants permissions to both prod and stage service accounts.

### 2. Deploy to Production

```bash
kubectl apply -k k8s/app/overlays/prod
```

### 3. Deploy to Staging

```bash
kubectl apply -k k8s/app/overlays/stage
```

## Key Design Decisions

- **Cluster RBAC is separate**: ClusterRole and ClusterRoleBinding are not in the base kustomization to avoid namePrefix being applied
- **Single ClusterRoleBinding**: Both environments share the same ClusterRole and are bound via a single ClusterRoleBinding with multiple subjects
- **No duplication**: Each environment only needs to maintain its own secrets (.env files)
