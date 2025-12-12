package database

import (
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	popsignerv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
	"github.com/Bidon15/banhbaoring/operator/internal/constants"
)

// MigrationJob creates a Job to run database migrations
func MigrationJob(cluster *popsignerv1.POPSignerCluster, apiVersion string) *batchv1.Job {
	name := fmt.Sprintf("%s-migrate", cluster.Name)
	labels := constants.Labels(cluster.Name, "migration", apiVersion)

	backoffLimit := int32(3)
	ttlSeconds := int32(300)

	dbSecretName := fmt.Sprintf("%s-postgres-credentials", cluster.Name)

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSeconds,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "migrate",
							Image: fmt.Sprintf("popsigner/control-plane:%s", apiVersion),
							Command: []string{
								"/app/control-plane",
								"migrate",
								"--up",
							},
							Env: []corev1.EnvVar{
								{
									Name: "DATABASE_URL",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: dbSecretName,
											},
											Key: "url",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
