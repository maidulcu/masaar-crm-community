# Kubernetes Deployment Guide

Deploy **Masaar CRM** on Kubernetes for enterprise-grade scalability and reliability.

## Prerequisites

- Kubernetes cluster (1.24+)
  - EKS, GKE, AKS, or self-hosted
  - Minimum 2 nodes with 4GB RAM each
- `kubectl` configured to access your cluster
- Docker registry access
- cert-manager (for SSL/TLS)
- nginx-ingress controller

## Quick Deployment

### 1. Build and Push Image

```bash
# Build Docker image
docker build -t your-registry/masaar-crm:v1 .

# Push to registry
docker push your-registry/masaar-crm:v1
```

### 2. Create Namespace

```bash
kubectl apply -f k8s/namespace.yaml

# Verify
kubectl get namespace masaar
```

### 3. Deploy Stack

```bash
# Deploy PostgreSQL
kubectl apply -f k8s/postgres.yaml

# Deploy Redis
kubectl apply -f k8s/redis.yaml

# Deploy Backend
kubectl apply -f k8s/backend.yaml

# Deploy Ingress
kubectl apply -f k8s/ingress.yaml

# Verify all pods running
kubectl get pods -n masaar
```

### 4. Configure DNS

```bash
# Get LoadBalancer IP
kubectl get svc -n ingress-nginx

# Update DNS: masaar.example.com -> LoadBalancer IP
# (In your DNS provider)

# Verify SSL certificate (wait 1-2 minutes)
kubectl get certificate -n masaar
kubectl describe certificate masaar-tls -n masaar
```

✅ **Application ready at:** https://masaar.example.com

---

## Architecture

```
┌─────────────────────────────────────────────┐
│         Internet / Load Balancer            │
└────────────────────┬────────────────────────┘
                     │
              ┌──────▼──────┐
              │   Ingress   │ (NGINX + Let's Encrypt)
              │  Controller │
              └──────┬──────┘
                     │
        ┌────────────┼────────────┐
        │            │            │
        ▼            ▼            ▼
    ┌────────┐  ┌────────┐  ┌─────────┐
    │ Masaar │  │ Masaar │  │Frontend │
    │ Pod 1  │  │ Pod 2  │  │ Pod 1   │
    │ (HPA)  │  │ (HPA)  │  │ (HPA)   │
    └────┬───┘  └────┬───┘  └────┬────┘
         │           │           │
         └─────┬─────┴──────┬────┘
               │            │
          ┌────▼────┐  ┌────▼────┐
          │PostgreSQL   │   Redis  │
          │StatefulSet  │  Deployment
          │(1 replica)  │  (1 replica)
          └────┬────┘  └────┬────┘
               │            │
          ┌────▼────┐  ┌────▼────┐
          │ PVC 50Gi│  │ PVC 10Gi│
          │Persistent  │Persistent
          │Volume      │Volume
          └─────────┘  └─────────┘
```

---

## Configuration

### Secrets Management

Update secrets before deploying:

```bash
# Generate secure JWT secret
openssl rand -hex 32

# Edit secrets
kubectl edit secret masaar-secret -n masaar

# Encoded secrets (base64):
echo -n "your-value" | base64
```

### ConfigMaps

Edit configuration:

```bash
kubectl edit configmap masaar-config -n masaar
```

---

## Operations

### View Status

```bash
# All resources
kubectl get all -n masaar

# Specific resources
kubectl get pods -n masaar
kubectl get svc -n masaar
kubectl get pvc -n masaar
kubectl get ingress -n masaar

# Detailed status
kubectl describe pod masaar-xxx -n masaar
```

### View Logs

```bash
# Backend logs
kubectl logs -f deployment/masaar -n masaar

# Frontend logs
kubectl logs -f deployment/frontend -n masaar

# PostgreSQL logs
kubectl logs -f statefulset/postgres -n masaar

# All logs
kubectl logs -f -n masaar --all-containers=true
```

### Execute Commands

```bash
# Shell into pod
kubectl exec -it pod/masaar-xxx -n masaar -- sh

# Database access
kubectl exec -it statefulset/postgres -n masaar -- \
  psql -U masaar -d masaar

# Redis access
kubectl exec -it deployment/redis -n masaar -- \
  redis-cli
```

### Scale Deployments

