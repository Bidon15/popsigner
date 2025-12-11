package dashboard

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	banhbaoringv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
	"github.com/Bidon15/banhbaoring/operator/internal/constants"
)

// Service builds the Dashboard Service.
func Service(cluster *banhbaoringv1.BanhBaoRingCluster) *corev1.Service {
	name := fmt.Sprintf("%s-dashboard", cluster.Name)

	version := cluster.Spec.Dashboard.Version
	if version == "" {
		version = constants.DefaultDashboardVersion
	}

	labels := constants.Labels(cluster.Name, constants.ComponentDashboard, version)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: constants.SelectorLabels(cluster.Name, constants.ComponentDashboard),
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       DashboardPort,
					TargetPort: intstr.FromInt(DashboardPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}
