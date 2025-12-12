# POPSigner Operator Helm Chart

Deploy and manage secure key infrastructure with the POPSigner Kubernetes Operator.

## Prerequisites

- Kubernetes 1.25+
- Helm 3.0+
- (Optional) cert-manager for TLS certificates
- (Optional) Prometheus Operator for monitoring

## Installation

### Add the Helm repository

```bash
helm repo add popsigner https://charts.popsigner.com
helm repo update
```

### Install the chart

```bash
helm install popsigner-operator popsigner/popsigner-operator \
  --namespace popsigner-system \
  --create-namespace
```

### Install from local chart

```bash
helm install popsigner-operator ./charts/popsigner-operator \
  --namespace popsigner-system \
  --create-namespace
```

## Configuration

The following table lists the configurable parameters:

| Parameter                    | Description                          | Default                |
| ---------------------------- | ------------------------------------ | ---------------------- |
| `replicaCount`               | Number of operator replicas          | `1`                    |
| `image.repository`           | Operator image repository            | `popsigner/operator`   |
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

After installing the operator, you can deploy POPSigner clusters:

### Create a namespace

```bash
kubectl create namespace popsigner
```

### Deploy a minimal cluster

```yaml
apiVersion: popsigner.com/v1
kind: POPSignerCluster
metadata:
  name: production
  namespace: popsigner
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
apiVersion: popsigner.com/v1
kind: POPSignerTenant
metadata:
  name: my-org
  namespace: popsigner
spec:
  clusterRef:
    name: production
  displayName: "My Organization"
  plan: starter
```

## CRDs

This chart installs the following Custom Resource Definitions:

- `POPSignerCluster` - Manages the full POPSigner infrastructure
- `POPSignerTenant` - Manages tenant organizations
- `POPSignerBackup` - Manages backup operations
- `POPSignerRestore` - Manages restore operations

## Upgrading

### From 0.x to 1.x

```bash
helm upgrade popsigner-operator popsigner/popsigner-operator \
  --namespace popsigner-system
```

## Uninstallation

```bash
helm uninstall popsigner-operator --namespace popsigner-system
```

**Note:** CRDs are not automatically deleted. To remove them:

```bash
kubectl delete crd popsignerclusters.popsigner.com
kubectl delete crd popsignertenants.popsigner.com
kubectl delete crd popsignerbackups.popsigner.com
kubectl delete crd popsignerrestores.popsigner.com
```

## Support

- Documentation: https://popsigner.com/docs
- Issues: https://github.com/Bidon15/popsigner/issues