```bash
# Manual scaling
kubectl scale deployment masaar --replicas=5 -n masaar

# HPA (Horizontal Pod Autoscaler) is configured:
# - Min 2, Max 10 replicas
# - Scales on CPU (70%) and Memory (80%) usage
kubectl get hpa -n masaar
```

---

## Persistence

### PostgreSQL Backup

```bash
# Create backup
kubectl exec statefulset/postgres -n masaar -- \
  pg_dump -U masaar masaar > backup.sql

# Restore backup
kubectl exec -i statefulset/postgres -n masaar -- \
  psql -U masaar masaar < backup.sql
```

### PersistentVolume Snapshots

```bash
# Check volumes
kubectl get pvc -n masaar

# Snapshot PostgreSQL (requires SnapshotClass)
cat <<EOF | kubectl apply -f -
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: postgres-snapshot
  namespace: masaar
spec:
  volumeSnapshotClassName: csi-snapshot-class
  source:
    persistentVolumeClaimName: postgres-pvc
EOF
```

---

## Updates

### Rolling Deployment

```bash
# Update image
kubectl set image deployment/masaar \
  masaar=your-registry/masaar-crm:v2 -n masaar

# Monitor rollout
kubectl rollout status deployment/masaar -n masaar

# Rollback if needed
kubectl rollout undo deployment/masaar -n masaar
```

---

## Monitoring & Logging

### Install Prometheus (Optional)

```bash
# Add Prometheus Helm repo
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack -n monitoring

# Access: kubectl port-forward -n monitoring svc/prometheus 9090:9090
# Visit: http://localhost:9090
```

### Install ELK Stack (Optional)

```bash
# Elasticsearch + Logstash + Kibana for centralized logging
helm install elasticsearch elastic/elasticsearch -n logging
helm install kibana elastic/kibana -n logging
```

### Metrics

```bash
# View metrics
kubectl top nodes
kubectl top pods -n masaar
```

---

## Production Checklist

- [ ] Use strong passwords in secrets
- [ ] Configure SSL/TLS certificates
- [ ] Set resource limits and requests
- [ ] Enable HPA (already configured)
- [ ] Configure Pod Disruption Budgets
- [ ] Set up monitoring and alerting
- [ ] Configure backup and restore procedures
- [ ] Test disaster recovery
- [ ] Configure RBAC for access control
- [ ] Enable network policies
- [ ] Configure resource quotas
- [ ] Set up log aggregation
- [ ] Enable audit logging
- [ ] Configure pod security policies

---

## Helm Charts

A Helm chart is available for simplified deployment:

```bash
# Add Masaar repo
helm repo add masaar https://charts.masaar.io
helm repo update

# Install
helm install masaar masaar/masaar-crm \
  --namespace masaar \
  --create-namespace \
  --values values.yaml

# Upgrade
helm upgrade masaar masaar/masaar-crm \
  --namespace masaar \
  --values values.yaml

# Uninstall
helm uninstall masaar -n masaar
```

---

## Troubleshooting

### Pod Stuck in Pending

```bash
# Check events
kubectl describe pod masaar-xxx -n masaar

# Check node capacity
kubectl top nodes
kubectl describe nodes
```

### Database Connection Failed

```bash
# Check PostgreSQL pod
kubectl get pod -n masaar | grep postgres
kubectl logs statefulset/postgres -n masaar

# Test connection
kubectl exec -it deployment/masaar -n masaar -- \
  psql -h postgres -U masaar -d masaar -c "SELECT NOW();"
```

### SSL Certificate Not Renewing

```bash
# Check cert-manager logs
kubectl logs -n cert-manager deployment/cert-manager

# Describe certificate
kubectl describe certificate masaar-tls -n masaar

# Force renewal
kubectl delete certificate masaar-tls -n masaar
```

---

## Performance Tuning

### Resource Requests/Limits

Adjust in `backend.yaml` based on traffic:

```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "2Gi"
    cpu: "2000m"
```

### Database Connection Pooling

Configure in environment variables:
- PostgreSQL max_connections
- Redis client pools
- Connection timeouts

### Horizontal Scaling

Current HPA targets:
- CPU: 70% utilization
- Memory: 80% utilization
- Min 2 replicas, Max 10 replicas

Adjust in `backend.yaml` to match your workload.

---

## Support

- **Kubernetes Docs:** https://kubernetes.io
- **Helm Docs:** https://helm.sh
- **Masaar Issues:** https://github.com/maidulcu/masaar-crm/issues
