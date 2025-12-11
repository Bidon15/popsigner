# Agent 13G: Monitoring Controller

## Overview

Implement monitoring stack deployment: Prometheus, Grafana, and AlertManager. Includes ServiceMonitors, pre-built dashboards, and alerting rules.

> **Requires:** Agent 13A (Operator Foundation) complete

---

## Deliverables

### 1. Prometheus Resources

```go
// internal/resources/monitoring/prometheus.go
package monitoring

import (
    "fmt"

    monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

    banhbaoringv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
    "github.com/Bidon15/banhbaoring/operator/internal/constants"
)

func Prometheus(cluster *banhbaoringv1.BanhBaoRingCluster) *monitoringv1.Prometheus {
    spec := cluster.Spec.Monitoring.Prometheus
    name := fmt.Sprintf("%s-prometheus", cluster.Name)
    labels := constants.Labels(cluster.Name, "prometheus", "")

    retention := spec.Retention
    if retention == "" {
        retention = "15d"
    }

    return &monitoringv1.Prometheus{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: cluster.Namespace,
            Labels:    labels,
        },
        Spec: monitoringv1.PrometheusSpec{
            Retention: monitoringv1.Duration(retention),
            ServiceMonitorSelector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    constants.LabelInstance: cluster.Name,
                },
            },
            RuleSelector: &metav1.LabelSelector{
                MatchLabels: map[string]string{
                    constants.LabelInstance: cluster.Name,
                },
            },
            Storage: &monitoringv1.StorageSpec{
                VolumeClaimTemplate: monitoringv1.EmbeddedPersistentVolumeClaim{
                    Spec: corev1.PersistentVolumeClaimSpec{
                        Resources: corev1.VolumeResourceRequirements{
                            Requests: corev1.ResourceList{
                                corev1.ResourceStorage: spec.Storage.Size,
                            },
                        },
                    },
                },
            },
        },
    }
}

func ServiceMonitor(cluster *banhbaoringv1.BanhBaoRingCluster, component string, port int) *monitoringv1.ServiceMonitor {
    name := fmt.Sprintf("%s-%s", cluster.Name, component)
    labels := constants.Labels(cluster.Name, component, "")

    return &monitoringv1.ServiceMonitor{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: cluster.Namespace,
            Labels:    labels,
        },
        Spec: monitoringv1.ServiceMonitorSpec{
            Selector: metav1.LabelSelector{
                MatchLabels: labels,
            },
            Endpoints: []monitoringv1.Endpoint{{
                Port:     "metrics",
                Interval: monitoringv1.Duration("30s"),
            }},
        },
    }
}
```

### 2. Grafana Resources

```go
// internal/resources/monitoring/grafana.go
package monitoring

import (
    "fmt"

    appsv1 "k8s.io/api/apps/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

    banhbaoringv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
    "github.com/Bidon15/banhbaoring/operator/internal/constants"
)

const GrafanaImage = "grafana/grafana:10.2.0"

func GrafanaDeployment(cluster *banhbaoringv1.BanhBaoRingCluster) *appsv1.Deployment {
    name := fmt.Sprintf("%s-grafana", cluster.Name)
    labels := constants.Labels(cluster.Name, "grafana", "10.2.0")
    replicas := int32(1)

    return &appsv1.Deployment{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: cluster.Namespace,
            Labels:    labels,
        },
        Spec: appsv1.DeploymentSpec{
            Replicas: &replicas,
            Selector: &metav1.LabelSelector{MatchLabels: labels},
            Template: corev1.PodTemplateSpec{
                ObjectMeta: metav1.ObjectMeta{Labels: labels},
                Spec: corev1.PodSpec{
                    Containers: []corev1.Container{{
                        Name:  "grafana",
                        Image: GrafanaImage,
                        Ports: []corev1.ContainerPort{{ContainerPort: 3000}},
                        Env: []corev1.EnvVar{
                            {Name: "GF_SECURITY_ADMIN_PASSWORD", ValueFrom: adminPasswordRef(cluster)},
                            {Name: "GF_INSTALL_PLUGINS", Value: "grafana-piechart-panel"},
                        },
                        VolumeMounts: []corev1.VolumeMount{
                            {Name: "dashboards", MountPath: "/etc/grafana/provisioning/dashboards"},
                            {Name: "datasources", MountPath: "/etc/grafana/provisioning/datasources"},
                        },
                    }},
                    Volumes: []corev1.Volume{
                        {Name: "dashboards", VolumeSource: corev1.VolumeSource{
                            ConfigMap: &corev1.ConfigMapVolumeSource{
                                LocalObjectReference: corev1.LocalObjectReference{Name: name + "-dashboards"},
                            },
                        }},
                        {Name: "datasources", VolumeSource: corev1.VolumeSource{
                            ConfigMap: &corev1.ConfigMapVolumeSource{
                                LocalObjectReference: corev1.LocalObjectReference{Name: name + "-datasources"},
                            },
                        }},
                    },
                },
            },
        },
    }
}

func adminPasswordRef(cluster *banhbaoringv1.BanhBaoRingCluster) *corev1.EnvVarSource {
    ref := cluster.Spec.Monitoring.Grafana.AdminPassword
    if ref == nil {
        return nil
    }
    return &corev1.EnvVarSource{
        SecretKeyRef: &corev1.SecretKeySelector{
            LocalObjectReference: corev1.LocalObjectReference{Name: ref.Name},
            Key:                  ref.Key,
        },
    }
}

func DatasourcesConfigMap(cluster *banhbaoringv1.BanhBaoRingCluster) *corev1.ConfigMap {
    name := fmt.Sprintf("%s-grafana", cluster.Name)
    prometheusURL := fmt.Sprintf("http://%s-prometheus:9090", cluster.Name)

    datasourcesYAML := fmt.Sprintf(`
apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: %s
    isDefault: true
`, prometheusURL)

    return &corev1.ConfigMap{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name + "-datasources",
            Namespace: cluster.Namespace,
        },
        Data: map[string]string{
            "datasources.yaml": datasourcesYAML,
        },
    }
}
```

### 3. Alert Rules

```go
// internal/resources/monitoring/alerts.go
package monitoring

import (
    "fmt"

    monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/util/intstr"

    banhbaoringv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
    "github.com/Bidon15/banhbaoring/operator/internal/constants"
)

func PrometheusRules(cluster *banhbaoringv1.BanhBaoRingCluster) *monitoringv1.PrometheusRule {
    name := fmt.Sprintf("%s-alerts", cluster.Name)
    labels := constants.Labels(cluster.Name, "alerts", "")

    return &monitoringv1.PrometheusRule{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: cluster.Namespace,
            Labels:    labels,
        },
        Spec: monitoringv1.PrometheusRuleSpec{
            Groups: []monitoringv1.RuleGroup{
                {
                    Name: "banhbaoring.rules",
                    Rules: []monitoringv1.Rule{
                        {
                            Alert: "OpenBaoSealed",
                            Expr:  intstr.FromString(`vault_core_unsealed == 0`),
                            For:   ptr("5m"),
                            Labels: map[string]string{
                                "severity": "critical",
                            },
                            Annotations: map[string]string{
                                "summary":     "OpenBao is sealed",
                                "description": "OpenBao has been sealed for more than 5 minutes",
                            },
                        },
                        {
                            Alert: "HighSigningLatency",
                            Expr:  intstr.FromString(`histogram_quantile(0.99, sum(rate(banhbaoring_sign_duration_seconds_bucket[5m])) by (le)) > 1`),
                            For:   ptr("5m"),
                            Labels: map[string]string{
                                "severity": "warning",
                            },
                            Annotations: map[string]string{
                                "summary": "High signing latency detected",
                            },
                        },
                        {
                            Alert: "APIHighErrorRate",
                            Expr:  intstr.FromString(`sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) > 0.05`),
                            For:   ptr("5m"),
                            Labels: map[string]string{
                                "severity": "warning",
                            },
                        },
                    },
                },
            },
        },
    }
}

func ptr(s string) *monitoringv1.Duration {
    d := monitoringv1.Duration(s)
    return &d
}
```

### 4. Controller Integration

```go
// controllers/cluster_monitoring.go
package controllers

func (r *ClusterReconciler) reconcileMonitoring(ctx context.Context, cluster *banhbaoringv1.BanhBaoRingCluster) error {
    if !cluster.Spec.Monitoring.Enabled {
        return nil
    }

    log := log.FromContext(ctx)
    log.Info("Reconciling monitoring stack")

    // Prometheus
    if err := r.createOrUpdate(ctx, cluster, monitoring.Prometheus(cluster)); err != nil {
        return err
    }

    // ServiceMonitors
    for _, component := range []string{"openbao", "api", "dashboard"} {
        sm := monitoring.ServiceMonitor(cluster, component, 8080)
        if err := r.createOrUpdate(ctx, cluster, sm); err != nil {
            log.Error(err, "Failed to create ServiceMonitor", "component", component)
        }
    }

    // Alert rules
    if err := r.createOrUpdate(ctx, cluster, monitoring.PrometheusRules(cluster)); err != nil {
        return err
    }

    // Grafana
    if cluster.Spec.Monitoring.Grafana.Enabled {
        if err := r.createOrUpdate(ctx, cluster, monitoring.DatasourcesConfigMap(cluster)); err != nil {
            return err
        }
        if err := r.createOrUpdate(ctx, cluster, monitoring.GrafanaDeployment(cluster)); err != nil {
            return err
        }
    }

    return nil
}
```

---

## Test Commands

```bash
cd operator
go build ./...
go test ./internal/resources/monitoring/... -v
```

---

## Acceptance Criteria

- [ ] Prometheus CR creation
- [ ] ServiceMonitors for all components
- [ ] Grafana Deployment with provisioning
- [ ] Datasource ConfigMap
- [ ] Alert rules (PrometheusRule)
- [ ] Controller integration

