package controllers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	popsignerv1 "github.com/Bidon15/popsigner/operator/api/v1"
	"github.com/Bidon15/popsigner/operator/internal/conditions"
	"github.com/Bidon15/popsigner/operator/internal/constants"
	openbaoClient "github.com/Bidon15/popsigner/operator/internal/openbao"
	"github.com/Bidon15/popsigner/operator/internal/resources"
	"github.com/Bidon15/popsigner/operator/internal/resources/openbao"
	"github.com/Bidon15/popsigner/operator/internal/unseal"
)

// reconcileOpenBao handles all OpenBao resources.
func (r *ClusterReconciler) reconcileOpenBao(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	log := log.FromContext(ctx)
	log.Info("Reconciling OpenBao")

	name := resources.ResourceName(cluster.Name, constants.ComponentOpenBao)

	// 0. Create ServiceAccount
	sa := openbao.ServiceAccount(cluster)
	if err := r.createOrUpdate(ctx, cluster, sa); err != nil {
		return fmt.Errorf("failed to reconcile serviceaccount: %w", err)
	}

	// 0.5 Create self-signed TLS secret (if not exists)
	tlsSecret, err := r.ensureTLSSecret(ctx, cluster, name)
	if err != nil {
		return fmt.Errorf("failed to ensure TLS secret: %w", err)
	}
	if tlsSecret != nil {
		if err := r.createOrUpdate(ctx, cluster, tlsSecret); err != nil {
			return fmt.Errorf("failed to reconcile TLS secret: %w", err)
		}
	}

	// 1. Create/update ConfigMap
	configMap, err := openbao.ConfigMap(cluster)
	if err != nil {
		return fmt.Errorf("failed to build configmap: %w", err)
	}
	if err := r.createOrUpdate(ctx, cluster, configMap); err != nil {
		return fmt.Errorf("failed to reconcile configmap: %w", err)
	}

	// 2. Create/update headless Service
	headlessSvc := openbao.HeadlessService(cluster)
	if err := r.createOrUpdate(ctx, cluster, headlessSvc); err != nil {
		return fmt.Errorf("failed to reconcile headless service: %w", err)
	}

	// 3. Create/update active Service
	activeSvc := openbao.ActiveService(cluster)
	if err := r.createOrUpdate(ctx, cluster, activeSvc); err != nil {
		return fmt.Errorf("failed to reconcile active service: %w", err)
	}

	// 4. Create/update internal Service
	internalSvc := openbao.InternalService(cluster)
	if err := r.createOrUpdate(ctx, cluster, internalSvc); err != nil {
		return fmt.Errorf("failed to reconcile internal service: %w", err)
	}

	// 5. Create/update StatefulSet
	sts := openbao.StatefulSet(cluster)

	// TODO: Add init container for plugin download once release artifacts are published
	// For now, we skip the plugin download init container since the GitHub release doesn't exist yet
	// The plugin can be registered manually after deployment
	// sts.Spec.Template.Spec.InitContainers = append(
	// 	sts.Spec.Template.Spec.InitContainers,
	// 	openbao.InitContainer(cluster),
	// )

	// Add unseal provider configuration if enabled
	if cluster.Spec.OpenBao.AutoUnseal.Enabled {
		if err := r.configureAutoUnseal(ctx, cluster, sts); err != nil {
			return fmt.Errorf("failed to configure auto-unseal: %w", err)
		}
	}

	if err := r.createOrUpdate(ctx, cluster, sts); err != nil {
		return fmt.Errorf("failed to reconcile statefulset: %w", err)
	}

	// 6. Initialize OpenBao (after pods are ready)
	// This creates a Job that initializes OpenBao, registers plugins, and enables secrets engines
	if r.isOpenBaoReady(ctx, cluster) {
		if err := r.initializeOpenBao(ctx, cluster); err != nil {
			log.Error(err, "Failed to initialize OpenBao")
			// Non-fatal - will retry on next reconcile
		}
	}

	return nil
}

