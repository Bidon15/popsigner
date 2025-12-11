// Package resources provides helpers for building Kubernetes resources.
package resources

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/Bidon15/banhbaoring/operator/internal/constants"
)

// ResourceName generates a resource name for a component.
func ResourceName(clusterName, component string) string {
	return clusterName + "-" + component
}

// Labels generates standard labels for a component.
func Labels(clusterName, component, version string) map[string]string {
	return constants.Labels(clusterName, component, version)
}

// SelectorLabels generates selector labels for a component.
func SelectorLabels(clusterName, component string) map[string]string {
	return constants.SelectorLabels(clusterName, component)
}

// ObjectMeta creates standard ObjectMeta for a resource.
func ObjectMeta(name, namespace string, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
		Labels:    labels,
	}
}

// ServiceSpec creates a ClusterIP service spec.
func ServiceSpec(selector map[string]string, port int32, targetPort int) corev1.ServiceSpec {
	return corev1.ServiceSpec{
		Type:     corev1.ServiceTypeClusterIP,
		Selector: selector,
		Ports: []corev1.ServicePort{
			{
				Port:       port,
				TargetPort: intstr.FromInt(targetPort),
				Protocol:   corev1.ProtocolTCP,
			},
		},
	}
}

// HeadlessServiceSpec creates a headless service spec for StatefulSets.
func HeadlessServiceSpec(selector map[string]string, port int32, targetPort int) corev1.ServiceSpec {
	spec := ServiceSpec(selector, port, targetPort)
	spec.ClusterIP = corev1.ClusterIPNone
	return spec
}

// PersistentVolumeClaim creates a PVC template for StatefulSets.
func PersistentVolumeClaim(name string, storageClass string, size string) corev1.PersistentVolumeClaim {
	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(size),
				},
			},
		},
	}

	if storageClass != "" {
		pvc.Spec.StorageClassName = &storageClass
	}

	return pvc
}

// ContainerPort creates a container port definition.
func ContainerPort(name string, port int32) corev1.ContainerPort {
	return corev1.ContainerPort{
		Name:          name,
		ContainerPort: port,
		Protocol:      corev1.ProtocolTCP,
	}
}

// EnvVar creates an environment variable from a value.
func EnvVar(name, value string) corev1.EnvVar {
	return corev1.EnvVar{
		Name:  name,
		Value: value,
	}
}

// EnvVarFromSecret creates an environment variable from a secret key.
func EnvVarFromSecret(name, secretName, key string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
				Key: key,
			},
		},
	}
}

// EnvVarFromConfigMap creates an environment variable from a ConfigMap key.
func EnvVarFromConfigMap(name, configMapName, key string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
				Key: key,
			},
		},
	}
}

// VolumeMount creates a volume mount.
func VolumeMount(name, mountPath string, readOnly bool) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
		ReadOnly:  readOnly,
	}
}

// SecretVolume creates a volume from a secret.
func SecretVolume(name, secretName string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}
}

// ConfigMapVolume creates a volume from a ConfigMap.
func ConfigMapVolume(name, configMapName string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: configMapName,
				},
			},
		},
	}
}

// EmptyDirVolume creates an emptyDir volume.
func EmptyDirVolume(name string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

// HTTPGetProbe creates an HTTP GET probe.
func HTTPGetProbe(path string, port int, initialDelay, period int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   path,
				Port:   intstr.FromInt(port),
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: initialDelay,
		PeriodSeconds:       period,
	}
}

// TCPSocketProbe creates a TCP socket probe.
func TCPSocketProbe(port int, initialDelay, period int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(port),
			},
		},
		InitialDelaySeconds: initialDelay,
		PeriodSeconds:       period,
	}
}

// DefaultResourceRequirements returns default resource requirements.
func DefaultResourceRequirements() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
	}
}

// MergeResourceRequirements merges resource requirements, preferring override values.
func MergeResourceRequirements(base, override corev1.ResourceRequirements) corev1.ResourceRequirements {
	result := base.DeepCopy()

	if override.Requests != nil {
		if result.Requests == nil {
			result.Requests = make(corev1.ResourceList)
		}
		for k, v := range override.Requests {
			result.Requests[k] = v
		}
	}

	if override.Limits != nil {
		if result.Limits == nil {
			result.Limits = make(corev1.ResourceList)
		}
		for k, v := range override.Limits {
			result.Limits[k] = v
		}
	}

	return *result
}
