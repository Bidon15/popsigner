# Agent 13H: Helm Chart & Release

## Overview

Create the Helm chart for operator installation, including CRDs, RBAC, and deployment manifests. Prepare for release to Helm repository.

> **Requires:** All other Phase 9 agents complete

---

## Deliverables

### 1. Chart Structure

```
charts/
└── banhbaoring-operator/
    ├── Chart.yaml
    ├── values.yaml
    ├── templates/
    │   ├── _helpers.tpl
    │   ├── deployment.yaml
    │   ├── service.yaml
    │   ├── serviceaccount.yaml
    │   ├── clusterrole.yaml
    │   ├── clusterrolebinding.yaml
    │   ├── NOTES.txt
    │   └── crds/
    │       ├── banhbaoringcluster-crd.yaml
    │       ├── banhbaoringtenant-crd.yaml
    │       ├── banhbaoringbackup-crd.yaml
    │       └── banhbaoringrestore-crd.yaml
    └── README.md
```

### 2. Chart.yaml

```yaml
# charts/banhbaoring-operator/Chart.yaml
apiVersion: v2
name: banhbaoring-operator
description: BanhBaoRing Kubernetes Operator - Deploy and manage secure key infrastructure
type: application
version: 0.1.0
appVersion: "1.0.0"
keywords:
  - key-management
  - openbao
  - vault
  - celestia
  - rollups
  - signing
maintainers:
  - name: BanhBaoRing Team
    url: https://github.com/Bidon15/banhbaoring
home: https://banhbaoring.io
sources:
  - https://github.com/Bidon15/banhbaoring
icon: https://banhbaoring.io/logo.png
```

### 3. values.yaml

```yaml
# charts/banhbaoring-operator/values.yaml

replicaCount: 1

image:
  repository: banhbaoring/operator
  pullPolicy: IfNotPresent
  tag: ""  # Defaults to appVersion

imagePullSecrets: []
nameOverride: ""
fullnameOverride: ""

serviceAccount:
  create: true
  annotations: {}
  name: ""

rbac:
  create: true

resources:
  limits:
    cpu: 500m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi

nodeSelector: {}
tolerations: []
affinity: {}

# Leader election
leaderElection:
  enabled: true

# Metrics
metrics:
  enabled: true
  port: 8080

# Health probes
health:
  port: 8081

# Log level: debug, info, warn, error
logLevel: info

# Install CRDs
installCRDs: true
```

### 4. Deployment Template

```yaml
# charts/banhbaoring-operator/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "banhbaoring-operator.fullname" . }}
  labels:
    {{- include "banhbaoring-operator.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "banhbaoring-operator.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "banhbaoring-operator.selectorLabels" . | nindent 8 }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "banhbaoring-operator.serviceAccountName" . }}
      securityContext:
        runAsNonRoot: true
      containers:
        - name: manager
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          args:
            - --leader-elect={{ .Values.leaderElection.enabled }}
            - --metrics-bind-address=:{{ .Values.metrics.port }}
            - --health-probe-bind-address=:{{ .Values.health.port }}
            - --log-level={{ .Values.logLevel }}
          ports:
            - name: metrics
              containerPort: {{ .Values.metrics.port }}
            - name: health
              containerPort: {{ .Values.health.port }}
          livenessProbe:
            httpGet:
              path: /healthz
              port: health
            initialDelaySeconds: 15
            periodSeconds: 20
          readinessProbe:
            httpGet:
              path: /readyz
              port: health
            initialDelaySeconds: 5
            periodSeconds: 10
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
          securityContext:
            allowPrivilegeEscalation: false
            capabilities:
              drop:
                - ALL
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
```

### 5. ClusterRole Template

