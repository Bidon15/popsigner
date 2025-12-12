package controllers

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/log"

	popsignerv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
	"github.com/Bidon15/banhbaoring/operator/internal/constants"
	"github.com/Bidon15/banhbaoring/operator/internal/resources/monitoring"
)

// reconcileMonitoring handles monitoring stack resources (Prometheus, Grafana, alerts).
func (r *ClusterReconciler) reconcileMonitoring(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	if !cluster.Spec.Monitoring.Enabled {
		return nil
	}

	log := log.FromContext(ctx)
	log.Info("Reconciling monitoring stack")

	// 1. Create/update Prometheus CR
	prom := monitoring.Prometheus(cluster)
	if err := r.createOrUpdate(ctx, cluster, prom); err != nil {
		return fmt.Errorf("failed to reconcile Prometheus: %w", err)
	}

	// 2. Create/update Prometheus Service
	promSvc := monitoring.PrometheusService(cluster)
	if err := r.createOrUpdate(ctx, cluster, promSvc); err != nil {
		return fmt.Errorf("failed to reconcile Prometheus service: %w", err)
	}

	// 3. Create ServiceMonitors for all components
	components := []struct {
		name string
		port int
	}{
		{constants.ComponentOpenBao, constants.PortOpenBao},
		{constants.ComponentAPI, constants.PortAPI},
		{constants.ComponentDashboard, constants.PortDashboard},
	}

	for _, component := range components {
		sm := monitoring.ServiceMonitor(cluster, component.name, component.port)
		if err := r.createOrUpdate(ctx, cluster, sm); err != nil {
			log.Error(err, "Failed to create ServiceMonitor", "component", component.name)
			// Continue with other components instead of failing
		}
	}

	// 4. Create/update Alert rules
	if cluster.Spec.Monitoring.Alerting.Enabled {
		rules := monitoring.PrometheusRules(cluster)
		if err := r.createOrUpdate(ctx, cluster, rules); err != nil {
			return fmt.Errorf("failed to reconcile PrometheusRules: %w", err)
		}
	}

	// 5. Create/update Grafana (if enabled)
	if cluster.Spec.Monitoring.Grafana.Enabled {
		if err := r.reconcileGrafana(ctx, cluster); err != nil {
			return fmt.Errorf("failed to reconcile Grafana: %w", err)
		}

		// Update Grafana endpoint in status
		if cluster.Spec.Domain != "" {
			cluster.Status.Endpoints.Grafana = fmt.Sprintf("https://grafana.%s", cluster.Spec.Domain)
		}
	}

	return nil
}

// reconcileGrafana handles Grafana resources.
func (r *ClusterReconciler) reconcileGrafana(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	log := log.FromContext(ctx)
	log.Info("Reconciling Grafana")

	// 1. Create/update datasources ConfigMap
	datasourceCM := monitoring.DatasourcesConfigMap(cluster)
	if err := r.createOrUpdate(ctx, cluster, datasourceCM); err != nil {
		return fmt.Errorf("failed to reconcile Grafana datasources ConfigMap: %w", err)
	}

	// 2. Create/update dashboards ConfigMap
	dashboardCM := monitoring.DashboardsConfigMap(cluster)
	if err := r.createOrUpdate(ctx, cluster, dashboardCM); err != nil {
		return fmt.Errorf("failed to reconcile Grafana dashboards ConfigMap: %w", err)
	}

	// 3. Create/update Grafana Deployment
	deployment := monitoring.GrafanaDeployment(cluster)
	if err := r.createOrUpdate(ctx, cluster, deployment); err != nil {
		return fmt.Errorf("failed to reconcile Grafana deployment: %w", err)
	}

	// 4. Create/update Grafana Service
	svc := monitoring.GrafanaService(cluster)
	if err := r.createOrUpdate(ctx, cluster, svc); err != nil {
		return fmt.Errorf("failed to reconcile Grafana service: %w", err)
	}

	return nil
}
