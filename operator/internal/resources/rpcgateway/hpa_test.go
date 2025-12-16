package rpcgateway

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
	"github.com/Bidon15/popsigner/operator/internal/constants"
)

func TestHorizontalPodAutoscaler(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Domain: "keys.example.com",
			RPCGateway: popsignerv1.RPCGatewaySpec{
				Enabled:  true,
				Version:  "1.0.0",
				Replicas: 3,
			},
		},
	}

	hpa := HorizontalPodAutoscaler(cluster)

	// Verify name
	expectedName := "test-cluster-rpc-gateway"
	if hpa.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, hpa.Name)
	}

	// Verify namespace
	if hpa.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", hpa.Namespace)
	}

	// Verify scale target ref
	if hpa.Spec.ScaleTargetRef.Name != "test-cluster-rpc-gateway" {
		t.Errorf("expected scale target ref name 'test-cluster-rpc-gateway', got %q", hpa.Spec.ScaleTargetRef.Name)
	}
	if hpa.Spec.ScaleTargetRef.Kind != "Deployment" {
		t.Errorf("expected scale target ref kind 'Deployment', got %q", hpa.Spec.ScaleTargetRef.Kind)
	}
	if hpa.Spec.ScaleTargetRef.APIVersion != "apps/v1" {
		t.Errorf("expected scale target ref API version 'apps/v1', got %q", hpa.Spec.ScaleTargetRef.APIVersion)
	}

	// Verify min replicas (should be max of default and configured)
	if *hpa.Spec.MinReplicas != 3 {
		t.Errorf("expected min replicas 3, got %d", *hpa.Spec.MinReplicas)
	}

	// Verify max replicas
	if hpa.Spec.MaxReplicas != 10 {
		t.Errorf("expected max replicas 10, got %d", hpa.Spec.MaxReplicas)
	}

	// Verify metrics
	if len(hpa.Spec.Metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(hpa.Spec.Metrics))
	}
	metric := hpa.Spec.Metrics[0]
	if metric.Resource.Name != corev1.ResourceCPU {
		t.Errorf("expected CPU metric, got %q", metric.Resource.Name)
	}
	if *metric.Resource.Target.AverageUtilization != 70 {
		t.Errorf("expected target CPU 70, got %d", *metric.Resource.Target.AverageUtilization)
	}

	// Verify labels
	if hpa.Labels[constants.LabelComponent] != constants.ComponentRPCGateway {
		t.Errorf("expected label component %q, got %q", constants.ComponentRPCGateway, hpa.Labels[constants.LabelComponent])
	}
}

func TestHorizontalPodAutoscalerDefaultReplicas(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Domain: "keys.example.com",
			RPCGateway: popsignerv1.RPCGatewaySpec{
				Enabled: true,
				// Replicas not set
			},
		},
	}

	hpa := HorizontalPodAutoscaler(cluster)

	// Verify default min replicas
	if *hpa.Spec.MinReplicas != int32(constants.DefaultRPCGatewayReplicas) {
		t.Errorf("expected default min replicas %d, got %d", constants.DefaultRPCGatewayReplicas, *hpa.Spec.MinReplicas)
	}
}

func TestHorizontalPodAutoscalerHigherConfiguredReplicas(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Domain: "keys.example.com",
			RPCGateway: popsignerv1.RPCGatewaySpec{
				Enabled:  true,
				Replicas: 5, // Higher than default
			},
		},
	}

	hpa := HorizontalPodAutoscaler(cluster)

	// Min replicas should use configured value since it's higher
	if *hpa.Spec.MinReplicas != 5 {
		t.Errorf("expected min replicas 5, got %d", *hpa.Spec.MinReplicas)
	}
}

func TestHorizontalPodAutoscalerBehavior(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Domain: "keys.example.com",
			RPCGateway: popsignerv1.RPCGatewaySpec{
				Enabled: true,
			},
		},
	}

	hpa := HorizontalPodAutoscaler(cluster)

	// Verify behavior is set
	if hpa.Spec.Behavior == nil {
		t.Fatal("expected behavior to be set")
	}

	// Verify scale down behavior
	if hpa.Spec.Behavior.ScaleDown == nil {
		t.Fatal("expected scaleDown behavior to be set")
	}
	if *hpa.Spec.Behavior.ScaleDown.StabilizationWindowSeconds != 300 {
		t.Errorf("expected scaleDown stabilization window 300, got %d", *hpa.Spec.Behavior.ScaleDown.StabilizationWindowSeconds)
	}

	// Verify scale up behavior
	if hpa.Spec.Behavior.ScaleUp == nil {
		t.Fatal("expected scaleUp behavior to be set")
	}
	if *hpa.Spec.Behavior.ScaleUp.StabilizationWindowSeconds != 30 {
		t.Errorf("expected scaleUp stabilization window 30, got %d", *hpa.Spec.Behavior.ScaleUp.StabilizationWindowSeconds)
	}
	if len(hpa.Spec.Behavior.ScaleUp.Policies) != 2 {
		t.Errorf("expected 2 scaleUp policies, got %d", len(hpa.Spec.Behavior.ScaleUp.Policies))
	}
}

func TestHorizontalPodAutoscalerDefaultVersion(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Domain: "keys.example.com",
			RPCGateway: popsignerv1.RPCGatewaySpec{
				Enabled: true,
				// Version not set
			},
		},
	}

	hpa := HorizontalPodAutoscaler(cluster)

	// Verify default version is used in labels
	if hpa.Labels[constants.LabelVersion] != constants.DefaultRPCGatewayVersion {
		t.Errorf("expected version label %q, got %q", constants.DefaultRPCGatewayVersion, hpa.Labels[constants.LabelVersion])
	}
}

