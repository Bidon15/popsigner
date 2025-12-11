# BanhBaoRing Operator Helm Chart

Deploy and manage secure key infrastructure with the BanhBaoRing Kubernetes Operator.

## Prerequisites

- Kubernetes 1.25+
- Helm 3.0+
- (Optional) cert-manager for TLS certificates
- (Optional) Prometheus Operator for monitoring

## Installation

### Add the Helm repository

```bash
helm repo add banhbaoring https://charts.banhbaoring.io
helm repo update
```

### Install the chart

```bash
helm install banhbaoring-operator banhbaoring/banhbaoring-operator \
  --namespace banhbaoring-system \
  --create-namespace
```

### Install from local chart

```bash
helm install banhbaoring-operator ./charts/banhbaoring-operator \
  --namespace banhbaoring-system \
  --create-namespace
```

## Configuration

The following table lists the configurable parameters:

| Parameter                    | Description                          | Default                |
| ---------------------------- | ------------------------------------ | ---------------------- |
| `replicaCount`               | Number of operator replicas          | `1`                    |
| `image.repository`           | Operator image repository            | `banhbaoring/operator` |
| `image.tag`                  | Operator image tag                   | `""` (uses appVersion) |
| `image.pullPolicy`           | Image pull policy                    | `IfNotPresent`         |
| `imagePullSecrets`           | Image pull secrets                   | `[]`                   |
| `nameOverride`               | Override chart name                  | `""`                   |
| `fullnameOverride`           | Override full name                   | `""`                   |
| `serviceAccount.create`      | Create service account               | `true`                 |
| `serviceAccount.annotations` | Service account annotations          | `{}`                   |
| `serviceAccount.name`        | Service account name                 | `""`                   |
| `rbac.create`                | Create RBAC resources                | `true`                 |
| `resources.limits.cpu`       | CPU limit                            | `500m`                 |
| `resources.limits.memory`    | Memory limit                         | `256Mi`                |
| `resources.requests.cpu`     | CPU request                          | `100m`                 |
| `resources.requests.memory`  | Memory request                       | `128Mi`                |
| `nodeSelector`               | Node selector                        | `{}`                   |
| `tolerations`                | Tolerations                          | `[]`                   |
| `affinity`                   | Affinity rules                       | `{}`                   |
| `leaderElection.enabled`     | Enable leader election               | `true`                 |
| `metrics.enabled`            | Enable metrics endpoint              | `true`                 |
| `metrics.port`               | Metrics port                         | `8080`                 |
| `health.port`                | Health probe port                    | `8081`                 |
| `logLevel`                   | Log level (debug, info, warn, error) | `info`                 |
| `installCRDs`                | Install CRDs with chart              | `true`                 |

## Usage

After installing the operator, you can deploy BanhBaoRing clusters:

### Create a namespace

```bash
kubectl create namespace banhbaoring
```

### Deploy a minimal cluster

```yaml
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
```

### Deploy a tenant

```yaml
apiVersion: banhbaoring.io/v1
kind: BanhBaoRingTenant
metadata:
  name: my-org
  namespace: banhbaoring
spec:
  clusterRef:
    name: production
  displayName: "My Organization"
  plan: starter
```

## CRDs

This chart installs the following Custom Resource Definitions:

- `BanhBaoRingCluster` - Manages the full BanhBaoRing infrastructure
- `BanhBaoRingTenant` - Manages tenant organizations
- `BanhBaoRingBackup` - Manages backup operations
- `BanhBaoRingRestore` - Manages restore operations

## Upgrading

### From 0.x to 1.x

```bash
helm upgrade banhbaoring-operator banhbaoring/banhbaoring-operator \
  --namespace banhbaoring-system
```

## Uninstallation

```bash
helm uninstall banhbaoring-operator --namespace banhbaoring-system
```

**Note:** CRDs are not automatically deleted. To remove them:

```bash
kubectl delete crd banhbaoringclusters.banhbaoring.io
kubectl delete crd banhbaoringtenants.banhbaoring.io
kubectl delete crd banhbaoringbackups.banhbaoring.io
kubectl delete crd banhbaoringrestores.banhbaoring.io
```

## Support

- Documentation: https://banhbaoring.io/docs
- Issues: https://github.com/Bidon15/banhbaoring/issues
