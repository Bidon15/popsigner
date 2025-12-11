package monitoring

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	banhbaoringv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
	"github.com/Bidon15/banhbaoring/operator/internal/constants"
	"github.com/Bidon15/banhbaoring/operator/internal/resources"
)

const (
	// GrafanaImage is the default Grafana container image.
	GrafanaImage = "grafana/grafana:10.2.0"
	// GrafanaVersion is the version of Grafana being deployed.
	GrafanaVersion = "10.2.0"
	// ComponentGrafana is the component name for Grafana.
	ComponentGrafana = "grafana"
)

// GrafanaDeployment creates a Deployment for Grafana.
func GrafanaDeployment(cluster *banhbaoringv1.BanhBaoRingCluster) *appsv1.Deployment {
	name := fmt.Sprintf("%s-grafana", cluster.Name)
	labels := constants.Labels(cluster.Name, ComponentGrafana, GrafanaVersion)
	selector := constants.SelectorLabels(cluster.Name, ComponentGrafana)
	replicas := int32(1)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: selector},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "grafana",
						Image: GrafanaImage,
						Ports: []corev1.ContainerPort{{
							Name:          "http",
							ContainerPort: int32(constants.PortGrafana),
						}},
						Env:            grafanaEnv(cluster),
						VolumeMounts:   grafanaVolumeMounts(),
						LivenessProbe:  resources.HTTPGetProbe("/api/health", constants.PortGrafana, 30, 10),
						ReadinessProbe: resources.HTTPGetProbe("/api/health", constants.PortGrafana, 5, 5),
					}},
					Volumes: grafanaVolumes(cluster),
				},
			},
		},
	}
}

func grafanaEnv(cluster *banhbaoringv1.BanhBaoRingCluster) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{Name: "GF_INSTALL_PLUGINS", Value: "grafana-piechart-panel"},
	}

	// Add admin password from secret if specified
	ref := cluster.Spec.Monitoring.Grafana.AdminPassword
	if ref != nil {
		env = append(env, corev1.EnvVar{
			Name: "GF_SECURITY_ADMIN_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: ref.Name},
					Key:                  ref.Key,
				},
			},
		})
	}

	return env
}

func grafanaVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{Name: "dashboards", MountPath: "/etc/grafana/provisioning/dashboards"},
		{Name: "datasources", MountPath: "/etc/grafana/provisioning/datasources"},
	}
}

func grafanaVolumes(cluster *banhbaoringv1.BanhBaoRingCluster) []corev1.Volume {
	name := fmt.Sprintf("%s-grafana", cluster.Name)
	return []corev1.Volume{
		{
			Name: "dashboards",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: name + "-dashboards"},
				},
			},
		},
		{
			Name: "datasources",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: name + "-datasources"},
				},
			},
		},
	}
}

// GrafanaService creates a Service for Grafana.
func GrafanaService(cluster *banhbaoringv1.BanhBaoRingCluster) *corev1.Service {
	name := fmt.Sprintf("%s-grafana", cluster.Name)
	labels := constants.Labels(cluster.Name, ComponentGrafana, GrafanaVersion)
	selector := constants.SelectorLabels(cluster.Name, ComponentGrafana)

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
				Name:     "http",
				Port:     int32(constants.PortGrafana),
				Protocol: corev1.ProtocolTCP,
			}},
		},
	}
}

// DatasourcesConfigMap creates a ConfigMap for Grafana datasources.
func DatasourcesConfigMap(cluster *banhbaoringv1.BanhBaoRingCluster) *corev1.ConfigMap {
	name := fmt.Sprintf("%s-grafana-datasources", cluster.Name)
	labels := constants.Labels(cluster.Name, ComponentGrafana, GrafanaVersion)
	prometheusURL := fmt.Sprintf("http://%s-prometheus:%d", cluster.Name, constants.PortPrometheus)

	datasourcesYAML := fmt.Sprintf(`apiVersion: 1
datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: %s
    isDefault: true
`, prometheusURL)

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"datasources.yaml": datasourcesYAML,
		},
	}
}

// DashboardsConfigMap creates a ConfigMap for Grafana dashboard provisioning.
func DashboardsConfigMap(cluster *banhbaoringv1.BanhBaoRingCluster) *corev1.ConfigMap {
	name := fmt.Sprintf("%s-grafana-dashboards", cluster.Name)
	labels := constants.Labels(cluster.Name, ComponentGrafana, GrafanaVersion)

	provisioningYAML := `apiVersion: 1
providers:
  - name: 'banhbaoring'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    options:
      path: /var/lib/grafana/dashboards
`

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"dashboards.yaml": provisioningYAML,
		},
	}
}
