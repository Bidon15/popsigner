package controllers

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
	"github.com/Bidon15/popsigner/operator/internal/conditions"
	"github.com/Bidon15/popsigner/operator/internal/constants"
	"github.com/Bidon15/popsigner/operator/internal/resources"
	"github.com/Bidon15/popsigner/operator/internal/resources/api"
	"github.com/Bidon15/popsigner/operator/internal/resources/dashboard"
	"github.com/Bidon15/popsigner/operator/internal/resources/rpcgateway"
)

// reconcileAPI handles API resources.
func (r *ClusterReconciler) reconcileAPI(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	log := log.FromContext(ctx)
	log.Info("Reconciling API")

	// 1. Create/update Deployment
	deployment := api.Deployment(cluster)
	if err := r.createOrUpdate(ctx, cluster, deployment); err != nil {
		return fmt.Errorf("failed to reconcile API deployment: %w", err)
	}

	// 2. Create/update Service
	svc := api.Service(cluster)
	if err := r.createOrUpdate(ctx, cluster, svc); err != nil {
		return fmt.Errorf("failed to reconcile API service: %w", err)
	}

	// 3. Create/update HPA (if autoscaling enabled)
	if cluster.Spec.API.Autoscaling.Enabled {
		hpa := api.HPA(cluster)
		if err := r.createOrUpdate(ctx, cluster, hpa); err != nil {
			return fmt.Errorf("failed to reconcile API HPA: %w", err)
		}
	}

	return nil
}

// reconcileDashboard handles Dashboard resources.
// NOTE: The dashboard is integrated into the control-plane. This function
// is kept for backwards compatibility but is typically not needed when
// dashboard.replicas is 0. The control-plane serves both API and dashboard.
func (r *ClusterReconciler) reconcileDashboard(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	log := log.FromContext(ctx)

	// Skip if dashboard replicas is 0 (dashboard is integrated into control-plane)
	if cluster.Spec.Dashboard.Replicas == 0 {
		log.Info("Dashboard replicas is 0, skipping (dashboard is served by control-plane)")
		return nil
	}

	log.Info("Reconciling Dashboard")

	// 1. Create/update Deployment
	deployment := dashboard.Deployment(cluster)
	if err := r.createOrUpdate(ctx, cluster, deployment); err != nil {
		return fmt.Errorf("failed to reconcile Dashboard deployment: %w", err)
	}

	// 2. Create/update Service
	svc := dashboard.Service(cluster)
	if err := r.createOrUpdate(ctx, cluster, svc); err != nil {
		return fmt.Errorf("failed to reconcile Dashboard service: %w", err)
	}

	return nil
}

// isAPIReady checks if API pods are ready.
func (r *ClusterReconciler) isAPIReady(ctx context.Context, cluster *popsignerv1.POPSignerCluster) bool {
	name := resources.ResourceName(cluster.Name, constants.ComponentAPI)

	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Namespace}, deployment); err != nil {
		return false
	}

	expectedReplicas := cluster.Spec.API.Replicas
	if expectedReplicas == 0 {
		expectedReplicas = int32(constants.DefaultAPIReplicas)
	}

	return deployment.Status.ReadyReplicas >= expectedReplicas
}

// isDashboardReady checks if Dashboard pods are ready.
// Returns true if dashboard is disabled (replicas=0) or all pods are ready.
func (r *ClusterReconciler) isDashboardReady(ctx context.Context, cluster *popsignerv1.POPSignerCluster) bool {
	// Dashboard is disabled (served by control-plane)
	if cluster.Spec.Dashboard.Replicas == 0 {
		return true
	}

	name := resources.ResourceName(cluster.Name, constants.ComponentDashboard)

	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Namespace}, deployment); err != nil {
		return false
	}

	expectedReplicas := cluster.Spec.Dashboard.Replicas
	if expectedReplicas == 0 {
		expectedReplicas = int32(constants.DefaultDashboardReplicas)
	}

	return deployment.Status.ReadyReplicas >= expectedReplicas
}

