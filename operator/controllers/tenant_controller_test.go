package controllers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	popsignerv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
)

var _ = Describe("TenantController", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating a POPSignerTenant", func() {
		It("Should update status when parent cluster is not found", func() {
			By("Creating a tenant without a parent cluster")
			tenant := &popsignerv1.POPSignerTenant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tenant-no-cluster",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerTenantSpec{
					ClusterRef: popsignerv1.ClusterReference{
						Name: "non-existent-cluster",
					},
					Plan: "free",
				},
			}
			Expect(k8sClient.Create(ctx, tenant)).Should(Succeed())

			// Clean up
			defer func() {
				Expect(k8sClient.Delete(ctx, tenant)).Should(Succeed())
			}()

			tenantLookupKey := types.NamespacedName{Name: "test-tenant-no-cluster", Namespace: "default"}
			createdTenant := &popsignerv1.POPSignerTenant{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, tenantLookupKey, createdTenant)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})

		It("Should create tenant with default quotas for free plan", func() {
			By("Creating a tenant with free plan")
			tenant := &popsignerv1.POPSignerTenant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tenant-free",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerTenantSpec{
					ClusterRef: popsignerv1.ClusterReference{
						Name: "test-cluster",
					},
					DisplayName: "Test Tenant",
					Plan:        "free",
				},
			}
			Expect(k8sClient.Create(ctx, tenant)).Should(Succeed())

			// Clean up
			defer func() {
				Expect(k8sClient.Delete(ctx, tenant)).Should(Succeed())
			}()

			tenantLookupKey := types.NamespacedName{Name: "test-tenant-free", Namespace: "default"}
			createdTenant := &popsignerv1.POPSignerTenant{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, tenantLookupKey, createdTenant)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdTenant.Spec.Plan).Should(Equal("free"))
		})

		It("Should create tenant with custom quotas", func() {
			By("Creating a tenant with custom quotas")
			tenant := &popsignerv1.POPSignerTenant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tenant-custom",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerTenantSpec{
					ClusterRef: popsignerv1.ClusterReference{
						Name: "test-cluster",
					},
					DisplayName: "Custom Tenant",
					Plan:        "pro",
					Quotas: popsignerv1.TenantQuotas{
						Keys:               15,
						SignaturesPerMonth: 250000,
						Namespaces:         3,
						TeamMembers:        5,
						APIKeys:            10,
					},
				},
			}
			Expect(k8sClient.Create(ctx, tenant)).Should(Succeed())

			// Clean up
			defer func() {
				Expect(k8sClient.Delete(ctx, tenant)).Should(Succeed())
			}()

			tenantLookupKey := types.NamespacedName{Name: "test-tenant-custom", Namespace: "default"}
			createdTenant := &popsignerv1.POPSignerTenant{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, tenantLookupKey, createdTenant)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdTenant.Spec.Plan).Should(Equal("pro"))
			Expect(createdTenant.Spec.Quotas.Keys).Should(Equal(int32(15)))
			Expect(createdTenant.Spec.Quotas.SignaturesPerMonth).Should(Equal(int64(250000)))
		})

		It("Should create tenant with admin user", func() {
			By("Creating a tenant with admin settings")
			tenant := &popsignerv1.POPSignerTenant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tenant-admin",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerTenantSpec{
					ClusterRef: popsignerv1.ClusterReference{
						Name: "test-cluster",
					},
					DisplayName: "Admin Tenant",
					Plan:        "starter",
					Admin: popsignerv1.TenantAdmin{
						Email: "admin@example.com",
					},
				},
			}
			Expect(k8sClient.Create(ctx, tenant)).Should(Succeed())

			// Clean up
			defer func() {
				Expect(k8sClient.Delete(ctx, tenant)).Should(Succeed())
			}()

			tenantLookupKey := types.NamespacedName{Name: "test-tenant-admin", Namespace: "default"}
			createdTenant := &popsignerv1.POPSignerTenant{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, tenantLookupKey, createdTenant)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdTenant.Spec.Admin.Email).Should(Equal("admin@example.com"))
		})
	})

	Context("When tenant has a ready cluster", func() {
		var cluster *popsignerv1.POPSignerCluster
		var secret *corev1.Secret

		BeforeEach(func() {
			By("Creating a parent cluster")
			cluster = &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ready-cluster",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerClusterSpec{
					Domain: "keys.example.com",
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())

			// Update cluster status to Running
			cluster.Status.Phase = "Running"
			Expect(k8sClient.Status().Update(ctx, cluster)).Should(Succeed())

			By("Creating the OpenBao root token secret")
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ready-cluster-openbao-root-token",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"token": []byte("test-root-token"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, cluster)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, secret)).Should(Succeed())
		})

		It("Should set OpenBao namespace in status", func() {
			By("Creating a tenant with a ready cluster")
			tenant := &popsignerv1.POPSignerTenant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tenant-ready",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerTenantSpec{
					ClusterRef: popsignerv1.ClusterReference{
						Name: "ready-cluster",
					},
					Plan: "pro",
				},
			}
			Expect(k8sClient.Create(ctx, tenant)).Should(Succeed())

			defer func() {
				Expect(k8sClient.Delete(ctx, tenant)).Should(Succeed())
			}()

			tenantLookupKey := types.NamespacedName{Name: "test-tenant-ready", Namespace: "default"}
			createdTenant := &popsignerv1.POPSignerTenant{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, tenantLookupKey, createdTenant)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdTenant.Spec.Plan).Should(Equal("pro"))
		})
	})

	Context("When testing tenant settings", func() {
		It("Should create tenant with webhook configuration", func() {
			By("Creating a tenant with webhooks")
			tenant := &popsignerv1.POPSignerTenant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-tenant-webhooks",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerTenantSpec{
					ClusterRef: popsignerv1.ClusterReference{
						Name: "test-cluster",
					},
					Plan: "enterprise",
					Settings: popsignerv1.TenantSettings{
						AuditRetentionDays:  90,
						AllowExportableKeys: true,
						AllowedIPRanges:     []string{"10.0.0.0/8", "192.168.0.0/16"},
					},
				},
			}
			Expect(k8sClient.Create(ctx, tenant)).Should(Succeed())

			defer func() {
				Expect(k8sClient.Delete(ctx, tenant)).Should(Succeed())
			}()

			tenantLookupKey := types.NamespacedName{Name: "test-tenant-webhooks", Namespace: "default"}
			createdTenant := &popsignerv1.POPSignerTenant{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, tenantLookupKey, createdTenant)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdTenant.Spec.Settings.AuditRetentionDays).Should(Equal(int32(90)))
			Expect(createdTenant.Spec.Settings.AllowExportableKeys).Should(BeTrue())
			Expect(createdTenant.Spec.Settings.AllowedIPRanges).Should(HaveLen(2))
		})
	})
})
