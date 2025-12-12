package api

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
	"github.com/Bidon15/popsigner/operator/internal/constants"
)

func TestDeployment(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Domain: "keys.example.com",
			API: popsignerv1.APISpec{
				Version:  "1.0.0",
				Replicas: 3,
			},
		},
	}

	deployment := Deployment(cluster)

	// Verify name
	expectedName := "test-cluster-api"
	if deployment.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, deployment.Name)
	}

	// Verify namespace
	if deployment.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", deployment.Namespace)
	}

	// Verify labels
	if deployment.Labels[constants.LabelComponent] != constants.ComponentAPI {
		t.Errorf("expected component label %q, got %q", constants.ComponentAPI, deployment.Labels[constants.LabelComponent])
	}

	// Verify replicas
	if *deployment.Spec.Replicas != 3 {
		t.Errorf("expected 3 replicas, got %d", *deployment.Spec.Replicas)
	}

	// Verify container image
	expectedImage := "popsigner/control-plane:1.0.0"
	if deployment.Spec.Template.Spec.Containers[0].Image != expectedImage {
		t.Errorf("expected image %q, got %q", expectedImage, deployment.Spec.Template.Spec.Containers[0].Image)
	}

	// Verify container port
	if deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort != APIPort {
		t.Errorf("expected port %d, got %d", APIPort, deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
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
	if len(env) != 4 {
		t.Errorf("expected 4 env vars, got %d", len(env))
	}

	// Check DATABASE_URL
	foundDBURL := false
	for _, e := range env {
		if e.Name == "DATABASE_URL" {
			foundDBURL = true
			if e.ValueFrom == nil || e.ValueFrom.SecretKeyRef == nil {
				t.Error("expected DATABASE_URL to be a secret ref")
			} else if e.ValueFrom.SecretKeyRef.Name != "test-cluster-postgres-credentials" {
				t.Errorf("expected secret name 'test-cluster-postgres-credentials', got %q", e.ValueFrom.SecretKeyRef.Name)
			}
		}
	}
	if !foundDBURL {
		t.Error("expected DATABASE_URL env var to be present")
	}
}

func TestDeploymentDefaultVersion(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Domain: "keys.example.com",
			API:    popsignerv1.APISpec{
				// Version not set
			},
		},
	}

	deployment := Deployment(cluster)

	// Verify default version is used
	expectedImage := "popsigner/control-plane:" + constants.DefaultAPIVersion
	if deployment.Spec.Template.Spec.Containers[0].Image != expectedImage {
		t.Errorf("expected image %q, got %q", expectedImage, deployment.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestDeploymentDefaultReplicas(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Domain: "keys.example.com",
			API: popsignerv1.APISpec{
				Version: "1.0.0",
				// Replicas not set (0)
			},
		},
	}

	deployment := Deployment(cluster)

	// Verify default replicas is used
	if *deployment.Spec.Replicas != int32(constants.DefaultAPIReplicas) {
		t.Errorf("expected %d replicas, got %d", constants.DefaultAPIReplicas, *deployment.Spec.Replicas)
	}
}

func TestService(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Domain: "keys.example.com",
			API: popsignerv1.APISpec{
				Version: "1.0.0",
			},
		},
	}

	svc := Service(cluster)

	// Verify name
	expectedName := "test-cluster-api"
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
	if svc.Spec.Ports[0].Port != APIPort {
		t.Errorf("expected port %d, got %d", APIPort, svc.Spec.Ports[0].Port)
	}

	// Verify selector labels
	if svc.Spec.Selector[constants.LabelComponent] != constants.ComponentAPI {
		t.Errorf("expected selector component %q, got %q", constants.ComponentAPI, svc.Spec.Selector[constants.LabelComponent])
	}
}

func TestHPA(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Domain: "keys.example.com",
			API: popsignerv1.APISpec{
				Version: "1.0.0",
				Autoscaling: popsignerv1.AutoscalingSpec{
					Enabled:     true,
					MinReplicas: 2,
					MaxReplicas: 20,
					TargetCPU:   80,
				},
			},
		},
	}

	hpa := HPA(cluster)

	// Verify name
	expectedName := "test-cluster-api"
	if hpa.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, hpa.Name)
	}

	// Verify namespace
	if hpa.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", hpa.Namespace)
	}

	// Verify min replicas
	if *hpa.Spec.MinReplicas != 2 {
		t.Errorf("expected min replicas 2, got %d", *hpa.Spec.MinReplicas)
	}

	// Verify max replicas
	if hpa.Spec.MaxReplicas != 20 {
		t.Errorf("expected max replicas 20, got %d", hpa.Spec.MaxReplicas)
	}

	// Verify target CPU
	if len(hpa.Spec.Metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(hpa.Spec.Metrics))
	}
	if *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization != 80 {
		t.Errorf("expected target CPU 80, got %d", *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
	}

	// Verify scale target ref
	if hpa.Spec.ScaleTargetRef.Name != "test-cluster-api" {
		t.Errorf("expected scale target ref name 'test-cluster-api', got %q", hpa.Spec.ScaleTargetRef.Name)
	}
	if hpa.Spec.ScaleTargetRef.Kind != "Deployment" {
		t.Errorf("expected scale target ref kind 'Deployment', got %q", hpa.Spec.ScaleTargetRef.Kind)
	}
}

func TestHPADefaultValues(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Domain: "keys.example.com",
			API: popsignerv1.APISpec{
				Version: "1.0.0",
				Autoscaling: popsignerv1.AutoscalingSpec{
					Enabled: true,
					// Other values not set
				},
			},
		},
	}

	hpa := HPA(cluster)

	// Verify default min replicas
	if *hpa.Spec.MinReplicas != int32(constants.DefaultAPIReplicas) {
		t.Errorf("expected default min replicas %d, got %d", constants.DefaultAPIReplicas, *hpa.Spec.MinReplicas)
	}

	// Verify default max replicas
	if hpa.Spec.MaxReplicas != 10 {
		t.Errorf("expected default max replicas 10, got %d", hpa.Spec.MaxReplicas)
	}

	// Verify default target CPU
	if *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization != 70 {
		t.Errorf("expected default target CPU 70, got %d", *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
	}
}

func TestBuildEnv(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Domain: "keys.example.com",
		},
	}

	env := buildEnv(cluster)

	// Verify we have all expected env vars
	envMap := make(map[string]bool)
	for _, e := range env {
		envMap[e.Name] = true
	}

	expectedEnvs := []string{"DATABASE_URL", "REDIS_URL", "OPENBAO_ADDR", "OPENBAO_TOKEN"}
	for _, expected := range expectedEnvs {
		if !envMap[expected] {
			t.Errorf("expected env var %q to be present", expected)
		}
	}

	// Verify OPENBAO_ADDR is a direct value (not secret ref)
	for _, e := range env {
		if e.Name == "OPENBAO_ADDR" {
			expectedAddr := "https://test-cluster-openbao-active:8200"
			if e.Value != expectedAddr {
				t.Errorf("expected OPENBAO_ADDR %q, got %q", expectedAddr, e.Value)
			}
		}
	}
}

func TestSecretRef(t *testing.T) {
	ref := secretRef("my-secret", "my-key")

	if ref.SecretKeyRef.Name != "my-secret" {
		t.Errorf("expected secret name 'my-secret', got %q", ref.SecretKeyRef.Name)
	}
	if ref.SecretKeyRef.Key != "my-key" {
		t.Errorf("expected key 'my-key', got %q", ref.SecretKeyRef.Key)
	}
}