// configureAutoUnseal adds auto-unseal configuration to the StatefulSet.
func (r *ClusterReconciler) configureAutoUnseal(ctx context.Context, cluster *popsignerv1.POPSignerCluster, sts *appsv1.StatefulSet) error {
	provider, err := unseal.GetProviderForCluster(cluster)
	if err != nil {
		return err
	}
	if provider == nil {
		return nil
	}

	// Validate the provider configuration
	if err := provider.Validate(&cluster.Spec.OpenBao.AutoUnseal); err != nil {
		return fmt.Errorf("invalid auto-unseal configuration: %w", err)
	}

	// Get additional environment variables
	envVars, err := provider.GetEnvVars(ctx, &cluster.Spec.OpenBao.AutoUnseal, cluster.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get provider env vars: %w", err)
	}

	// Add env vars to the openbao container
	for i := range sts.Spec.Template.Spec.Containers {
		if sts.Spec.Template.Spec.Containers[i].Name == "openbao" {
			sts.Spec.Template.Spec.Containers[i].Env = append(
				sts.Spec.Template.Spec.Containers[i].Env,
				envVars...,
			)

			// Add volume mounts
			volumeMounts := provider.GetVolumeMounts(&cluster.Spec.OpenBao.AutoUnseal)
			sts.Spec.Template.Spec.Containers[i].VolumeMounts = append(
				sts.Spec.Template.Spec.Containers[i].VolumeMounts,
				volumeMounts...,
			)
			break
		}
	}

	// Add volumes
	volumes := provider.GetVolumes(&cluster.Spec.OpenBao.AutoUnseal)
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, volumes...)

	return nil
}

// isOpenBaoReady checks if OpenBao pods are ready.
func (r *ClusterReconciler) isOpenBaoReady(ctx context.Context, cluster *popsignerv1.POPSignerCluster) bool {
	name := resources.ResourceName(cluster.Name, constants.ComponentOpenBao)

	sts := &appsv1.StatefulSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Namespace}, sts); err != nil {
		return false
	}

	expectedReplicas := cluster.Spec.OpenBao.Replicas
	if expectedReplicas == 0 {
		expectedReplicas = int32(constants.DefaultOpenBaoReplicas)
	}

	return sts.Status.ReadyReplicas >= expectedReplicas
}

// updateOpenBaoStatus updates the cluster status with OpenBao info.
func (r *ClusterReconciler) updateOpenBaoStatus(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	name := resources.ResourceName(cluster.Name, constants.ComponentOpenBao)

	sts := &appsv1.StatefulSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Namespace}, sts); err != nil {
		if errors.IsNotFound(err) {
			cluster.Status.OpenBao = popsignerv1.ComponentStatus{
				Ready:   false,
				Message: "StatefulSet not found",
			}
			return nil
		}
		return err
	}

	expectedReplicas := *sts.Spec.Replicas
	ready := sts.Status.ReadyReplicas >= expectedReplicas

	cluster.Status.OpenBao = popsignerv1.ComponentStatus{
		Ready:    ready,
		Version:  cluster.Spec.OpenBao.Version,
		Replicas: fmt.Sprintf("%d/%d", sts.Status.ReadyReplicas, expectedReplicas),
	}

	condStatus := metav1.ConditionFalse
	reason := conditions.ReasonNotReady
	message := fmt.Sprintf("Waiting for OpenBao pods: %d/%d ready", sts.Status.ReadyReplicas, expectedReplicas)

	if ready {
		condStatus = metav1.ConditionTrue
		reason = conditions.ReasonReady
		message = "OpenBao cluster is ready"
	}

	conditions.SetCondition(&cluster.Status.Conditions, conditions.TypeOpenBaoReady, condStatus, reason, message)

	return nil
}

