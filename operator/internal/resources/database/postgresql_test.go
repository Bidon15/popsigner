package database

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
	"github.com/Bidon15/popsigner/operator/internal/constants"
)

func TestStatefulSet(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Database: popsignerv1.DatabaseSpec{
				Managed:  true,
				Version:  "16",
				Replicas: 1,
				Storage: popsignerv1.StorageSpec{
					Size:         resource.MustParse("10Gi"),
					StorageClass: "standard",
				},
			},
		},
	}

	sts := StatefulSet(cluster)

	// Verify name
	expectedName := "test-cluster-postgres"
	if sts.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, sts.Name)
	}

	// Verify namespace
	if sts.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", sts.Namespace)
	}

	// Verify labels
	if sts.Labels[constants.LabelComponent] != constants.ComponentPostgres {
		t.Errorf("expected component label %q, got %q", constants.ComponentPostgres, sts.Labels[constants.LabelComponent])
	}

	// Verify replicas
	if *sts.Spec.Replicas != 1 {
		t.Errorf("expected 1 replica, got %d", *sts.Spec.Replicas)
	}

	// Verify container image
	expectedImage := "postgres:16"
	if sts.Spec.Template.Spec.Containers[0].Image != expectedImage {
		t.Errorf("expected image %q, got %q", expectedImage, sts.Spec.Template.Spec.Containers[0].Image)
	}

	// Verify container port
	if sts.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort != PostgresPort {
		t.Errorf("expected port %d, got %d", PostgresPort, sts.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
	}

	// Verify volume claim template
	if len(sts.Spec.VolumeClaimTemplates) != 1 {
		t.Fatalf("expected 1 volume claim template, got %d", len(sts.Spec.VolumeClaimTemplates))
	}
	if sts.Spec.VolumeClaimTemplates[0].Name != "data" {
		t.Errorf("expected volume claim template name 'data', got %q", sts.Spec.VolumeClaimTemplates[0].Name)
	}

	// Verify storage class
	if *sts.Spec.VolumeClaimTemplates[0].Spec.StorageClassName != "standard" {
		t.Errorf("expected storage class 'standard', got %q", *sts.Spec.VolumeClaimTemplates[0].Spec.StorageClassName)
	}
}

func TestStatefulSetDefaultVersion(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Database: popsignerv1.DatabaseSpec{
				Managed: true,
				// Version not set
				Storage: popsignerv1.StorageSpec{
					Size: resource.MustParse("10Gi"),
				},
			},
		},
	}

	sts := StatefulSet(cluster)

	// Verify default version is used
	expectedImage := "postgres:" + constants.DefaultPostgresVersion
	if sts.Spec.Template.Spec.Containers[0].Image != expectedImage {
		t.Errorf("expected image %q, got %q", expectedImage, sts.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestStatefulSetDefaultReplicas(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Database: popsignerv1.DatabaseSpec{
				Managed: true,
				Version: "16",
				// Replicas not set (0)
				Storage: popsignerv1.StorageSpec{
					Size: resource.MustParse("10Gi"),
				},
			},
		},
	}

	sts := StatefulSet(cluster)

	// Verify default replicas is used
	if *sts.Spec.Replicas != 1 {
		t.Errorf("expected 1 replica, got %d", *sts.Spec.Replicas)
	}
}

func TestService(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Database: popsignerv1.DatabaseSpec{
				Managed: true,
				Version: "16",
			},
		},
	}

	svc := Service(cluster)

	// Verify name
	expectedName := "test-cluster-postgres"
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
	if svc.Spec.Ports[0].Port != PostgresPort {
		t.Errorf("expected port %d, got %d", PostgresPort, svc.Spec.Ports[0].Port)
	}
}

func TestCredentialsSecret(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Database: popsignerv1.DatabaseSpec{
				Managed: true,
				Version: "16",
			},
		},
	}

	password := "test-password"
	secret := CredentialsSecret(cluster, password)

	// Verify name
	expectedName := "test-cluster-postgres-credentials"
	if secret.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, secret.Name)
	}

	// Verify namespace
	if secret.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", secret.Namespace)
	}

	// Verify data keys exist
	if secret.StringData["username"] != "banhbaoring" {
		t.Errorf("expected username 'banhbaoring', got %q", secret.StringData["username"])
	}
	if secret.StringData["password"] != password {
		t.Errorf("expected password %q, got %q", password, secret.StringData["password"])
	}
	if secret.StringData["database"] != "banhbaoring" {
		t.Errorf("expected database 'banhbaoring', got %q", secret.StringData["database"])
	}

	// Verify URL contains password and correct host
	expectedURLPart := "postgres://banhbaoring:test-password@test-cluster-postgres:5432/banhbaoring"
	if secret.StringData["url"] != expectedURLPart+"?sslmode=disable" {
		t.Errorf("expected URL containing %q, got %q", expectedURLPart, secret.StringData["url"])
	}
}

func TestInitConfigMap(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Database: popsignerv1.DatabaseSpec{
				Managed: true,
				Version: "16",
			},
		},
	}

	cm := InitConfigMap(cluster)

	// Verify name
	expectedName := "test-cluster-postgres-init"
	if cm.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, cm.Name)
	}

	// Verify namespace
	if cm.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", cm.Namespace)
	}

	// Verify init SQL exists
	if _, ok := cm.Data["01-schema.sql"]; !ok {
		t.Error("expected '01-schema.sql' in ConfigMap data")
	}

	// Verify SQL contains expected tables
	sql := cm.Data["01-schema.sql"]
	expectedTables := []string{
		"organizations",
		"users",
		"org_members",
		"namespaces",
		"api_keys",
		"audit_logs",
	}
	for _, table := range expectedTables {
		if !containsString(sql, table) {
			t.Errorf("expected SQL to contain table %q", table)
		}
	}
}

func TestMigrationJob(t *testing.T) {
	cluster := &popsignerv1.POPSignerCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: popsignerv1.POPSignerClusterSpec{
			Database: popsignerv1.DatabaseSpec{
				Managed: true,
			},
		},
	}

	job := MigrationJob(cluster, "1.0.0")

	// Verify name
	expectedName := "test-cluster-migrate"
	if job.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, job.Name)
	}

	// Verify namespace
	if job.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", job.Namespace)
	}

	// Verify backoff limit
	if *job.Spec.BackoffLimit != 3 {
		t.Errorf("expected backoff limit 3, got %d", *job.Spec.BackoffLimit)
	}

	// Verify image
	expectedImage := "popsigner/control-plane:1.0.0"
	if job.Spec.Template.Spec.Containers[0].Image != expectedImage {
		t.Errorf("expected image %q, got %q", expectedImage, job.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestStorageClassPtr(t *testing.T) {
	// Test empty string
	result := storageClassPtr("")
	if result != nil {
		t.Error("expected nil for empty string")
	}

	// Test non-empty string
	result = storageClassPtr("standard")
	if result == nil {
		t.Fatal("expected non-nil for non-empty string")
	}
	if *result != "standard" {
		t.Errorf("expected 'standard', got %q", *result)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
