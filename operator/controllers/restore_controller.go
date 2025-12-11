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

// RestoreReconciler reconciles a BanhBaoRingRestore object
type RestoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=banhbaoring.io,resources=banhbaoringrestores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=banhbaoring.io,resources=banhbaoringrestores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=banhbaoring.io,resources=banhbaoringrestores/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *RestoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling BanhBaoRingRestore", "name", req.Name)

	// Fetch the restore
	restore := &banhbaoringv1.BanhBaoRingRestore{}
	if err := r.Get(ctx, req.NamespacedName, restore); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// TODO: Implement restore reconciliation
	// Step 1: Verify parent cluster exists
	// Step 2: Verify backup exists and is complete
	// Step 3: Stop applications if configured
	// Step 4: Restore each component
	// Step 5: Start applications
	// Step 6: Verify restore success

	// Check if restore is complete
	if restore.Status.Phase == "Completed" || restore.Status.Phase == "Failed" {
		return ctrl.Result{}, nil
	}

	// Requeue to monitor progress
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RestoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&banhbaoringv1.BanhBaoRingRestore{}).
		Complete(r)
}
