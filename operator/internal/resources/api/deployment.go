// Package api provides API resource builders for the BanhBaoRing operator.
package api

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
	APIImage = "banhbaoring/control-plane"
	APIPort  = 8080
)

// Deployment builds the API Deployment.
func Deployment(cluster *banhbaoringv1.BanhBaoRingCluster) *appsv1.Deployment {
	spec := cluster.Spec.API
	name := fmt.Sprintf("%s-api", cluster.Name)

	version := spec.Version
	if version == "" {
		version = constants.DefaultAPIVersion
	}

	labels := constants.Labels(cluster.Name, constants.ComponentAPI, version)

	replicas := spec.Replicas
	if replicas == 0 {
		replicas = int32(constants.DefaultAPIReplicas)
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: constants.SelectorLabels(cluster.Name, constants.ComponentAPI),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "api",
							Image: fmt.Sprintf("%s:%s", APIImage, version),
							Ports: []corev1.ContainerPort{
								{Name: "http", ContainerPort: APIPort, Protocol: corev1.ProtocolTCP},
							},
							Env: buildEnv(cluster),
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/health",
										Port:   intstr.FromInt(APIPort),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/health",
										Port:   intstr.FromInt(APIPort),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 15,
								PeriodSeconds:       20,
							},
							Resources: mergeResources(spec.Resources),
						},
					},
				},
			},
		},
	}
}

// buildEnv creates environment variables for the API container.
func buildEnv(cluster *banhbaoringv1.BanhBaoRingCluster) []corev1.EnvVar {
	dbSecret := fmt.Sprintf("%s-postgres-credentials", cluster.Name)
	redisSecret := fmt.Sprintf("%s-redis-connection", cluster.Name)
	openbaoSvc := fmt.Sprintf("%s-openbao-active", cluster.Name)

	return []corev1.EnvVar{
		{Name: "DATABASE_URL", ValueFrom: secretRef(dbSecret, "url")},
		{Name: "REDIS_URL", ValueFrom: secretRef(redisSecret, "url")},
		{Name: "OPENBAO_ADDR", Value: fmt.Sprintf("https://%s:8200", openbaoSvc)},
		{Name: "OPENBAO_TOKEN", ValueFrom: secretRef(cluster.Name+"-openbao-root", "token")},
	}
}

// secretRef creates a secret key reference for environment variables.
func secretRef(name, key string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{Name: name},
			Key:                  key,
		},
	}
}

// mergeResources returns resource requirements with defaults.
func mergeResources(override corev1.ResourceRequirements) corev1.ResourceRequirements {
	defaults := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	}

	if override.Requests != nil || override.Limits != nil {
		return override
	}
	return defaults
}