// initializeOpenBao performs first-time initialization.
// It creates and runs a Job that:
// 1. Initializes OpenBao (vault operator init)
// 2. Stores root token and unseal keys in a Secret
// 3. Unseals OpenBao (if not using auto-unseal)
// 4. Registers plugins
// 5. Enables required secrets engines (secret/kv-v2, pki, transit)
func (r *ClusterReconciler) initializeOpenBao(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	log := log.FromContext(ctx)
	log.Info("Initializing OpenBao cluster")

	name := resources.ResourceName(cluster.Name, constants.ComponentOpenBao)
	jobName := openbao.InitJobName(cluster.Name)

	// Check if init job already exists and completed
	existingJob := &batchv1.Job{}
	err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: cluster.Namespace}, existingJob)
	if err == nil {
		// Job exists, check its status
		if existingJob.Status.Succeeded > 0 {
			log.Info("OpenBao init job already completed successfully")
			return nil
		}
		if existingJob.Status.Failed > 0 && existingJob.Status.Failed >= *existingJob.Spec.BackoffLimit {
			log.Error(nil, "OpenBao init job failed permanently", "failures", existingJob.Status.Failed)
			return fmt.Errorf("OpenBao init job failed after %d attempts", existingJob.Status.Failed)
		}
		// Job still running or retrying
		log.Info("OpenBao init job still running", "active", existingJob.Status.Active, "failed", existingJob.Status.Failed)
		return nil
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check init job: %w", err)
	}

	// Check if root token secret already exists (manual init)
	rootSecretName := fmt.Sprintf("%s-root", name)
	rootSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: rootSecretName, Namespace: cluster.Namespace}, rootSecret)
	if err == nil {
		log.Info("Root token secret already exists, skipping init job",
			"secret", rootSecretName)
		// Configure secrets engines using the existing token
		return r.configureSecretsEngines(ctx, cluster)
	}
	if !errors.IsNotFound(err) {
		return fmt.Errorf("failed to check root secret: %w", err)
	}

	// Create the init job
	log.Info("Creating OpenBao init job", "job", jobName)
	initJob := openbao.InitJob(cluster)

	if err := r.createOrUpdate(ctx, cluster, initJob); err != nil {
		return fmt.Errorf("failed to create init job: %w", err)
	}

	log.Info("OpenBao init job created, waiting for completion")
	return nil
}

// configureSecretsEngines enables required secrets engines using the OpenBao API.
// This is called after OpenBao is initialized and we have the root token.
func (r *ClusterReconciler) configureSecretsEngines(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	log := log.FromContext(ctx)
	log.Info("Configuring OpenBao secrets engines")

	// Get OpenBao client
	baoClient, err := r.getOpenBaoClientForCluster(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get OpenBao client: %w", err)
	}

	// Enable required secrets engines
	engines := openbao.DefaultSecretsEngines()
	for _, engine := range engines {
		log.Info("Enabling secrets engine", "path", engine.Path, "type", engine.Type)
		
		var opts map[string]interface{}
		if engine.Options != nil {
			opts = make(map[string]interface{})
			for k, v := range engine.Options {
				opts[k] = v
			}
		}
		// For kv-v2, we need to specify version
		if engine.Type == "kv-v2" {
			if opts == nil {
				opts = make(map[string]interface{})
			}
			opts["version"] = "2"
			// The actual type for the API is just "kv"
			if err := baoClient.EnableSecretsEngineWithOptions(ctx, engine.Path, "kv", opts); err != nil {
				log.Error(err, "Failed to enable secrets engine (may already exist)", 
					"path", engine.Path, "type", engine.Type)
			}
		} else {
			if err := baoClient.EnableSecretsEngine(ctx, engine.Path, engine.Type); err != nil {
				log.Error(err, "Failed to enable secrets engine (may already exist)", 
					"path", engine.Path, "type", engine.Type)
			}
		}
	}

	log.Info("Secrets engines configured successfully")
	return nil
}

// getOpenBaoClientForCluster creates an OpenBao client for the cluster.
func (r *ClusterReconciler) getOpenBaoClientForCluster(ctx context.Context, cluster *popsignerv1.POPSignerCluster) (*openbaoClient.Client, error) {
	name := resources.ResourceName(cluster.Name, constants.ComponentOpenBao)

	// Get root token from secret
	rootSecretName := fmt.Sprintf("%s-root", name)
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      rootSecretName,
		Namespace: cluster.Namespace,
	}, secret); err != nil {
		return nil, fmt.Errorf("failed to get OpenBao root token secret: %w", err)
	}

	token, ok := secret.Data["token"]
	if !ok {
		return nil, fmt.Errorf("token key not found in secret %s", rootSecretName)
	}

	// Build OpenBao address
	addr := fmt.Sprintf("https://%s.%s.svc.cluster.local:8200", name, cluster.Namespace)

	return openbaoClient.NewClient(addr, string(token)), nil
}

