package api

import (
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	popsignerv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
	"github.com/Bidon15/banhbaoring/operator/internal/constants"
)

// HPA builds the HorizontalPodAutoscaler for the API.
func HPA(cluster *popsignerv1.POPSignerCluster) *autoscalingv2.HorizontalPodAutoscaler {
	spec := cluster.Spec.API.Autoscaling
	name := fmt.Sprintf("%s-api", cluster.Name)

	version := cluster.Spec.API.Version
	if version == "" {
		version = constants.DefaultAPIVersion
	}

	labels := constants.Labels(cluster.Name, constants.ComponentAPI, version)

	minReplicas := spec.MinReplicas
	if minReplicas == 0 {
		minReplicas = int32(constants.DefaultAPIReplicas)
	}

	maxReplicas := spec.MaxReplicas
	if maxReplicas == 0 {
		maxReplicas = 10
	}

	targetCPU := spec.TargetCPU
	if targetCPU == 0 {
		targetCPU = 70
	}

	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       name,
			},
			MinReplicas: &minReplicas,
			MaxReplicas: maxReplicas,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &targetCPU,
						},
					},
				},
			},
		},
	}
}
