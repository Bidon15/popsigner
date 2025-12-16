package rpcgateway

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
	"github.com/Bidon15/popsigner/operator/internal/constants"
)

func TestService(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Domain: "keys.example.com",
			RPCGateway: popsignerv1.RPCGatewaySpec{
				Enabled: true,
				Version: "1.0.0",
			},
		},
	}

	svc := Service(cluster)

	// Verify name
	expectedName := "test-cluster-rpc-gateway"
	if svc.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, svc.Name)
	}

	// Verify namespace
	if svc.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", svc.Namespace)
	}

	// Verify service type
	if svc.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Errorf("expected service type ClusterIP, got %q", svc.Spec.Type)
	}

	// Verify port
	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(svc.Spec.Ports))
	}
	if svc.Spec.Ports[0].Port != int32(constants.PortRPCGateway) {
		t.Errorf("expected port %d, got %d", constants.PortRPCGateway, svc.Spec.Ports[0].Port)
	}
	if svc.Spec.Ports[0].Name != "jsonrpc" {
		t.Errorf("expected port name 'jsonrpc', got %q", svc.Spec.Ports[0].Name)
	}

	// Verify selector labels
	if svc.Spec.Selector[constants.LabelComponent] != constants.ComponentRPCGateway {
		t.Errorf("expected selector component %q, got %q", constants.ComponentRPCGateway, svc.Spec.Selector[constants.LabelComponent])
	}
	if svc.Spec.Selector[constants.LabelInstance] != "test-cluster" {
		t.Errorf("expected selector instance 'test-cluster', got %q", svc.Spec.Selector[constants.LabelInstance])
	}

	// Verify prometheus annotations
	if svc.Annotations["prometheus.io/scrape"] != "true" {
		t.Error("expected prometheus scrape annotation to be 'true'")
	}
	if svc.Annotations["prometheus.io/port"] != "8545" {
		t.Error("expected prometheus port annotation to be '8545'")
	}

	// Verify labels
	if svc.Labels[constants.LabelComponent] != constants.ComponentRPCGateway {
		t.Errorf("expected label component %q, got %q", constants.ComponentRPCGateway, svc.Labels[constants.LabelComponent])
	}
	if svc.Labels[constants.LabelManagedBy] != constants.ManagedBy {
		t.Errorf("expected label managed-by %q, got %q", constants.ManagedBy, svc.Labels[constants.LabelManagedBy])
	}
}

func TestServiceDefaultVersion(t *testing.T) {
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

	svc := Service(cluster)

	// Verify default version is used in labels
	if svc.Labels[constants.LabelVersion] != constants.DefaultRPCGatewayVersion {
		t.Errorf("expected version label %q, got %q", constants.DefaultRPCGatewayVersion, svc.Labels[constants.LabelVersion])
	}
}

