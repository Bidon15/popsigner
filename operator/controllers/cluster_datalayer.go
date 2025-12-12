package controllers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"

	popsignerv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
	"github.com/Bidon15/banhbaoring/operator/internal/conditions"
	"github.com/Bidon15/banhbaoring/operator/internal/resources/database"
	"github.com/Bidon15/banhbaoring/operator/internal/resources/redis"
)

// reconcilePostgreSQL handles PostgreSQL resources
func (r *ClusterReconciler) reconcilePostgreSQL(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	log := log.FromContext(ctx)

	if !cluster.Spec.Database.Managed {
		log.Info("Using external database, skipping PostgreSQL deployment")
		return nil
	}

	log.Info("Reconciling PostgreSQL")

	name := fmt.Sprintf("%s-postgres", cluster.Name)

	// 1. Ensure credentials secret exists
	credSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: name + "-credentials", Namespace: cluster.Namespace}, credSecret); err != nil {
		if errors.IsNotFound(err) {
			// Generate new password
			password, err := generatePassword(32)
			if err != nil {
				return fmt.Errorf("failed to generate password: %w", err)
			}

			credSecret = database.CredentialsSecret(cluster, password)
			if err := r.createOrUpdate(ctx, cluster, credSecret); err != nil {
				return fmt.Errorf("failed to create credentials secret: %w", err)
			}
		} else {
			return err
		}
	}

	// 2. Create init ConfigMap
	initCM := database.InitConfigMap(cluster)
	if err := r.createOrUpdate(ctx, cluster, initCM); err != nil {
		return fmt.Errorf("failed to reconcile init configmap: %w", err)
	}

	// 3. Create Service
	svc := database.Service(cluster)
	if err := r.createOrUpdate(ctx, cluster, svc); err != nil {
		return fmt.Errorf("failed to reconcile service: %w", err)
	}

	// 4. Create StatefulSet
	sts := database.StatefulSet(cluster)
	if err := r.createOrUpdate(ctx, cluster, sts); err != nil {
		return fmt.Errorf("failed to reconcile statefulset: %w", err)
	}

	return nil
}

// reconcileRedis handles Redis resources
func (r *ClusterReconciler) reconcileRedis(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	log := log.FromContext(ctx)

	if !cluster.Spec.Redis.Managed {
		log.Info("Using external Redis, skipping deployment")
		return nil
	}

	log.Info("Reconciling Redis")

	// 1. Create connection Secret
	connSecret := redis.ConnectionSecret(cluster)
	if err := r.createOrUpdate(ctx, cluster, connSecret); err != nil {
		return fmt.Errorf("failed to reconcile connection secret: %w", err)
	}

	// 2. Create Service
	svc := redis.Service(cluster)
	if err := r.createOrUpdate(ctx, cluster, svc); err != nil {
		return fmt.Errorf("failed to reconcile service: %w", err)
	}

	// 3. Create StatefulSet
	if cluster.Spec.Redis.Mode == "cluster" {
		// TODO: Implement Redis Cluster mode
		log.Info("Redis Cluster mode not yet implemented, falling back to standalone")
	}

	sts := redis.StatefulSet(cluster)
	if err := r.createOrUpdate(ctx, cluster, sts); err != nil {
		return fmt.Errorf("failed to reconcile statefulset: %w", err)
	}

	return nil
}

// isDataLayerReady checks if PostgreSQL and Redis are ready
func (r *ClusterReconciler) isDataLayerReady(ctx context.Context, cluster *popsignerv1.POPSignerCluster) bool {
	// Check PostgreSQL
	if cluster.Spec.Database.Managed {
		name := fmt.Sprintf("%s-postgres", cluster.Name)
		sts := &appsv1.StatefulSet{}
		if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Namespace}, sts); err != nil {
			return false
		}
		if sts.Status.ReadyReplicas < 1 {
			return false
		}
	}

	// Check Redis
	if cluster.Spec.Redis.Managed {
		name := fmt.Sprintf("%s-redis", cluster.Name)
		sts := &appsv1.StatefulSet{}
		if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Namespace}, sts); err != nil {
			return false
		}
		if sts.Status.ReadyReplicas < 1 {
			return false
		}
	}

	return true
}

// updateDatabaseStatus updates the cluster status with database info
func (r *ClusterReconciler) updateDatabaseStatus(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	// PostgreSQL status
	if cluster.Spec.Database.Managed {
		name := fmt.Sprintf("%s-postgres", cluster.Name)
		sts := &appsv1.StatefulSet{}
		if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Namespace}, sts); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
			cluster.Status.Database = popsignerv1.ComponentStatus{
				Ready:   false,
				Message: "StatefulSet not found",
			}
		} else {
			ready := sts.Status.ReadyReplicas >= 1
			cluster.Status.Database = popsignerv1.ComponentStatus{
				Ready:    ready,
				Version:  cluster.Spec.Database.Version,
				Replicas: fmt.Sprintf("%d/%d", sts.Status.ReadyReplicas, *sts.Spec.Replicas),
			}
		}
	} else {
		cluster.Status.Database = popsignerv1.ComponentStatus{
			Ready:   true,
			Message: "Using external database",
		}
	}

	// Redis status
	if cluster.Spec.Redis.Managed {
		name := fmt.Sprintf("%s-redis", cluster.Name)
		sts := &appsv1.StatefulSet{}
		if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Namespace}, sts); err != nil {
			if !errors.IsNotFound(err) {
				return err
			}
			cluster.Status.Redis = popsignerv1.ComponentStatus{
				Ready:   false,
				Message: "StatefulSet not found",
			}
		} else {
			ready := sts.Status.ReadyReplicas >= 1
			cluster.Status.Redis = popsignerv1.ComponentStatus{
				Ready:    ready,
				Version:  cluster.Spec.Redis.Version,
				Replicas: fmt.Sprintf("%d/%d", sts.Status.ReadyReplicas, *sts.Spec.Replicas),
			}
		}
	} else {
		cluster.Status.Redis = popsignerv1.ComponentStatus{
			Ready:   true,
			Message: "Using external Redis",
		}
	}

	// Update condition
	dbReady := cluster.Status.Database.Ready && cluster.Status.Redis.Ready
	condStatus := metav1.ConditionFalse
	reason := "NotReady"
	message := "Waiting for data layer"

	if dbReady {
		condStatus = metav1.ConditionTrue
		reason = "Ready"
		message = "Database and Redis are ready"
	}

	conditions.SetCondition(&cluster.Status.Conditions, conditions.TypeDatabaseReady, condStatus, reason, message)

	return nil
}

// generatePassword creates a secure random password
func generatePassword(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
