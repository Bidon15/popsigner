package dashboard

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	banhbaoringv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
	"github.com/Bidon15/banhbaoring/operator/internal/constants"
)

func TestDeployment(t *testing.T) {
	cluster := &banhbaoringv1.BanhBaoRingCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: banhbaoringv1.BanhBaoRingClusterSpec{
			Domain: "keys.example.com",
			Dashboard: banhbaoringv1.DashboardSpec{
				Version:  "1.0.0",
				Replicas: 3,
			},
		},
	}

	deployment := Deployment(cluster)

	// Verify name
	expectedName := "test-cluster-dashboard"
	if deployment.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, deployment.Name)
	}

	// Verify namespace
	if deployment.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", deployment.Namespace)
	}

	// Verify labels
	if deployment.Labels[constants.LabelComponent] != constants.ComponentDashboard {
		t.Errorf("expected component label %q, got %q", constants.ComponentDashboard, deployment.Labels[constants.LabelComponent])
	}

	// Verify replicas
	if *deployment.Spec.Replicas != 3 {
		t.Errorf("expected 3 replicas, got %d", *deployment.Spec.Replicas)
	}

	// Verify container image
	expectedImage := "banhbaoring/dashboard:1.0.0"
	if deployment.Spec.Template.Spec.Containers[0].Image != expectedImage {
		t.Errorf("expected image %q, got %q", expectedImage, deployment.Spec.Template.Spec.Containers[0].Image)
	}

	// Verify container port
	if deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort != DashboardPort {
		t.Errorf("expected port %d, got %d", DashboardPort, deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
	}

	// Verify readiness probe
	probe := deployment.Spec.Template.Spec.Containers[0].ReadinessProbe
	if probe == nil {
		t.Fatal("expected readiness probe to be set")
	}
	if probe.HTTPGet.Path != "/health" {
		t.Errorf("expected readiness probe path '/health', got %q", probe.HTTPGet.Path)
	}

	// Verify environment variables
	env := deployment.Spec.Template.Spec.Containers[0].Env
	if len(env) != 1 {
		t.Errorf("expected 1 env var, got %d", len(env))
	}

	// Check API_URL
	if env[0].Name != "API_URL" {
		t.Errorf("expected env var name 'API_URL', got %q", env[0].Name)
	}
	expectedAPIURL := "http://test-cluster-api:8080"
	if env[0].Value != expectedAPIURL {
		t.Errorf("expected API_URL %q, got %q", expectedAPIURL, env[0].Value)
	}
}

func TestDeploymentDefaultVersion(t *testing.T) {
	cluster := &banhbaoringv1.BanhBaoRingCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: banhbaoringv1.BanhBaoRingClusterSpec{
			Domain:    "keys.example.com",
			Dashboard: banhbaoringv1.DashboardSpec{
				// Version not set
			},
		},
	}

	deployment := Deployment(cluster)

	// Verify default version is used
	expectedImage := "banhbaoring/dashboard:" + constants.DefaultDashboardVersion
	if deployment.Spec.Template.Spec.Containers[0].Image != expectedImage {
		t.Errorf("expected image %q, got %q", expectedImage, deployment.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestDeploymentDefaultReplicas(t *testing.T) {
	cluster := &banhbaoringv1.BanhBaoRingCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: banhbaoringv1.BanhBaoRingClusterSpec{
			Domain: "keys.example.com",
			Dashboard: banhbaoringv1.DashboardSpec{
				Version: "1.0.0",
				// Replicas not set (0)
			},
		},
	}

	deployment := Deployment(cluster)

	// Verify default replicas is used
	if *deployment.Spec.Replicas != int32(constants.DefaultDashboardReplicas) {
		t.Errorf("expected %d replicas, got %d", constants.DefaultDashboardReplicas, *deployment.Spec.Replicas)
	}
}

func TestService(t *testing.T) {
	cluster := &banhbaoringv1.BanhBaoRingCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: banhbaoringv1.BanhBaoRingClusterSpec{
			Domain: "keys.example.com",
			Dashboard: banhbaoringv1.DashboardSpec{
				Version: "1.0.0",
			},
		},
	}

	svc := Service(cluster)

	// Verify name
	expectedName := "test-cluster-dashboard"
	if svc.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, svc.Name)
	}

	// Verify namespace
	if svc.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", svc.Namespace)
	}

	// Verify port
	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(svc.Spec.Ports))
	}
	if svc.Spec.Ports[0].Port != DashboardPort {
		t.Errorf("expected port %d, got %d", DashboardPort, svc.Spec.Ports[0].Port)
	}

	// Verify selector labels
	if svc.Spec.Selector[constants.LabelComponent] != constants.ComponentDashboard {
		t.Errorf("expected selector component %q, got %q", constants.ComponentDashboard, svc.Spec.Selector[constants.LabelComponent])
	}
}

func TestServiceDefaultVersion(t *testing.T) {
	cluster := &banhbaoringv1.BanhBaoRingCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: banhbaoringv1.BanhBaoRingClusterSpec{
			Domain:    "keys.example.com",
			Dashboard: banhbaoringv1.DashboardSpec{
				// Version not set
			},
		},
	}

	svc := Service(cluster)

	// Verify default version is used in labels
	if svc.Labels[constants.LabelVersion] != constants.DefaultDashboardVersion {
		t.Errorf("expected version label %q, got %q", constants.DefaultDashboardVersion, svc.Labels[constants.LabelVersion])
	}
}

func TestMergeResources(t *testing.T) {
	// Test with no override
	result := mergeResources(banhbaoringv1.BanhBaoRingCluster{}.Spec.Dashboard.Resources)

	if result.Requests == nil {
		t.Fatal("expected default requests to be set")
	}
	if result.Limits == nil {
		t.Fatal("expected default limits to be set")
	}
}
