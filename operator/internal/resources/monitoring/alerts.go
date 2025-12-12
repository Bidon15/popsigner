package monitoring

import (
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	popsignerv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
	"github.com/Bidon15/banhbaoring/operator/internal/constants"
)

const (
	// ComponentAlerts is the component name for alerting rules.
	ComponentAlerts = "alerts"
)

// PrometheusRules creates a PrometheusRule for the cluster.
func PrometheusRules(cluster *popsignerv1.POPSignerCluster) *monitoringv1.PrometheusRule {
	name := fmt.Sprintf("%s-alerts", cluster.Name)
	labels := constants.Labels(cluster.Name, ComponentAlerts, "")

	return &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:  "popsigner.rules",
					Rules: buildAlertRules(),
				},
			},
		},
	}
}

func buildAlertRules() []monitoringv1.Rule {
	return []monitoringv1.Rule{
		{
			Alert: "OpenBaoSealed",
			Expr:  intstr.FromString(`vault_core_unsealed == 0`),
			For:   durationPtr("5m"),
			Labels: map[string]string{
				"severity": "critical",
			},
			Annotations: map[string]string{
				"summary":     "OpenBao is sealed",
				"description": "OpenBao has been sealed for more than 5 minutes",
			},
		},
		{
			Alert: "OpenBaoHighLeadershipChanges",
			Expr:  intstr.FromString(`increase(vault_core_leadership_setup_failed[1h]) > 3`),
			For:   durationPtr("5m"),
			Labels: map[string]string{
				"severity": "warning",
			},
			Annotations: map[string]string{
				"summary":     "OpenBao high leadership changes",
				"description": "OpenBao has experienced more than 3 leadership changes in the last hour",
			},
		},
		{
			Alert: "HighSigningLatency",
			Expr:  intstr.FromString(`histogram_quantile(0.99, sum(rate(popsigner_sign_duration_seconds_bucket[5m])) by (le)) > 1`),
			For:   durationPtr("5m"),
			Labels: map[string]string{
				"severity": "warning",
			},
			Annotations: map[string]string{
				"summary":     "High signing latency detected",
				"description": "99th percentile signing latency is above 1 second",
			},
		},
		{
			Alert: "APIHighErrorRate",
			Expr:  intstr.FromString(`sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) > 0.05`),
			For:   durationPtr("5m"),
			Labels: map[string]string{
				"severity": "warning",
			},
			Annotations: map[string]string{
				"summary":     "High API error rate",
				"description": "API error rate is above 5% for the last 5 minutes",
			},
		},
		{
			Alert: "APIDown",
			Expr:  intstr.FromString(`up{job=~".*-api"} == 0`),
			For:   durationPtr("2m"),
			Labels: map[string]string{
				"severity": "critical",
			},
			Annotations: map[string]string{
				"summary":     "API is down",
				"description": "API has been down for more than 2 minutes",
			},
		},
		{
			Alert: "PostgreSQLDown",
			Expr:  intstr.FromString(`pg_up == 0`),
			For:   durationPtr("2m"),
			Labels: map[string]string{
				"severity": "critical",
			},
			Annotations: map[string]string{
				"summary":     "PostgreSQL is down",
				"description": "PostgreSQL database has been down for more than 2 minutes",
			},
		},
		{
			Alert: "RedisDown",
			Expr:  intstr.FromString(`redis_up == 0`),
			For:   durationPtr("2m"),
			Labels: map[string]string{
				"severity": "critical",
			},
			Annotations: map[string]string{
				"summary":     "Redis is down",
				"description": "Redis has been down for more than 2 minutes",
			},
		},
		{
			Alert: "HighMemoryUsage",
			Expr:  intstr.FromString(`container_memory_usage_bytes / container_spec_memory_limit_bytes > 0.9`),
			For:   durationPtr("5m"),
			Labels: map[string]string{
				"severity": "warning",
			},
			Annotations: map[string]string{
				"summary":     "High memory usage",
				"description": "Container memory usage is above 90%",
			},
		},
		{
			Alert: "PodRestartLoop",
			Expr:  intstr.FromString(`increase(kube_pod_container_status_restarts_total[1h]) > 5`),
			For:   durationPtr("10m"),
			Labels: map[string]string{
				"severity": "warning",
			},
			Annotations: map[string]string{
				"summary":     "Pod restart loop detected",
				"description": "Pod has restarted more than 5 times in the last hour",
			},
		},
	}
}

func durationPtr(s string) *monitoringv1.Duration {
	d := monitoringv1.Duration(s)
	return &d
}
