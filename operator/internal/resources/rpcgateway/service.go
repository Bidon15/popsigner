package rpcgateway

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
	"github.com/Bidon15/popsigner/operator/internal/constants"
)

// Service builds the RPC Gateway Service.
func Service(cluster *popsignerv1.POPSignerCluster) *corev1.Service {
	name := fmt.Sprintf("%s-%s", cluster.Name, constants.ComponentRPCGateway)

	version := cluster.Spec.RPCGateway.Version
	if version == "" {
		version = constants.DefaultRPCGatewayVersion
	}

	labels := constants.Labels(cluster.Name, constants.ComponentRPCGateway, version)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   fmt.Sprintf("%d", constants.PortRPCGateway),
			},
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: constants.SelectorLabels(cluster.Name, constants.ComponentRPCGateway),
			Ports: []corev1.ServicePort{
				{
					Name:       "jsonrpc",
					Port:       int32(constants.PortRPCGateway),
					TargetPort: intstr.FromInt(constants.PortRPCGateway),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

