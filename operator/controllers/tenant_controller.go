package controllers

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	banhbaoringv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
)

// TenantReconciler reconciles a BanhBaoRingTenant object
type TenantReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=banhbaoring.io,resources=banhbaoringtenants,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=banhbaoring.io,resources=banhbaoringtenants/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=banhbaoring.io,resources=banhbaoringtenants/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop
func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling BanhBaoRingTenant", "name", req.Name)

	// Fetch the tenant
	tenant := &banhbaoringv1.BanhBaoRingTenant{}
	if err := r.Get(ctx, req.NamespacedName, tenant); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// TODO: Implement tenant reconciliation
	// Step 1: Verify parent cluster exists and is ready
	// Step 2: Create OpenBao namespace for tenant
	// Step 3: Configure quotas and policies
	// Step 4: Create initial admin user
	// Step 5: Update status

	// Requeue after 60s for periodic reconciliation
	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&banhbaoringv1.BanhBaoRingTenant{}).
		Complete(r)
}
