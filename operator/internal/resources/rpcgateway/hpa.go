package rpcgateway

import (
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
	"github.com/Bidon15/popsigner/operator/internal/constants"
)

// HorizontalPodAutoscaler builds the HPA for the RPC Gateway.
func HorizontalPodAutoscaler(cluster *popsignerv1.POPSignerCluster) *autoscalingv2.HorizontalPodAutoscaler {
	name := fmt.Sprintf("%s-%s", cluster.Name, constants.ComponentRPCGateway)

	version := cluster.Spec.RPCGateway.Version
	if version == "" {
		version = constants.DefaultRPCGatewayVersion
	}

	labels := constants.Labels(cluster.Name, constants.ComponentRPCGateway, version)

	// Default scaling parameters
	minReplicas := int32(constants.DefaultRPCGatewayReplicas)
	maxReplicas := int32(10)
	targetCPU := int32(70)

	// Use configured replicas as minimum if higher
	if cluster.Spec.RPCGateway.Replicas > minReplicas {
		minReplicas = cluster.Spec.RPCGateway.Replicas
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
			Behavior: &autoscalingv2.HorizontalPodAutoscalerBehavior{
				ScaleDown: &autoscalingv2.HPAScalingRules{
					StabilizationWindowSeconds: int32Ptr(300), // 5 minutes
					Policies: []autoscalingv2.HPAScalingPolicy{
						{
							Type:          autoscalingv2.PodsScalingPolicy,
							Value:         1,
							PeriodSeconds: 60,
						},
					},
				},
				ScaleUp: &autoscalingv2.HPAScalingRules{
					StabilizationWindowSeconds: int32Ptr(30),
					Policies: []autoscalingv2.HPAScalingPolicy{
						{
							Type:          autoscalingv2.PodsScalingPolicy,
							Value:         2,
							PeriodSeconds: 30,
						},
						{
							Type:          autoscalingv2.PercentScalingPolicy,
							Value:         100,
							PeriodSeconds: 30,
						},
					},
					SelectPolicy: selectPolicyPtr(autoscalingv2.MaxChangePolicySelect),
				},
			},
		},
	}
}

func int32Ptr(i int32) *int32                                                          { return &i }
func selectPolicyPtr(p autoscalingv2.ScalingPolicySelect) *autoscalingv2.ScalingPolicySelect { return &p }