```yaml
# charts/banhbaoring-operator/templates/clusterrole.yaml
{{- if .Values.rbac.create }}
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: {{ include "banhbaoring-operator.fullname" . }}-manager
rules:
  # BanhBaoRing resources
  - apiGroups: ["banhbaoring.io"]
    resources: ["*"]
    verbs: ["*"]
  # Core resources
  - apiGroups: [""]
    resources: ["configmaps", "secrets", "services", "persistentvolumeclaims"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: [""]
    resources: ["pods", "events"]
    verbs: ["get", "list", "watch"]
  # Apps
  - apiGroups: ["apps"]
    resources: ["statefulsets", "deployments"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  # Batch
  - apiGroups: ["batch"]
    resources: ["jobs", "cronjobs"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  # Networking
  - apiGroups: ["networking.k8s.io"]
    resources: ["ingresses", "networkpolicies"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  # Autoscaling
  - apiGroups: ["autoscaling"]
    resources: ["horizontalpodautoscalers"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  # cert-manager
  - apiGroups: ["cert-manager.io"]
    resources: ["certificates"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  # Prometheus Operator
  - apiGroups: ["monitoring.coreos.com"]
    resources: ["prometheuses", "servicemonitors", "prometheusrules"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
{{- end }}
```

### 6. NOTES.txt

```
# charts/banhbaoring-operator/templates/NOTES.txt
🔔 Ring ring! BanhBaoRing Operator installed successfully!

To deploy a BanhBaoRing cluster:

1. Create a namespace:
   kubectl create namespace banhbaoring

2. Create required secrets:
   kubectl create secret generic stripe-secrets \
     --namespace banhbaoring \
     --from-literal=secret-key=sk_live_xxx

3. Deploy a cluster:
   cat <<EOF | kubectl apply -f -
   apiVersion: banhbaoring.io/v1
   kind: BanhBaoRingCluster
   metadata:
     name: production
     namespace: banhbaoring
   spec:
     domain: keys.mycompany.com
     openbao:
       replicas: 3
     database:
       managed: true
     redis:
       managed: true
   EOF

4. Check status:
   kubectl get banhbaoringcluster -n banhbaoring

For more information: https://banhbaoring.io/docs
```

### 7. Makefile Additions

```makefile
# Makefile additions for Helm

CHART_DIR := charts/banhbaoring-operator

.PHONY: helm-lint helm-template helm-package helm-push

helm-lint:
	helm lint $(CHART_DIR)

helm-template:
	helm template banhbaoring $(CHART_DIR) --debug

helm-package:
	helm package $(CHART_DIR) -d dist/

helm-push: helm-package
	helm push dist/banhbaoring-operator-*.tgz oci://ghcr.io/bidon15/charts
```

### 8. Release Workflow

```yaml
# .github/workflows/release.yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build operator
        run: |
          cd operator
          make docker-build IMG=ghcr.io/bidon15/banhbaoring-operator:${{ github.ref_name }}

      - name: Push image
        run: |
          echo ${{ secrets.GITHUB_TOKEN }} | docker login ghcr.io -u ${{ github.actor }} --password-stdin
          docker push ghcr.io/bidon15/banhbaoring-operator:${{ github.ref_name }}

      - name: Package Helm chart
        run: |
          cd operator
          make helm-package

      - name: Push Helm chart
        run: |
          cd operator
          echo ${{ secrets.GITHUB_TOKEN }} | helm registry login ghcr.io -u ${{ github.actor }} --password-stdin
          make helm-push
```

---

## Test Commands

```bash
cd operator

# Lint chart
make helm-lint

# Template locally
make helm-template

# Package
make helm-package

# Install in test cluster
kind create cluster --name helm-test
helm install banhbaoring-operator ./charts/banhbaoring-operator \
  --namespace banhbaoring-system \
  --create-namespace
```

---

## Acceptance Criteria

- [ ] Chart.yaml with proper metadata
- [ ] values.yaml with all configurable options
- [ ] Deployment, Service, RBAC templates
- [ ] CRD templates included
- [ ] NOTES.txt with usage instructions
- [ ] Helm lint passes
- [ ] Helm template generates valid YAML
- [ ] Release workflow ready

