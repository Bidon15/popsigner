// Package redis provides Redis resource builders for the BanhBaoRing operator.
package redis

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	banhbaoringv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
	"github.com/Bidon15/banhbaoring/operator/internal/constants"
)

const (
	RedisImage = "redis"
	RedisPort  = 6379
)

// StatefulSet builds the Redis StatefulSet (standalone mode)
func StatefulSet(cluster *banhbaoringv1.BanhBaoRingCluster) *appsv1.StatefulSet {
	spec := cluster.Spec.Redis
	name := fmt.Sprintf("%s-redis", cluster.Name)
	labels := constants.Labels(cluster.Name, constants.ComponentRedis, spec.Version)

	replicas := spec.Replicas
	if replicas == 0 {
		replicas = 1
	}

	version := spec.Version
	if version == "" {
		version = constants.DefaultRedisVersion
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: name,
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "redis",
							Image: fmt.Sprintf("%s:%s-alpine", RedisImage, version),
							Command: []string{
								"redis-server",
								"--appendonly", "yes",
								"--maxmemory", "256mb",
								"--maxmemory-policy", "allkeys-lru",
							},
							Ports: []corev1.ContainerPort{
								{Name: "redis", ContainerPort: RedisPort},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "data", MountPath: "/data"},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"redis-cli", "ping"},
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{"redis-cli", "ping"},
									},
								},
								InitialDelaySeconds: 15,
								PeriodSeconds:       20,
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "data",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: spec.Storage.Size,
							},
						},
						StorageClassName: storageClassPtr(spec.Storage.StorageClass),
					},
				},
			},
		},
	}
}

// Service builds the Redis Service
func Service(cluster *banhbaoringv1.BanhBaoRingCluster) *corev1.Service {
	name := fmt.Sprintf("%s-redis", cluster.Name)
	labels := constants.Labels(cluster.Name, constants.ComponentRedis, cluster.Spec.Redis.Version)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: labels,
			Ports: []corev1.ServicePort{
				{Name: "redis", Port: RedisPort, TargetPort: intstr.FromInt(RedisPort)},
			},
		},
	}
}

// ConnectionSecret builds the Redis connection Secret
func ConnectionSecret(cluster *banhbaoringv1.BanhBaoRingCluster) *corev1.Secret {
	name := fmt.Sprintf("%s-redis", cluster.Name)
	labels := constants.Labels(cluster.Name, constants.ComponentRedis, cluster.Spec.Redis.Version)

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-connection",
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"url": fmt.Sprintf("redis://%s:6379", name),
		},
	}
}

func storageClassPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