// updateAPIStatus updates the cluster status with API info.
func (r *ClusterReconciler) updateAPIStatus(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	name := resources.ResourceName(cluster.Name, constants.ComponentAPI)

	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Namespace}, deployment); err != nil {
		if errors.IsNotFound(err) {
			cluster.Status.API = popsignerv1.ComponentStatus{
				Ready:   false,
				Message: "Deployment not found",
			}
			return nil
		}
		return err
	}

	expectedReplicas := *deployment.Spec.Replicas
	ready := deployment.Status.ReadyReplicas >= expectedReplicas

	version := cluster.Spec.API.Version
	if version == "" {
		version = constants.DefaultAPIVersion
	}

	cluster.Status.API = popsignerv1.ComponentStatus{
		Ready:    ready,
		Version:  version,
		Replicas: fmt.Sprintf("%d/%d", deployment.Status.ReadyReplicas, expectedReplicas),
	}

	condStatus := metav1.ConditionFalse
	reason := conditions.ReasonNotReady
	message := fmt.Sprintf("Waiting for API pods: %d/%d ready", deployment.Status.ReadyReplicas, expectedReplicas)

	if ready {
		condStatus = metav1.ConditionTrue
		reason = conditions.ReasonReady
		message = "API is ready"
	}

	conditions.SetCondition(&cluster.Status.Conditions, conditions.TypeAPIReady, condStatus, reason, message)

	// Update endpoints
	if ready && cluster.Spec.Domain != "" {
		cluster.Status.Endpoints.API = fmt.Sprintf("https://api.%s", cluster.Spec.Domain)
	}

	return nil
}

// updateDashboardStatus updates the cluster status with Dashboard info.
func (r *ClusterReconciler) updateDashboardStatus(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	// Dashboard is disabled/integrated into control-plane
	if cluster.Spec.Dashboard.Replicas == 0 {
		cluster.Status.Dashboard = popsignerv1.ComponentStatus{
			Ready:   true,
			Message: "Integrated into control-plane",
		}
		conditions.SetCondition(&cluster.Status.Conditions, conditions.TypeDashboardReady,
			metav1.ConditionTrue, conditions.ReasonReady, "Dashboard is served by control-plane")

		// Dashboard endpoint is same as API when integrated
		if cluster.Spec.Domain != "" {
			cluster.Status.Endpoints.Dashboard = fmt.Sprintf("https://api.%s", cluster.Spec.Domain)
		}
		return nil
	}

	name := resources.ResourceName(cluster.Name, constants.ComponentDashboard)

	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Namespace}, deployment); err != nil {
		if errors.IsNotFound(err) {
			cluster.Status.Dashboard = popsignerv1.ComponentStatus{
				Ready:   false,
				Message: "Deployment not found",
			}
			return nil
		}
		return err
	}

	expectedReplicas := *deployment.Spec.Replicas
	ready := deployment.Status.ReadyReplicas >= expectedReplicas

	version := cluster.Spec.Dashboard.Version
	if version == "" {
		version = constants.DefaultDashboardVersion
	}

	cluster.Status.Dashboard = popsignerv1.ComponentStatus{
		Ready:    ready,
		Version:  version,
		Replicas: fmt.Sprintf("%d/%d", deployment.Status.ReadyReplicas, expectedReplicas),
	}

	condStatus := metav1.ConditionFalse
	reason := conditions.ReasonNotReady
	message := fmt.Sprintf("Waiting for Dashboard pods: %d/%d ready", deployment.Status.ReadyReplicas, expectedReplicas)

	if ready {
		condStatus = metav1.ConditionTrue
		reason = conditions.ReasonReady
		message = "Dashboard is ready"
	}

	conditions.SetCondition(&cluster.Status.Conditions, conditions.TypeDashboardReady, condStatus, reason, message)

	// Update endpoints
	if ready && cluster.Spec.Domain != "" {
		cluster.Status.Endpoints.Dashboard = fmt.Sprintf("https://dashboard.%s", cluster.Spec.Domain)
	}

	return nil
}

// isAppsReady checks if both API and Dashboard are ready.
func (r *ClusterReconciler) isAppsReady(ctx context.Context, cluster *popsignerv1.POPSignerCluster) bool {
	return r.isAPIReady(ctx, cluster) && r.isDashboardReady(ctx, cluster)
}

// updateAppsStatus updates the cluster status with both API and Dashboard info.
func (r *ClusterReconciler) updateAppsStatus(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	if err := r.updateAPIStatus(ctx, cluster); err != nil {
		return fmt.Errorf("failed to update API status: %w", err)
	}

	if err := r.updateDashboardStatus(ctx, cluster); err != nil {
		return fmt.Errorf("failed to update Dashboard status: %w", err)
	}

	if err := r.updateRPCGatewayStatus(ctx, cluster); err != nil {
		return fmt.Errorf("failed to update RPC Gateway status: %w", err)
	}

	return nil
}

