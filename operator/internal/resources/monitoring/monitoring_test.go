package monitoring

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
	"github.com/Bidon15/popsigner/operator/internal/constants"
)

func testCluster() *popsignerv1.POPSignerCluster {
	return &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Domain: "keys.example.com",
			Monitoring: popsignerv1.MonitoringSpec{
				Enabled: true,
				Prometheus: popsignerv1.PrometheusSpec{
					Retention: "30d",
					Storage: popsignerv1.StorageSpec{
						Size:         resource.MustParse("100Gi"),
						StorageClass: "fast-ssd",
					},
				},
				Grafana: popsignerv1.GrafanaSpec{
					Enabled: true,
					AdminPassword: &popsignerv1.SecretKeyRef{
						Name: "grafana-secrets",
						Key:  "admin-password",
					},
				},
				Alerting: popsignerv1.AlertingSpec{
					Enabled: true,
				},
			},
		},
	}
}

func TestPrometheus(t *testing.T) {
	cluster := testCluster()
	prom := Prometheus(cluster)

	// Verify name
	expectedName := "test-cluster-prometheus"
	if prom.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, prom.Name)
	}

	// Verify namespace
	if prom.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", prom.Namespace)
	}

	// Verify labels
	if prom.Labels[constants.LabelComponent] != ComponentPrometheus {
		t.Errorf("expected component label %q, got %q", ComponentPrometheus, prom.Labels[constants.LabelComponent])
	}

	// Verify retention
	if string(prom.Spec.Retention) != "30d" {
		t.Errorf("expected retention '30d', got %q", prom.Spec.Retention)
	}

	// Verify ServiceMonitor selector (in CommonPrometheusFields)
	if prom.Spec.CommonPrometheusFields.ServiceMonitorSelector.MatchLabels[constants.LabelInstance] != "test-cluster" {
		t.Errorf("expected ServiceMonitorSelector instance 'test-cluster', got %q",
			prom.Spec.CommonPrometheusFields.ServiceMonitorSelector.MatchLabels[constants.LabelInstance])
	}

	// Verify storage
	if prom.Spec.Storage == nil {
		t.Fatal("expected storage to be set")
	}
	storageSize := prom.Spec.Storage.VolumeClaimTemplate.Spec.Resources.Requests["storage"]
	if storageSize.String() != "100Gi" {
		t.Errorf("expected storage size '100Gi', got %q", storageSize.String())
	}
	if *prom.Spec.Storage.VolumeClaimTemplate.Spec.StorageClassName != "fast-ssd" {
		t.Errorf("expected storage class 'fast-ssd', got %q",
			*prom.Spec.Storage.VolumeClaimTemplate.Spec.StorageClassName)
	}
}

func TestPrometheusDefaultRetention(t *testing.T) {
	cluster := testCluster()
	cluster.Spec.Monitoring.Prometheus.Retention = ""

	prom := Prometheus(cluster)

	if string(prom.Spec.Retention) != DefaultRetention {
		t.Errorf("expected default retention %q, got %q", DefaultRetention, prom.Spec.Retention)
	}
}

func TestPrometheusWithoutStorage(t *testing.T) {
	cluster := testCluster()
	cluster.Spec.Monitoring.Prometheus.Storage = popsignerv1.StorageSpec{}

	prom := Prometheus(cluster)

	if prom.Spec.Storage != nil {
		t.Error("expected storage to be nil when not configured")
	}
}

func TestServiceMonitor(t *testing.T) {
	cluster := testCluster()
	sm := ServiceMonitor(cluster, "openbao", 8200)

	// Verify name
	expectedName := "test-cluster-openbao"
	if sm.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, sm.Name)
	}

	// Verify namespace
	if sm.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", sm.Namespace)
	}

	// Verify selector
	selector := sm.Spec.Selector.MatchLabels
	if selector[constants.LabelComponent] != "openbao" {
		t.Errorf("expected selector component 'openbao', got %q", selector[constants.LabelComponent])
	}

	// Verify endpoints
	if len(sm.Spec.Endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(sm.Spec.Endpoints))
	}
	if sm.Spec.Endpoints[0].Port != "metrics" {
		t.Errorf("expected endpoint port 'metrics', got %q", sm.Spec.Endpoints[0].Port)
	}
	if string(sm.Spec.Endpoints[0].Interval) != DefaultScrapeInterval {
		t.Errorf("expected interval %q, got %q", DefaultScrapeInterval, sm.Spec.Endpoints[0].Interval)
	}
}

