// Package database provides PostgreSQL resource builders for the BanhBaoRing operator.
package database

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
	PostgresImage = "postgres"
	PostgresPort  = 5432
)

// StatefulSet builds the PostgreSQL StatefulSet
func StatefulSet(cluster *banhbaoringv1.BanhBaoRingCluster) *appsv1.StatefulSet {
	spec := cluster.Spec.Database
	name := fmt.Sprintf("%s-postgres", cluster.Name)
	labels := constants.Labels(cluster.Name, constants.ComponentPostgres, spec.Version)

	replicas := spec.Replicas
	if replicas == 0 {
		replicas = 1
	}

	version := spec.Version
	if version == "" {
		version = constants.DefaultPostgresVersion
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
							Name:  "postgres",
							Image: fmt.Sprintf("%s:%s", PostgresImage, version),
							Ports: []corev1.ContainerPort{
								{Name: "postgres", ContainerPort: PostgresPort},
							},
							Env: []corev1.EnvVar{
								{
									Name: "POSTGRES_USER",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: name + "-credentials"},
											Key:                  "username",
										},
									},
								},
								{
									Name: "POSTGRES_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: name + "-credentials"},
											Key:                  "password",
										},
									},
								},
								{
									Name:  "POSTGRES_DB",
									Value: "banhbaoring",
								},
								{
									Name:  "PGDATA",
									Value: "/var/lib/postgresql/data/pgdata",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "data", MountPath: "/var/lib/postgresql/data"},
								{Name: "init-scripts", MountPath: "/docker-entrypoint-initdb.d"},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"pg_isready",
											"-U", "banhbaoring",
											"-d", "banhbaoring",
										},
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"pg_isready",
											"-U", "banhbaoring",
											"-d", "banhbaoring",
										},
									},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       30,
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("250m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1"),
									corev1.ResourceMemory: resource.MustParse("1Gi"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "init-scripts",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: name + "-init",
									},
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

// Service builds the PostgreSQL Service
func Service(cluster *banhbaoringv1.BanhBaoRingCluster) *corev1.Service {
	name := fmt.Sprintf("%s-postgres", cluster.Name)
	labels := constants.Labels(cluster.Name, constants.ComponentPostgres, cluster.Spec.Database.Version)

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
				{Name: "postgres", Port: PostgresPort, TargetPort: intstr.FromInt(PostgresPort)},
			},
		},
	}
}

// CredentialsSecret builds the PostgreSQL credentials Secret
func CredentialsSecret(cluster *banhbaoringv1.BanhBaoRingCluster, password string) *corev1.Secret {
	name := fmt.Sprintf("%s-postgres", cluster.Name)
	labels := constants.Labels(cluster.Name, constants.ComponentPostgres, cluster.Spec.Database.Version)

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-credentials",
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"username": "banhbaoring",
			"password": password,
			"database": "banhbaoring",
			"url":      fmt.Sprintf("postgres://banhbaoring:%s@%s:5432/banhbaoring?sslmode=disable", password, name),
		},
	}
}

// InitConfigMap builds the init scripts ConfigMap
func InitConfigMap(cluster *banhbaoringv1.BanhBaoRingCluster) *corev1.ConfigMap {
	name := fmt.Sprintf("%s-postgres", cluster.Name)
	labels := constants.Labels(cluster.Name, constants.ComponentPostgres, cluster.Spec.Database.Version)

	// Schema from control-plane migrations
	initSQL := `
-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Organizations table
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    plan VARCHAR(50) DEFAULT 'free',
    stripe_customer_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    name VARCHAR(255),
    email_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Organization members
CREATE TABLE IF NOT EXISTS org_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(org_id, user_id)
);

-- Namespaces
CREATE TABLE IF NOT EXISTS namespaces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(org_id, name)
);

-- API Keys
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(12) NOT NULL,
    scopes TEXT[],
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Audit logs
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id),
    action VARCHAR(255) NOT NULL,
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),
    metadata JSONB,
    ip_address INET,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_audit_logs_org_id ON audit_logs(org_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys(key_prefix);
`

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-init",
			Namespace: cluster.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"01-schema.sql": initSQL,
		},
	}
}

func storageClassPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