// registerPlugin registers the secp256k1 plugin.
// This is now handled by the init job, but we keep this for manual registration if needed.
func (r *ClusterReconciler) registerPlugin(ctx context.Context, cluster *popsignerv1.POPSignerCluster) error {
	log := log.FromContext(ctx)
	log.Info("Registering secp256k1 plugin")

	// Plugin registration is handled by the init job
	// This method can be used for manual registration if needed
	return nil
}

// createOrUpdate creates or updates a Kubernetes resource.
func (r *ClusterReconciler) createOrUpdate(ctx context.Context, cluster *popsignerv1.POPSignerCluster, obj client.Object) error {
	// Set owner reference
	if err := ctrl.SetControllerReference(cluster, obj, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	// Try to create the resource
	if err := r.Create(ctx, obj); err != nil {
		if errors.IsAlreadyExists(err) {
			// Resource exists, update it
			existing := obj.DeepCopyObject().(client.Object)
			key := types.NamespacedName{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			}
			if err := r.Get(ctx, key, existing); err != nil {
				return fmt.Errorf("failed to get existing resource: %w", err)
			}

			// Copy resource version for update
			obj.SetResourceVersion(existing.GetResourceVersion())

			if err := r.Update(ctx, obj); err != nil {
				return fmt.Errorf("failed to update resource: %w", err)
			}
		} else {
			return fmt.Errorf("failed to create resource: %w", err)
		}
	}

	return nil
}

// ensureTLSSecret creates a self-signed TLS certificate for OpenBao if it doesn't exist.
func (r *ClusterReconciler) ensureTLSSecret(ctx context.Context, cluster *popsignerv1.POPSignerCluster, name string) (*corev1.Secret, error) {
	secretName := name + "-tls"

	// Check if secret already exists
	existing := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: cluster.Namespace}, existing)
	if err == nil {
		return nil, nil // Secret exists, no need to create
	}
	if !errors.IsNotFound(err) {
		return nil, err
	}

	// Generate self-signed certificate
	certPEM, keyPEM, caPEM, err := generateSelfSignedCert(name, cluster.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate: %w", err)
	}

	labels := resources.Labels(cluster.Name, constants.ComponentOpenBao, cluster.Spec.OpenBao.Version)

	return &corev1.Secret{
		ObjectMeta: resources.ObjectMeta(secretName, cluster.Namespace, labels),
		Type:       corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": certPEM,
			"tls.key": keyPEM,
			"ca.crt":  caPEM,
		},
	}, nil
}

// generateSelfSignedCert generates a self-signed CA and certificate for OpenBao.
func generateSelfSignedCert(name, namespace string) (certPEM, keyPEM, caPEM []byte, err error) {
	// Generate CA key
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, err
	}

	// Create CA certificate
	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"POPSigner"},
			CommonName:   "OpenBao CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, err
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return nil, nil, nil, err
	}

	// Generate server key
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, err
	}

	// Create server certificate
	serverTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"POPSigner"},
			CommonName:   name,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(1, 0, 0), // 1 year
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		DNSNames: []string{
			name,
			name + "." + namespace,
			name + "." + namespace + ".svc",
			name + "." + namespace + ".svc.cluster.local",
			"*." + name,
			"*." + name + "." + namespace,
			"*." + name + "." + namespace + ".svc",
			"*." + name + "." + namespace + ".svc.cluster.local",
			"localhost",
		},
		IPAddresses: nil,
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, &serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, err
	}

	// Encode to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCertDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)})
	caPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})

	return certPEM, keyPEM, caPEM, nil
}
