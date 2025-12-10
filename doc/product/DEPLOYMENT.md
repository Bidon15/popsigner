# Deployment Guide

This document provides instructions for deploying the BanhBao OpenBao infrastructure on Kubernetes.

---

## 1. Overview

### 1.1 Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Kubernetes Cluster                                │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Ingress Controller                           │   │
│  │                    (nginx / traefik / istio)                        │   │
│  └────────────────────────────┬────────────────────────────────────────┘   │
│                               │                                             │
│          ┌────────────────────┼────────────────────┐                       │
│          │                    │                    │                       │
│          ▼                    ▼                    ▼                       │
│  ┌───────────────┐   ┌───────────────┐   ┌───────────────────────────┐    │
│  │   OpenBao     │   │   OpenBao     │   │   OpenBao                 │    │
│  │   Node 1      │   │   Node 2      │   │   Node 3                  │    │
│  │   (active)    │   │   (standby)   │   │   (standby)               │    │
│  └───────┬───────┘   └───────┬───────┘   └───────────┬───────────────┘    │
│          │                   │                       │                     │
│          └───────────────────┼───────────────────────┘                     │
│                              │                                             │
│                              ▼                                             │
│                    ┌─────────────────┐                                     │
│                    │   Raft Storage  │                                     │
│                    │   (PVC)         │                                     │
│                    └─────────────────┘                                     │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                         Monitoring                                   │   │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                  │   │
│  │  │ Prometheus  │  │  Grafana    │  │ AlertManager│                  │   │
│  │  └─────────────┘  └─────────────┘  └─────────────┘                  │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 Components

| Component | Purpose | Replicas |
| --------- | ------- | -------- |
| OpenBao Server | Transit engine for key management | 3 (HA) |
| Ingress | TLS termination, routing | 2+ |
| Raft Storage | Integrated storage backend | Per node |
| Prometheus | Metrics collection | 1 |
| Grafana | Dashboards | 1 |

---

## 2. Prerequisites

### 2.1 Infrastructure Requirements

| Resource | Minimum | Recommended |
| -------- | ------- | ----------- |
| Kubernetes version | 1.25+ | 1.28+ |
| Nodes | 3 | 5+ |
| CPU per OpenBao pod | 500m | 2 cores |
| Memory per OpenBao pod | 512Mi | 2Gi |
| Storage per node | 10Gi | 50Gi SSD |

### 2.2 Required Tools

```bash
# kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl && sudo mv kubectl /usr/local/bin/

# helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# openbao CLI
wget https://github.com/openbao/openbao/releases/download/v2.0.0/bao_2.0.0_linux_amd64.zip
unzip bao_2.0.0_linux_amd64.zip
sudo mv bao /usr/local/bin/
```

### 2.3 DNS & Certificates