func TestPrometheusService(t *testing.T) {
	cluster := testCluster()
	svc := PrometheusService(cluster)

	// Verify name
	expectedName := "test-cluster-prometheus"
	if svc.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, svc.Name)
	}

	// Verify port
	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(svc.Spec.Ports))
	}
	if svc.Spec.Ports[0].Port != int32(constants.PortPrometheus) {
		t.Errorf("expected port %d, got %d", constants.PortPrometheus, svc.Spec.Ports[0].Port)
	}
}

func TestGrafanaDeployment(t *testing.T) {
	cluster := testCluster()
	deployment := GrafanaDeployment(cluster)

	// Verify name
	expectedName := "test-cluster-grafana"
	if deployment.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, deployment.Name)
	}

	// Verify namespace
	if deployment.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", deployment.Namespace)
	}

	// Verify labels
	if deployment.Labels[constants.LabelComponent] != ComponentGrafana {
		t.Errorf("expected component label %q, got %q", ComponentGrafana, deployment.Labels[constants.LabelComponent])
	}

	// Verify replicas
	if *deployment.Spec.Replicas != 1 {
		t.Errorf("expected 1 replica, got %d", *deployment.Spec.Replicas)
	}

	// Verify container image
	if deployment.Spec.Template.Spec.Containers[0].Image != GrafanaImage {
		t.Errorf("expected image %q, got %q", GrafanaImage, deployment.Spec.Template.Spec.Containers[0].Image)
	}

	// Verify container port
	if deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort != int32(constants.PortGrafana) {
		t.Errorf("expected port %d, got %d",
			constants.PortGrafana, deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
	}

	// Verify volume mounts
	mounts := deployment.Spec.Template.Spec.Containers[0].VolumeMounts
	if len(mounts) != 2 {
		t.Fatalf("expected 2 volume mounts, got %d", len(mounts))
	}

	// Verify admin password env var
	env := deployment.Spec.Template.Spec.Containers[0].Env
	foundAdminPassword := false
	for _, e := range env {
		if e.Name == "GF_SECURITY_ADMIN_PASSWORD" {
			foundAdminPassword = true
			if e.ValueFrom == nil || e.ValueFrom.SecretKeyRef == nil {
				t.Error("expected admin password to be a secret ref")
			} else if e.ValueFrom.SecretKeyRef.Name != "grafana-secrets" {
				t.Errorf("expected secret name 'grafana-secrets', got %q", e.ValueFrom.SecretKeyRef.Name)
			}
		}
	}
	if !foundAdminPassword {
		t.Error("expected GF_SECURITY_ADMIN_PASSWORD env var to be present")
	}

	// Verify probes
	if deployment.Spec.Template.Spec.Containers[0].LivenessProbe == nil {
		t.Error("expected liveness probe to be set")
	}
	if deployment.Spec.Template.Spec.Containers[0].ReadinessProbe == nil {
		t.Error("expected readiness probe to be set")
	}
}

func TestGrafanaDeploymentWithoutAdminPassword(t *testing.T) {
	cluster := testCluster()
	cluster.Spec.Monitoring.Grafana.AdminPassword = nil

	deployment := GrafanaDeployment(cluster)

	// Verify admin password env var is NOT present
	env := deployment.Spec.Template.Spec.Containers[0].Env
	for _, e := range env {
		if e.Name == "GF_SECURITY_ADMIN_PASSWORD" {
			t.Error("expected GF_SECURITY_ADMIN_PASSWORD env var to NOT be present")
		}
	}
}

