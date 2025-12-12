package controllers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
	"github.com/Bidon15/popsigner/operator/internal/openbao"
	"github.com/Bidon15/popsigner/operator/internal/plans"
)

// TenantReconciler reconciles a POPSignerTenant object
type TenantReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=popsigner.com,resources=popsignertenants,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=popsigner.com,resources=popsignertenants/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=popsigner.com,resources=popsignertenants/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop
func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("Reconciling POPSignerTenant", "name", req.Name)

	// Fetch the tenant
	tenant := &popsignerv1.POPSignerTenant{}
	if err := r.Get(ctx, req.NamespacedName, tenant); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get parent cluster
	cluster := &popsignerv1.POPSignerCluster{}
	clusterKey := client.ObjectKey{
		Name:      tenant.Spec.ClusterRef.Name,
		Namespace: tenant.Namespace,
	}
	if err := r.Get(ctx, clusterKey, cluster); err != nil {
		log.Error(err, "Failed to get parent cluster")
		return r.updateTenantStatus(ctx, tenant, "Failed", "Parent cluster not found")
	}

	// Check cluster is ready
	if cluster.Status.Phase != "Running" {
		log.Info("Waiting for cluster to be ready", "clusterPhase", cluster.Status.Phase)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Get OpenBao client
	baoClient, err := r.getOpenBaoClient(ctx, cluster)
	if err != nil {
		log.Error(err, "Failed to get OpenBao client")
		return r.updateTenantStatus(ctx, tenant, "Failed", "Failed to connect to OpenBao")
	}

	// Reconcile tenant resources
	if err := r.reconcileTenantNamespace(ctx, tenant, cluster, baoClient); err != nil {
		log.Error(err, "Failed to reconcile namespace")
		return r.updateTenantStatus(ctx, tenant, "Failed", err.Error())
	}

	if err := r.reconcileTenantPolicies(ctx, tenant, cluster, baoClient); err != nil {
		log.Error(err, "Failed to reconcile policies")
		return r.updateTenantStatus(ctx, tenant, "Failed", err.Error())
	}

	if err := r.reconcileTenantQuotas(ctx, tenant, cluster); err != nil {
		log.Error(err, "Failed to reconcile quotas")
		// Non-fatal, continue
	}

	// Create admin user in Control Plane
	if err := r.reconcileTenantAdmin(ctx, tenant, cluster); err != nil {
		log.Error(err, "Failed to create admin user")
		// Non-fatal, continue
	}

	return r.updateTenantStatus(ctx, tenant, "Active", "")
}

// getOpenBaoClient creates an OpenBao client for the cluster
func (r *TenantReconciler) getOpenBaoClient(ctx context.Context, cluster *popsignerv1.POPSignerCluster) (*openbao.Client, error) {
	// Get root token from secret
	// The secret name follows the pattern: {cluster-name}-openbao-root-token
	secretName := fmt.Sprintf("%s-openbao-root-token", cluster.Name)
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: cluster.Namespace,
	}, secret); err != nil {
		return nil, fmt.Errorf("failed to get OpenBao root token secret: %w", err)
	}

	token, ok := secret.Data["token"]
	if !ok {
		return nil, fmt.Errorf("token key not found in secret %s", secretName)
	}

	// Build OpenBao address
	// Default to internal service address
	addr := fmt.Sprintf("http://%s-openbao.%s.svc.cluster.local:8200", cluster.Name, cluster.Namespace)
	if cluster.Status.Endpoints.OpenBao != "" {
		addr = cluster.Status.Endpoints.OpenBao
	}

	return openbao.NewClient(addr, string(token)), nil
}

// reconcileTenantNamespace creates an OpenBao namespace for tenant isolation
func (r *TenantReconciler) reconcileTenantNamespace(ctx context.Context, tenant *popsignerv1.POPSignerTenant, cluster *popsignerv1.POPSignerCluster, baoClient *openbao.Client) error {
	log := log.FromContext(ctx)

	namespaceName := fmt.Sprintf("tenant-%s", tenant.Name)
	tenant.Status.OpenBaoNamespace = namespaceName

	// Check if namespace already exists
	exists, err := baoClient.NamespaceExists(ctx, namespaceName)
	if err != nil {
		return fmt.Errorf("checking namespace existence: %w", err)
	}

	if exists {
		log.Info("OpenBao namespace already exists", "namespace", namespaceName)
		return nil
	}

	// Create namespace
	log.Info("Creating OpenBao namespace", "namespace", namespaceName)
	if err := baoClient.CreateNamespace(ctx, namespaceName); err != nil {
		return fmt.Errorf("creating OpenBao namespace: %w", err)
	}

	// Enable the secp256k1 secrets engine in the namespace
	nsClient := baoClient.WithNamespace(namespaceName)
	if err := nsClient.EnableSecretsEngine(ctx, "keys", "secp256k1"); err != nil {
		log.Error(err, "Failed to enable secp256k1 secrets engine", "namespace", namespaceName)
		// Non-fatal - might already be enabled or plugin not installed yet
	}

	return nil
}

