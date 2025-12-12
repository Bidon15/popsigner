// Package monitoring provides resource builders for the monitoring stack.
package monitoring

import (
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	popsignerv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
	"github.com/Bidon15/banhbaoring/operator/internal/constants"
)

const (
	// ComponentPrometheus is the component name for Prometheus.
	ComponentPrometheus = "prometheus"
	// DefaultRetention is the default metrics retention period.
	DefaultRetention = "15d"
	// DefaultScrapeInterval is the default scrape interval.
	DefaultScrapeInterval = "30s"
)

// Prometheus creates a Prometheus CR for the cluster.
func Prometheus(cluster *popsignerv1.POPSignerCluster) *monitoringv1.Prometheus {
	spec := cluster.Spec.Monitoring.Prometheus
	name := fmt.Sprintf("%s-prometheus", cluster.Name)
	labels := constants.Labels(cluster.Name, ComponentPrometheus, "")

	retention := spec.Retention
	if retention == "" {
		retention = DefaultRetention
	}

	prom := &monitoringv1.Prometheus{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: monitoringv1.PrometheusSpec{
			CommonPrometheusFields: monitoringv1.CommonPrometheusFields{
				ServiceMonitorSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						constants.LabelInstance: cluster.Name,
					},
				},
			},
			Retention: monitoringv1.Duration(retention),
			RuleSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					constants.LabelInstance: cluster.Name,
				},
			},
		},
	}

	// Configure storage if specified
	if !spec.Storage.Size.IsZero() {
		prom.Spec.Storage = &monitoringv1.StorageSpec{
			VolumeClaimTemplate: monitoringv1.EmbeddedPersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: spec.Storage.Size,
						},
					},
				},
			},
		}

		// Add storage class if specified
		if spec.Storage.StorageClass != "" {
			prom.Spec.Storage.VolumeClaimTemplate.Spec.StorageClassName = &spec.Storage.StorageClass
		}
	}

	return prom
}

// ServiceMonitor creates a ServiceMonitor for a component.
func ServiceMonitor(cluster *popsignerv1.POPSignerCluster, component string, port int) *monitoringv1.ServiceMonitor {
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
				MatchLabels: constants.SelectorLabels(cluster.Name, component),
			},
			Endpoints: []monitoringv1.Endpoint{{
				Port:     "metrics",
				Interval: monitoringv1.Duration(DefaultScrapeInterval),
			}},
		},
	}
}

// PrometheusService creates a Service for Prometheus.
func PrometheusService(cluster *popsignerv1.POPSignerCluster) *corev1.Service {
	name := fmt.Sprintf("%s-prometheus", cluster.Name)
	labels := constants.Labels(cluster.Name, ComponentPrometheus, "")
	selector := constants.SelectorLabels(cluster.Name, ComponentPrometheus)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: selector,
			Ports: []corev1.ServicePort{{
				Name:     "web",
				Port:     int32(constants.PortPrometheus),
				Protocol: corev1.ProtocolTCP,
			}},
		},
	}
}
