package openbao

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	popsignerv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
	"github.com/Bidon15/banhbaoring/operator/internal/constants"
)

func TestHeadlessService(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			OpenBao: popsignerv1.OpenBaoSpec{
				Version: "2.0.0",
			},
		},
	}

	svc := HeadlessService(cluster)

	// Check name
	expectedName := "test-cluster-openbao"
	if svc.Name != expectedName {
		t.Errorf("Service name = %v, want %v", svc.Name, expectedName)
	}

	// Check namespace
	if svc.Namespace != "default" {
		t.Errorf("Service namespace = %v, want default", svc.Namespace)
	}

	// Check it's headless
	if svc.Spec.ClusterIP != corev1.ClusterIPNone {
		t.Errorf("Service ClusterIP = %v, want None", svc.Spec.ClusterIP)
	}

	// Check ports
	if len(svc.Spec.Ports) != 2 {
		t.Fatalf("Expected 2 ports, got %d", len(svc.Spec.Ports))
	}

	portNames := make(map[string]int32)
	for _, p := range svc.Spec.Ports {
		portNames[p.Name] = p.Port
	}

	if port, ok := portNames["api"]; !ok || port != OpenBaoPort {
		t.Errorf("Expected api port %d, got %d", OpenBaoPort, port)
	}
	if port, ok := portNames["cluster"]; !ok || port != OpenBaoClusterPort {
		t.Errorf("Expected cluster port %d, got %d", OpenBaoClusterPort, port)
	}

	// Check PublishNotReadyAddresses for StatefulSet DNS
	if !svc.Spec.PublishNotReadyAddresses {
		t.Error("PublishNotReadyAddresses should be true for headless service")
	}

	// Check labels
	if svc.Labels[constants.LabelComponent] != constants.ComponentOpenBao {
		t.Errorf("Label component = %v, want %v", svc.Labels[constants.LabelComponent], constants.ComponentOpenBao)
	}
}

func TestActiveService(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "production",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			OpenBao: popsignerv1.OpenBaoSpec{
				Version: "2.0.0",
			},
		},
	}

	svc := ActiveService(cluster)

	// Check name
	expectedName := "test-cluster-openbao-active"
	if svc.Name != expectedName {
		t.Errorf("Service name = %v, want %v", svc.Name, expectedName)
	}

	// Check namespace
	if svc.Namespace != "production" {
		t.Errorf("Service namespace = %v, want production", svc.Namespace)
	}

	// Check it's not headless
	if svc.Spec.ClusterIP == corev1.ClusterIPNone {
		t.Error("Active service should not be headless")
	}

	// Check type
	if svc.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Errorf("Service type = %v, want ClusterIP", svc.Spec.Type)
	}

	// Check ports - only API port
	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("Expected 1 port, got %d", len(svc.Spec.Ports))
	}
	if svc.Spec.Ports[0].Port != OpenBaoPort {
		t.Errorf("Expected port %d, got %d", OpenBaoPort, svc.Spec.Ports[0].Port)
	}

	// Check selector labels
	if svc.Spec.Selector[constants.LabelComponent] != constants.ComponentOpenBao {
		t.Errorf("Selector component = %v, want %v", svc.Spec.Selector[constants.LabelComponent], constants.ComponentOpenBao)
	}
}

func TestInternalService(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			OpenBao: popsignerv1.OpenBaoSpec{
				Version: "2.0.0",
			},
		},
	}

	svc := InternalService(cluster)

	// Check name
	expectedName := "test-cluster-openbao-internal"
	if svc.Name != expectedName {
		t.Errorf("Service name = %v, want %v", svc.Name, expectedName)
	}

	// Check it's ClusterIP type
	if svc.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Errorf("Service type = %v, want ClusterIP", svc.Spec.Type)
	}

	// Check ports - only API port
	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("Expected 1 port, got %d", len(svc.Spec.Ports))
	}
	if svc.Spec.Ports[0].Port != OpenBaoPort {
		t.Errorf("Expected port %d, got %d", OpenBaoPort, svc.Spec.Ports[0].Port)
	}
}