// reconcileTenantPolicies creates OpenBao policies for tenant
func (r *TenantReconciler) reconcileTenantPolicies(ctx context.Context, tenant *popsignerv1.POPSignerTenant, cluster *popsignerv1.POPSignerCluster, baoClient *openbao.Client) error {
	log := log.FromContext(ctx)

	policyName := fmt.Sprintf("tenant-%s", tenant.Name)
	namespaceName := fmt.Sprintf("tenant-%s", tenant.Name)

	// Create policy for tenant access
	policy := fmt.Sprintf(`
# Tenant policy for %s
# Allows full access to keys within the tenant namespace

# Key management
path "%s/keys/*" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

# Signing operations
path "%s/keys/sign/*" {
  capabilities = ["create", "update"]
}

# Verification operations
path "%s/keys/verify/*" {
  capabilities = ["create", "update"]
}

# Export (if allowed by settings)
path "%s/keys/export/*" {
  capabilities = ["read"]
}
`, tenant.Name, namespaceName, namespaceName, namespaceName, namespaceName)

	log.Info("Creating OpenBao policy", "policy", policyName)
	if err := baoClient.CreatePolicy(ctx, policyName, policy); err != nil {
		return fmt.Errorf("creating OpenBao policy: %w", err)
	}

	return nil
}

// reconcileTenantQuotas applies quotas based on plan
func (r *TenantReconciler) reconcileTenantQuotas(ctx context.Context, tenant *popsignerv1.POPSignerTenant, cluster *popsignerv1.POPSignerCluster) error {
	log := log.FromContext(ctx)

	// Get quotas for the plan
	planQuotas := plans.GetPlanQuotas(tenant.Spec.Plan)
	log.Info("Applying plan quotas", "plan", tenant.Spec.Plan,
		"keys", planQuotas.Keys,
		"signaturesPerMonth", planQuotas.SignaturesPerMonth)

	// Apply custom quotas from spec if they are more restrictive
	quotas := tenant.Spec.Quotas
	if quotas.Keys > 0 && (planQuotas.Keys == -1 || quotas.Keys < planQuotas.Keys) {
		planQuotas.Keys = quotas.Keys
	}
	if quotas.SignaturesPerMonth > 0 && (planQuotas.SignaturesPerMonth == -1 || quotas.SignaturesPerMonth < planQuotas.SignaturesPerMonth) {
		planQuotas.SignaturesPerMonth = quotas.SignaturesPerMonth
	}
	if quotas.Namespaces > 0 && (planQuotas.Namespaces == -1 || quotas.Namespaces < planQuotas.Namespaces) {
		planQuotas.Namespaces = quotas.Namespaces
	}
	if quotas.TeamMembers > 0 && (planQuotas.TeamMembers == -1 || quotas.TeamMembers < planQuotas.TeamMembers) {
		planQuotas.TeamMembers = quotas.TeamMembers
	}
	if quotas.APIKeys > 0 && (planQuotas.APIKeys == -1 || quotas.APIKeys < planQuotas.APIKeys) {
		planQuotas.APIKeys = quotas.APIKeys
	}

	// TODO: Store quotas in Control Plane database
	// The API enforces these limits at runtime
	// This would be done via the Control Plane internal API:
	// POST /api/v1/internal/tenants/{id}/quotas

	return nil
}

// reconcileTenantAdmin creates the initial admin user
func (r *TenantReconciler) reconcileTenantAdmin(ctx context.Context, tenant *popsignerv1.POPSignerTenant, cluster *popsignerv1.POPSignerCluster) error {
	if tenant.Spec.Admin.Email == "" {
		return nil
	}

	log := log.FromContext(ctx)
	log.Info("Creating admin user", "email", tenant.Spec.Admin.Email)

	// TODO: Create admin user via Control Plane API
	// POST /api/v1/internal/tenants/{id}/admin
	// {
	//   "email": "admin@example.com",
	//   "password": "...",
	//   "role": "admin"
	// }

	return nil
}

// updateTenantStatus updates the tenant status
func (r *TenantReconciler) updateTenantStatus(ctx context.Context, tenant *popsignerv1.POPSignerTenant, phase, message string) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	tenant.Status.Phase = phase
	now := metav1.Now()

	if phase == "Active" {
		tenant.Status.LastActiveAt = &now
		if tenant.Status.CreatedAt == nil {
			tenant.Status.CreatedAt = &now
		}

		// Set ready condition
		setCondition(&tenant.Status.Conditions, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "TenantActive",
			Message:            "Tenant is active and ready",
			ObservedGeneration: tenant.Generation,
			LastTransitionTime: now,
		})
	} else if phase == "Failed" {
		// Set failed condition
		setCondition(&tenant.Status.Conditions, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "TenantFailed",
			Message:            message,
			ObservedGeneration: tenant.Generation,
			LastTransitionTime: now,
		})
	}

	if err := r.Status().Update(ctx, tenant); err != nil {
		log.Error(err, "Failed to update tenant status")
		return ctrl.Result{}, err
	}

	if phase == "Failed" {
		return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

// setCondition updates or appends a condition
func setCondition(conditions *[]metav1.Condition, condition metav1.Condition) {
	for i, c := range *conditions {
		if c.Type == condition.Type {
			(*conditions)[i] = condition
			return
		}
	}
	*conditions = append(*conditions, condition)
}

// SetupWithManager sets up the controller with the Manager.
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&popsignerv1.POPSignerTenant{}).
		Complete(r)
}