// reconcileRPCGateway ensures the RPC Gateway is deployed if enabled.
func (r *ClusterReconciler) reconcileRPCGateway(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	log := log.FromContext(ctx)

	if !cluster.Spec.RPCGateway.Enabled {
		// Gateway not enabled, ensure resources are deleted
		return r.cleanupRPCGateway(ctx, cluster)
	}

	log.Info("Reconciling RPC Gateway")

	// Create/Update Deployment
	deployment := rpcgateway.Deployment(cluster)
	if err := r.createOrUpdate(ctx, cluster, deployment); err != nil {
		return fmt.Errorf("failed to reconcile RPC Gateway deployment: %w", err)
	}

	// Create/Update Service
	service := rpcgateway.Service(cluster)
	if err := r.createOrUpdate(ctx, cluster, service); err != nil {
		return fmt.Errorf("failed to reconcile RPC Gateway service: %w", err)
	}

	// Create/Update HPA
	hpa := rpcgateway.HorizontalPodAutoscaler(cluster)
	if err := r.createOrUpdate(ctx, cluster, hpa); err != nil {
		return fmt.Errorf("failed to reconcile RPC Gateway HPA: %w", err)
	}

	log.Info("RPC Gateway reconciled successfully")
	return nil
}

// cleanupRPCGateway removes RPC Gateway resources when disabled.
func (r *ClusterReconciler) cleanupRPCGateway(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	log := log.FromContext(ctx)
	name := resources.ResourceName(cluster.Name, constants.ComponentRPCGateway)

	// Delete HPA
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Namespace}, hpa); err == nil {
		if err := r.Delete(ctx, hpa); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete RPC Gateway HPA: %w", err)
		}
		log.Info("Deleted RPC Gateway HPA")
	}

	// Delete Service
	svc := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Namespace}, svc); err == nil {
		if err := r.Delete(ctx, svc); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete RPC Gateway service: %w", err)
		}
		log.Info("Deleted RPC Gateway Service")
	}

	// Delete Deployment
	deploy := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Namespace}, deploy); err == nil {
		if err := r.Delete(ctx, deploy); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("failed to delete RPC Gateway deployment: %w", err)
		}
		log.Info("Deleted RPC Gateway Deployment")
	}

	return nil
}

// isRPCGatewayReady checks if RPC Gateway pods are ready.
// Returns true if gateway is disabled or all pods are ready.
func (r *ClusterReconciler) isRPCGatewayReady(ctx context.Context, cluster *popsignerv1.POPSignerCluster) bool {
	// Gateway is disabled
	if !cluster.Spec.RPCGateway.Enabled {
		return true
	}

	name := resources.ResourceName(cluster.Name, constants.ComponentRPCGateway)

	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Namespace}, deployment); err != nil {
		return false
	}

	expectedReplicas := cluster.Spec.RPCGateway.Replicas
	if expectedReplicas == 0 {
		expectedReplicas = int32(constants.DefaultRPCGatewayReplicas)
	}

	return deployment.Status.ReadyReplicas >= expectedReplicas
}

// updateRPCGatewayStatus updates the cluster status with RPC Gateway info.
func (r *ClusterReconciler) updateRPCGatewayStatus(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	// RPC Gateway is disabled
	if !cluster.Spec.RPCGateway.Enabled {
		cluster.Status.RPCGateway = popsignerv1.ComponentStatus{
			Ready:   true,
			Message: "Disabled",
		}
		conditions.SetCondition(&cluster.Status.Conditions, conditions.TypeRPCGatewayReady,
			metav1.ConditionTrue, conditions.ReasonDisabled, "RPC Gateway is disabled")
		return nil
	}

	name := resources.ResourceName(cluster.Name, constants.ComponentRPCGateway)

	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Namespace}, deployment); err != nil {
		if errors.IsNotFound(err) {
			cluster.Status.RPCGateway = popsignerv1.ComponentStatus{
				Ready:   false,
				Message: "Deployment not found",
			}
			return nil
		}
		return err
	}

	expectedReplicas := *deployment.Spec.Replicas
	ready := deployment.Status.ReadyReplicas >= expectedReplicas

	version := cluster.Spec.RPCGateway.Version
	if version == "" {
		version = constants.DefaultRPCGatewayVersion
	}

	cluster.Status.RPCGateway = popsignerv1.ComponentStatus{
		Ready:    ready,
		Version:  version,
		Replicas: fmt.Sprintf("%d/%d", deployment.Status.ReadyReplicas, expectedReplicas),
	}

	condStatus := metav1.ConditionFalse
	reason := conditions.ReasonNotReady
	message := fmt.Sprintf("Waiting for RPC Gateway pods: %d/%d ready", deployment.Status.ReadyReplicas, expectedReplicas)

	if ready {
		condStatus = metav1.ConditionTrue
		reason = conditions.ReasonReady
		message = "RPC Gateway is ready"
	}

	conditions.SetCondition(&cluster.Status.Conditions, conditions.TypeRPCGatewayReady, condStatus, reason, message)

	// Update endpoints
	if ready && cluster.Spec.Domain != "" {
		cluster.Status.Endpoints.RPCGateway = fmt.Sprintf("https://rpc.%s", cluster.Spec.Domain)
	}

	return nil
}