- Domain name for OpenBao (e.g., `bao.yourdomain.com`)
- TLS certificate (Let's Encrypt or custom CA)
- Wildcard certificate recommended for HA setup

---

## 3. Kubernetes Cluster Setup

### 3.1 Option A: Cloud Provider Managed Kubernetes

#### AWS EKS

```bash
# Install eksctl
curl --silent --location "https://github.com/weaveworks/eksctl/releases/latest/download/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp
sudo mv /tmp/eksctl /usr/local/bin

# Create cluster
eksctl create cluster \
  --name banhbao-cluster \
  --region us-east-1 \
  --nodegroup-name standard-workers \
  --node-type t3.medium \
  --nodes 3 \
  --nodes-min 3 \
  --nodes-max 5 \
  --managed

# Verify
kubectl get nodes
```

#### GCP GKE

```bash
# Create cluster
gcloud container clusters create banhbao-cluster \
  --zone us-central1-a \
  --num-nodes 3 \
  --machine-type e2-medium \
  --enable-autorepair \
  --enable-autoupgrade

# Get credentials
gcloud container clusters get-credentials banhbao-cluster --zone us-central1-a

# Verify
kubectl get nodes
```

#### Azure AKS

```bash
# Create resource group
az group create --name banhbao-rg --location eastus

# Create cluster
az aks create \
  --resource-group banhbao-rg \
  --name banhbao-cluster \
  --node-count 3 \
  --node-vm-size Standard_D2s_v3 \
  --enable-managed-identity \
  --generate-ssh-keys

# Get credentials
az aks get-credentials --resource-group banhbao-rg --name banhbao-cluster

# Verify
kubectl get nodes
```

### 3.2 Option B: Self-Hosted Kubernetes

#### Using kubeadm

```bash
# On control plane node
sudo kubeadm init --pod-network-cidr=10.244.0.0/16

# Set up kubectl
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

# Install CNI (Flannel)
kubectl apply -f https://raw.githubusercontent.com/flannel-io/flannel/master/Documentation/kube-flannel.yml

# Join worker nodes (run on each worker)
# Use the kubeadm join command from init output
```

#### Using k3s (Lightweight)

```bash
# On first node (server)
curl -sfL https://get.k3s.io | sh -s - --cluster-init

# Get token
sudo cat /var/lib/rancher/k3s/server/node-token

# On additional nodes
curl -sfL https://get.k3s.io | K3S_URL=https://first-node:6443 K3S_TOKEN=<token> sh -

# Configure kubectl
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
```

---

## 4. OpenBao Deployment

### 4.1 Namespace Setup

```bash
# Create namespace
kubectl create namespace banhbao

# Set as default context
kubectl config set-context --current --namespace=banhbao
```

### 4.2 Helm Chart Installation

```bash
# Add OpenBao Helm repository
helm repo add openbao https://openbao.github.io/openbao-helm
helm repo update

# Create values file
cat > openbao-values.yaml << 'EOF'
global:
  enabled: true
  tlsDisable: false

server:
  enabled: true
  image:
    repository: quay.io/openbao/openbao
    tag: "2.0.0"
  
  # Resource limits
  resources:
    requests:
      memory: 512Mi
      cpu: 500m
    limits:
      memory: 2Gi
      cpu: 2000m
  
  # High Availability
  ha:
    enabled: true
    replicas: 3
    raft:
      enabled: true
      setNodeId: true
      config: |
        ui = true
        listener "tcp" {
          tls_disable = 0
          address = "[::]:8200"
          cluster_address = "[::]:8201"
          tls_cert_file = "/vault/userconfig/tls/tls.crt"
          tls_key_file = "/vault/userconfig/tls/tls.key"
          tls_client_ca_file = "/vault/userconfig/tls/ca.crt"
        }
        storage "raft" {
          path = "/vault/data"
          retry_join {
            leader_api_addr = "https://openbao-0.openbao-internal:8200"
            leader_ca_cert_file = "/vault/userconfig/tls/ca.crt"
          }
          retry_join {
            leader_api_addr = "https://openbao-1.openbao-internal:8200"
            leader_ca_cert_file = "/vault/userconfig/tls/ca.crt"
          }
          retry_join {
            leader_api_addr = "https://openbao-2.openbao-internal:8200"
            leader_ca_cert_file = "/vault/userconfig/tls/ca.crt"
          }
        }
        service_registration "kubernetes" {}
        telemetry {
          prometheus_retention_time = "30s"
          disable_hostname = true
        }
  
  # Persistent storage
  dataStorage:
    enabled: true
    size: 10Gi
    storageClass: null  # Use default storage class
  
  # Audit logging
  auditStorage:
    enabled: true
    size: 5Gi
  
  # Extra volumes for TLS
  extraVolumes:
    - type: secret
      name: openbao-tls
      path: /vault/userconfig/tls
  
  # Readiness/Liveness probes
  readinessProbe:
    enabled: true
    path: "/v1/sys/health?standbyok=true&sealedcode=204&uninitcode=204"
  livenessProbe:
    enabled: true
    path: "/v1/sys/health?standbyok=true"
    initialDelaySeconds: 60

  # Service configuration
  service:
    enabled: true
    type: ClusterIP
    port: 8200

  # Ingress
  ingress:
    enabled: true
    annotations:
      kubernetes.io/ingress.class: nginx
      cert-manager.io/cluster-issuer: letsencrypt-prod
    hosts:
      - host: bao.yourdomain.com
        paths:
          - /
    tls:
      - secretName: openbao-tls-ingress
        hosts:
          - bao.yourdomain.com

# UI
ui:
  enabled: true
  serviceType: ClusterIP

# CSI Provider (optional)
csi:
  enabled: false

# Injector (optional)
injector:
  enabled: false
EOF

# Install
helm install openbao openbao/openbao \
  --namespace banhbao \
  --values openbao-values.yaml
```

### 4.3 TLS Certificate Setup

#### Using cert-manager (Recommended)

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Wait for cert-manager
kubectl wait --for=condition=ready pod -l app=cert-manager -n cert-manager --timeout=120s

# Create ClusterIssuer for Let's Encrypt
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@yourdomain.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF

# Create internal TLS certificate
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: openbao-tls
  namespace: banhbao
spec:
  secretName: openbao-tls
  duration: 8760h  # 1 year
  renewBefore: 720h  # 30 days
  subject:
    organizations:
      - BanhBao
  commonName: openbao.banhbao.svc.cluster.local
  isCA: false
  privateKey:
    algorithm: RSA
    size: 4096
  usages:
    - server auth
    - client auth
  dnsNames:
    - openbao
    - openbao.banhbao
    - openbao.banhbao.svc
    - openbao.banhbao.svc.cluster.local
    - openbao-0.openbao-internal
    - openbao-1.openbao-internal
    - openbao-2.openbao-internal
    - "*.openbao-internal"
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
EOF
```

#### Using Self-Signed Certificates

```bash
# Generate CA
openssl genrsa -out ca.key 4096
openssl req -x509 -new -nodes -key ca.key -sha256 -days 3650 \
  -out ca.crt -subj "/CN=BanhBao CA"

# Generate server certificate
openssl genrsa -out tls.key 4096
openssl req -new -key tls.key -out tls.csr \
  -subj "/CN=openbao.banhbao.svc.cluster.local"

# Create extensions file
cat > extfile.cnf << EOF
subjectAltName = DNS:openbao, DNS:openbao.banhbao, DNS:openbao.banhbao.svc, DNS:openbao.banhbao.svc.cluster.local, DNS:openbao-0.openbao-internal, DNS:openbao-1.openbao-internal, DNS:openbao-2.openbao-internal, DNS:*.openbao-internal, DNS:bao.yourdomain.com
EOF

openssl x509 -req -in tls.csr -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out tls.crt -days 365 -sha256 -extfile extfile.cnf

# Create Kubernetes secret
kubectl create secret generic openbao-tls \
  --namespace banhbao \
  --from-file=tls.crt \
  --from-file=tls.key \
  --from-file=ca.crt
```

### 4.4 Initialize OpenBao

```bash
# Wait for pods to be ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=openbao --timeout=300s -n banhbao

# Initialize (only on first node)
kubectl exec -n banhbao openbao-0 -- bao operator init \
  -key-shares=5 \
  -key-threshold=3 \
  -format=json > init-keys.json

# IMPORTANT: Securely store init-keys.json
# Contains unseal keys and root token

# Extract root token
export BAO_ROOT_TOKEN=$(cat init-keys.json | jq -r '.root_token')

# Unseal each node (repeat for openbao-0, openbao-1, openbao-2)
for i in 0 1 2; do
  for key in $(cat init-keys.json | jq -r '.unseal_keys_b64[0,1,2]'); do
    kubectl exec -n banhbao openbao-$i -- bao operator unseal $key
  done
done

# Verify status
kubectl exec -n banhbao openbao-0 -- bao status
```

### 4.5 Configure Transit Engine

```bash
# Port forward for local access
kubectl port-forward -n banhbao svc/openbao 8200:8200 &

# Set environment
export BAO_ADDR="https://127.0.0.1:8200"
export BAO_TOKEN="$BAO_ROOT_TOKEN"
export BAO_CACERT="./ca.crt"  # If using self-signed

# Enable Transit engine
bao secrets enable transit

# Create policy for BanhBao keyring
cat > banhbao-policy.hcl << 'EOF'
# Allow creating and managing keys
path "transit/keys/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

# Allow signing operations
path "transit/sign/*" {
  capabilities = ["create", "update"]
}

# Allow verification
path "transit/verify/*" {
  capabilities = ["create", "update"]
}

# Allow reading wrapping key (for imports)
path "transit/wrapping_key" {
  capabilities = ["read"]
}

# Allow key import
path "transit/keys/*/import" {
  capabilities = ["create", "update"]
}

# Allow key export (only for exportable keys)
path "transit/export/encryption-key/*" {
  capabilities = ["read"]
}
EOF

bao policy write banhbao banhbao-policy.hcl

# Create authentication method for applications
bao auth enable kubernetes

# Configure Kubernetes auth
bao write auth/kubernetes/config \
  kubernetes_host="https://$KUBERNETES_PORT_443_TCP_ADDR:443"

# Create role for BanhBao applications
bao write auth/kubernetes/role/banhbao \
  bound_service_account_names=banhbao \
  bound_service_account_namespaces=banhbao \
  policies=banhbao \
  ttl=24h
```

---

## 5. Application Token Setup

### 5.1 Service Account Token (Kubernetes Auth)

```bash
# Create service account
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ServiceAccount
metadata:
  name: banhbao
  namespace: banhbao
---
apiVersion: v1
kind: Secret
metadata:
  name: banhbao-token
  namespace: banhbao
  annotations:
    kubernetes.io/service-account.name: banhbao
type: kubernetes.io/service-account-token
EOF

# Applications authenticate via:
# 1. Read JWT from /var/run/secrets/kubernetes.io/serviceaccount/token
# 2. POST to /v1/auth/kubernetes/login with role=banhbao
```

### 5.2 Static Token (External Applications)

```bash
# Create token for external applications
bao token create \
  -policy=banhbao \
  -ttl=8760h \
  -display-name="banhbao-external" \
  -format=json > external-token.json

export BAO_TOKEN=$(cat external-token.json | jq -r '.auth.client_token')

# Store securely and provide to external applications
```

### 5.3 AppRole (Recommended for Production)

```bash
# Enable AppRole auth
bao auth enable approle

# Create role
bao write auth/approle/role/banhbao \
  token_policies="banhbao" \
  token_ttl=1h \
  token_max_ttl=24h \
  secret_id_ttl=10m \
  secret_id_num_uses=1

# Get RoleID (static, store in config)
bao read auth/approle/role/banhbao/role-id

# Get SecretID (dynamic, fetch at runtime)
bao write -f auth/approle/role/banhbao/secret-id
```

---

## 6. Ingress Configuration

### 6.1 NGINX Ingress

```bash
# Install NGINX Ingress Controller
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update

helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx \
  --create-namespace \
  --set controller.replicaCount=2

# Get external IP
kubectl get svc -n ingress-nginx
```

### 6.2 Traefik Ingress

```bash
# Install Traefik
helm repo add traefik https://traefik.github.io/charts
helm repo update

helm install traefik traefik/traefik \
  --namespace traefik \
  --create-namespace

# Create IngressRoute for OpenBao
cat <<EOF | kubectl apply -f -
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: openbao
  namespace: banhbao
spec:
  entryPoints:
    - websecure
  routes:
    - match: Host(\`bao.yourdomain.com\`)
      kind: Rule
      services:
        - name: openbao
          port: 8200
  tls:
    certResolver: letsencrypt
EOF
```

---

## 7. Monitoring Setup

### 7.1 Prometheus & Grafana

```bash
# Install kube-prometheus-stack
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false

# Create ServiceMonitor for OpenBao
cat <<EOF | kubectl apply -f -
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: openbao
  namespace: banhbao
  labels:
    release: prometheus
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: openbao
  endpoints:
    - port: http
      path: /v1/sys/metrics
      params:
        format: ["prometheus"]
      bearerTokenSecret:
        name: openbao-metrics-token
        key: token
EOF
```

### 7.2 OpenBao Metrics Token

```bash
# Create metrics policy
cat > metrics-policy.hcl << 'EOF'
path "sys/metrics" {
  capabilities = ["read"]
}
EOF

bao policy write metrics metrics-policy.hcl

# Create metrics token
bao token create \
  -policy=metrics \
  -ttl=8760h \
  -display-name="prometheus-metrics" \
  -format=json > metrics-token.json

# Create secret for ServiceMonitor
kubectl create secret generic openbao-metrics-token \
  --namespace banhbao \
  --from-literal=token=$(cat metrics-token.json | jq -r '.auth.client_token')
```

### 7.3 Grafana Dashboard

Import the OpenBao dashboard (ID: 12904) in Grafana:

1. Access Grafana: `kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80`
2. Login (default: admin/prom-operator)
3. Go to Dashboards → Import
4. Enter dashboard ID: 12904
5. Select Prometheus data source

---

## 8. Backup & Recovery

### 8.1 Automated Raft Snapshots

```bash
# Create backup CronJob
cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: CronJob
metadata:
  name: openbao-backup
  namespace: banhbao
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: banhbao
          containers:
          - name: backup
            image: quay.io/openbao/openbao:2.0.0
            command:
            - /bin/sh
            - -c
            - |
              export BAO_ADDR=https://openbao:8200
              export BAO_TOKEN=\$(cat /vault/secrets/token)
              bao operator raft snapshot save /backup/snapshot-\$(date +%Y%m%d-%H%M%S).snap
            volumeMounts:
            - name: backup
              mountPath: /backup
            - name: secrets
              mountPath: /vault/secrets
          volumes:
          - name: backup
            persistentVolumeClaim:
              claimName: openbao-backup
          - name: secrets
            secret:
              secretName: openbao-backup-token
          restartPolicy: OnFailure
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: openbao-backup
  namespace: banhbao
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
EOF
```

### 8.2 Manual Backup

```bash
# Take snapshot
kubectl exec -n banhbao openbao-0 -- bao operator raft snapshot save /tmp/raft-backup.snap

# Copy to local machine
kubectl cp banhbao/openbao-0:/tmp/raft-backup.snap ./raft-backup.snap
```

### 8.3 Disaster Recovery

```bash
# Restore from snapshot
kubectl cp ./raft-backup.snap banhbao/openbao-0:/tmp/raft-backup.snap

kubectl exec -n banhbao openbao-0 -- bao operator raft snapshot restore /tmp/raft-backup.snap

# Force new leader election if needed
kubectl exec -n banhbao openbao-0 -- bao operator raft remove-peer openbao-1
kubectl exec -n banhbao openbao-0 -- bao operator raft remove-peer openbao-2
```

---

## 9. Security Hardening

### 9.1 Network Policies

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: openbao-network-policy
  namespace: banhbao
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: openbao
  policyTypes:
    - Ingress
    - Egress
  ingress:
    # Allow from ingress controller
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
      ports:
        - protocol: TCP
          port: 8200
    # Allow cluster internal traffic
    - from:
        - podSelector: {}
      ports:
        - protocol: TCP
          port: 8200
        - protocol: TCP
          port: 8201
  egress:
    # Allow DNS
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: UDP
          port: 53
    # Allow cluster internal
    - to:
        - podSelector: {}
```

### 9.2 Pod Security

```yaml
# pod-security.yaml
apiVersion: v1
kind: Pod
metadata:
  name: openbao
spec:
  securityContext:
    runAsNonRoot: true
    runAsUser: 100
    fsGroup: 1000
  containers:
  - name: openbao
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop:
          - ALL
        add:
          - IPC_LOCK  # Required for mlock
```

### 9.3 Audit Logging

```bash
# Enable file audit
bao audit enable file file_path=/vault/audit/audit.log

# Enable syslog audit (to external SIEM)
bao audit enable syslog tag="openbao" facility="AUTH"
```

---

## 10. Troubleshooting

### 10.1 Common Issues

| Issue | Symptom | Solution |
| ----- | ------- | -------- |
| Pods not starting | CrashLoopBackOff | Check logs: `kubectl logs -n banhbao openbao-0` |
| Sealed after restart | 503 errors | Run unseal commands on all nodes |
| TLS errors | x509 certificate errors | Verify certificate SAN includes all hostnames |
| Raft not joining | Leader election failures | Check network policies, DNS resolution |
| Storage full | Write failures | Increase PVC size or clean up old data |

### 10.2 Debug Commands

```bash
# Check pod status
kubectl get pods -n banhbao -o wide

# View logs
kubectl logs -n banhbao openbao-0 -f

# Check events
kubectl get events -n banhbao --sort-by='.lastTimestamp'

# Exec into pod
kubectl exec -it -n banhbao openbao-0 -- /bin/sh

# Check Raft status
kubectl exec -n banhbao openbao-0 -- bao operator raft list-peers

# Check seal status
kubectl exec -n banhbao openbao-0 -- bao status
```

### 10.3 Health Checks

```bash
# Check health endpoint
kubectl exec -n banhbao openbao-0 -- \
  curl -sk https://localhost:8200/v1/sys/health | jq

# Expected response (healthy):
# {"initialized":true,"sealed":false,"standby":false,...}
```

---

## 11. Upgrade Procedure

### 11.1 Rolling Upgrade

```bash
# Update Helm values with new image tag
sed -i 's/tag: "2.0.0"/tag: "2.1.0"/' openbao-values.yaml

# Perform rolling upgrade
helm upgrade openbao openbao/openbao \
  --namespace banhbao \
  --values openbao-values.yaml \
  --wait

# Verify
kubectl rollout status statefulset/openbao -n banhbao
```

### 11.2 Pre-Upgrade Checklist

- [ ] Take Raft snapshot backup
- [ ] Review release notes for breaking changes
- [ ] Test upgrade in staging environment
- [ ] Verify unseal keys are available
- [ ] Schedule maintenance window
- [ ] Notify dependent applications

---

## 12. Quick Start Commands

```bash
# Complete setup in one script
#!/bin/bash
set -e

# 1. Create namespace
kubectl create namespace banhbao

# 2. Install OpenBao
helm install openbao openbao/openbao -n banhbao -f openbao-values.yaml

# 3. Wait for pods
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=openbao -n banhbao --timeout=300s

# 4. Initialize
kubectl exec -n banhbao openbao-0 -- bao operator init -key-shares=5 -key-threshold=3 -format=json > init-keys.json

# 5. Unseal
for i in 0 1 2; do
  for key in $(jq -r '.unseal_keys_b64[0,1,2]' init-keys.json); do
    kubectl exec -n banhbao openbao-$i -- bao operator unseal $key
  done
done

# 6. Enable Transit
export BAO_TOKEN=$(jq -r '.root_token' init-keys.json)
kubectl exec -n banhbao openbao-0 -- env BAO_TOKEN=$BAO_TOKEN bao secrets enable transit

echo "OpenBao is ready!"
echo "Root token: $BAO_TOKEN"
echo "Unseal keys in: init-keys.json"
```

---

## 13. References

- [OpenBao Documentation](https://openbao.org/docs/)
- [OpenBao Helm Chart](https://github.com/openbao/openbao-helm)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [cert-manager Documentation](https://cert-manager.io/docs/)