func TestGrafanaService(t *testing.T) {
	cluster := testCluster()
	svc := GrafanaService(cluster)

	// Verify name
	expectedName := "test-cluster-grafana"
	if svc.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, svc.Name)
	}

	// Verify port
	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(svc.Spec.Ports))
	}
	if svc.Spec.Ports[0].Port != int32(constants.PortGrafana) {
		t.Errorf("expected port %d, got %d", constants.PortGrafana, svc.Spec.Ports[0].Port)
	}
}

func TestDatasourcesConfigMap(t *testing.T) {
	cluster := testCluster()
	cm := DatasourcesConfigMap(cluster)

	// Verify name
	expectedName := "test-cluster-grafana-datasources"
	if cm.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, cm.Name)
	}

	// Verify namespace
	if cm.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", cm.Namespace)
	}

	// Verify datasources.yaml exists and contains Prometheus URL
	yaml, ok := cm.Data["datasources.yaml"]
	if !ok {
		t.Fatal("expected datasources.yaml to be present")
	}
	if len(yaml) == 0 {
		t.Error("expected datasources.yaml to have content")
	}
	// Check for Prometheus URL
	expectedURL := "http://test-cluster-prometheus:9090"
	if !contains(yaml, expectedURL) {
		t.Errorf("expected datasources.yaml to contain %q", expectedURL)
	}
}

func TestDashboardsConfigMap(t *testing.T) {
	cluster := testCluster()
	cm := DashboardsConfigMap(cluster)

	// Verify name
	expectedName := "test-cluster-grafana-dashboards"
	if cm.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, cm.Name)
	}

	// Verify namespace
	if cm.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", cm.Namespace)
	}

	// Verify dashboards.yaml exists
	yaml, ok := cm.Data["dashboards.yaml"]
	if !ok {
		t.Fatal("expected dashboards.yaml to be present")
	}
	if len(yaml) == 0 {
		t.Error("expected dashboards.yaml to have content")
	}
}

func TestPrometheusRules(t *testing.T) {
	cluster := testCluster()
	rules := PrometheusRules(cluster)

	// Verify name
	expectedName := "test-cluster-alerts"
	if rules.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, rules.Name)
	}

	// Verify namespace
	if rules.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", rules.Namespace)
	}

	// Verify labels
	if rules.Labels[constants.LabelComponent] != ComponentAlerts {
		t.Errorf("expected component label %q, got %q", ComponentAlerts, rules.Labels[constants.LabelComponent])
	}

	// Verify rule groups
	if len(rules.Spec.Groups) != 1 {
		t.Fatalf("expected 1 rule group, got %d", len(rules.Spec.Groups))
	}

	group := rules.Spec.Groups[0]
	if group.Name != "popsigner.rules" {
		t.Errorf("expected group name 'popsigner.rules', got %q", group.Name)
	}

	// Verify we have alert rules
	if len(group.Rules) == 0 {
		t.Error("expected at least one alert rule")
	}

	// Verify OpenBaoSealed alert exists
	foundOpenBaoSealed := false
	for _, rule := range group.Rules {
		if rule.Alert == "OpenBaoSealed" {
			foundOpenBaoSealed = true
			if rule.Labels["severity"] != "critical" {
				t.Errorf("expected severity 'critical', got %q", rule.Labels["severity"])
			}
		}
	}
	if !foundOpenBaoSealed {
		t.Error("expected OpenBaoSealed alert to be present")
	}

	// Verify HighSigningLatency alert exists
	foundHighSigningLatency := false
	for _, rule := range group.Rules {
		if rule.Alert == "HighSigningLatency" {
			foundHighSigningLatency = true
		}
	}
	if !foundHighSigningLatency {
		t.Error("expected HighSigningLatency alert to be present")
	}

	// Verify APIHighErrorRate alert exists
	foundAPIHighErrorRate := false
	for _, rule := range group.Rules {
		if rule.Alert == "APIHighErrorRate" {
			foundAPIHighErrorRate = true
		}
	}
	if !foundAPIHighErrorRate {
		t.Error("expected APIHighErrorRate alert to be present")
	}
}

func TestDurationPtr(t *testing.T) {
	d := durationPtr("5m")
	if d == nil {
		t.Fatal("expected duration pointer to not be nil")
	}
	if string(*d) != "5m" {
		t.Errorf("expected duration '5m', got %q", *d)
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
