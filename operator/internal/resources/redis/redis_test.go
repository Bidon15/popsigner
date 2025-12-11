package redis

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	banhbaoringv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
	"github.com/Bidon15/banhbaoring/operator/internal/constants"
)

func TestStatefulSet(t *testing.T) {
	cluster := &banhbaoringv1.BanhBaoRingCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: banhbaoringv1.BanhBaoRingClusterSpec{
			Redis: banhbaoringv1.RedisSpec{
				Managed:  true,
				Version:  "7",
				Mode:     "standalone",
				Replicas: 1,
				Storage: banhbaoringv1.StorageSpec{
					Size:         resource.MustParse("5Gi"),
					StorageClass: "standard",
				},
			},
		},
	}

	sts := StatefulSet(cluster)

	// Verify name
	expectedName := "test-cluster-redis"
	if sts.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, sts.Name)
	}

	// Verify namespace
	if sts.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", sts.Namespace)
	}

	// Verify labels
	if sts.Labels[constants.LabelComponent] != constants.ComponentRedis {
		t.Errorf("expected component label %q, got %q", constants.ComponentRedis, sts.Labels[constants.LabelComponent])
	}

	// Verify replicas
	if *sts.Spec.Replicas != 1 {
		t.Errorf("expected 1 replica, got %d", *sts.Spec.Replicas)
	}

	// Verify container image
	expectedImage := "redis:7-alpine"
	if sts.Spec.Template.Spec.Containers[0].Image != expectedImage {
		t.Errorf("expected image %q, got %q", expectedImage, sts.Spec.Template.Spec.Containers[0].Image)
	}

	// Verify container port
	if sts.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort != RedisPort {
		t.Errorf("expected port %d, got %d", RedisPort, sts.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
	}

	// Verify command includes persistence
	cmd := sts.Spec.Template.Spec.Containers[0].Command
	if len(cmd) == 0 {
		t.Fatal("expected redis command to be set")
	}
	if cmd[0] != "redis-server" {
		t.Errorf("expected 'redis-server' command, got %q", cmd[0])
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
	cluster := &banhbaoringv1.BanhBaoRingCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: banhbaoringv1.BanhBaoRingClusterSpec{
			Redis: banhbaoringv1.RedisSpec{
				Managed: true,
				// Version not set
				Storage: banhbaoringv1.StorageSpec{
					Size: resource.MustParse("5Gi"),
				},
			},
		},
	}

	sts := StatefulSet(cluster)

	// Verify default version is used
	expectedImage := "redis:" + constants.DefaultRedisVersion + "-alpine"
	if sts.Spec.Template.Spec.Containers[0].Image != expectedImage {
		t.Errorf("expected image %q, got %q", expectedImage, sts.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestStatefulSetDefaultReplicas(t *testing.T) {
	cluster := &banhbaoringv1.BanhBaoRingCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: banhbaoringv1.BanhBaoRingClusterSpec{
			Redis: banhbaoringv1.RedisSpec{
				Managed: true,
				Version: "7",
				// Replicas not set (0)
				Storage: banhbaoringv1.StorageSpec{
					Size: resource.MustParse("5Gi"),
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
	cluster := &banhbaoringv1.BanhBaoRingCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: banhbaoringv1.BanhBaoRingClusterSpec{
			Redis: banhbaoringv1.RedisSpec{
				Managed: true,
				Version: "7",
			},
		},
	}

	svc := Service(cluster)

	// Verify name
	expectedName := "test-cluster-redis"
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
	if svc.Spec.Ports[0].Port != RedisPort {
		t.Errorf("expected port %d, got %d", RedisPort, svc.Spec.Ports[0].Port)
	}
}

func TestConnectionSecret(t *testing.T) {
	cluster := &banhbaoringv1.BanhBaoRingCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: banhbaoringv1.BanhBaoRingClusterSpec{
			Redis: banhbaoringv1.RedisSpec{
				Managed: true,
				Version: "7",
			},
		},
	}

	secret := ConnectionSecret(cluster)

	// Verify name
	expectedName := "test-cluster-redis-connection"
	if secret.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, secret.Name)
	}

	// Verify namespace
	if secret.Namespace != "default" {
		t.Errorf("expected namespace 'default', got %q", secret.Namespace)
	}

	// Verify URL
	expectedURL := "redis://test-cluster-redis:6379"
	if secret.StringData["url"] != expectedURL {
		t.Errorf("expected URL %q, got %q", expectedURL, secret.StringData["url"])
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
