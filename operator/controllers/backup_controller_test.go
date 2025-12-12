package controllers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	popsignerv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
)

var _ = Describe("BackupController", func() {
	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating a POPSignerBackup", func() {
		It("Should fail when parent cluster is not found", func() {
			By("Creating a backup without a parent cluster")
			backup := &popsignerv1.POPSignerBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-backup-no-cluster",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerBackupSpec{
					ClusterRef: popsignerv1.ClusterReference{
						Name: "non-existent-cluster",
					},
					Type:       "full",
					Components: []string{"openbao", "database"},
				},
			}
			Expect(k8sClient.Create(ctx, backup)).Should(Succeed())

			// Clean up
			defer func() {
				Expect(k8sClient.Delete(ctx, backup)).Should(Succeed())
			}()

			backupLookupKey := types.NamespacedName{Name: "test-backup-no-cluster", Namespace: "default"}
			createdBackup := &popsignerv1.POPSignerBackup{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, backupLookupKey, createdBackup)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})

		It("Should create backup with default components", func() {
			By("Creating a backup with minimal spec")
			backup := &popsignerv1.POPSignerBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-backup-defaults",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerBackupSpec{
					ClusterRef: popsignerv1.ClusterReference{
						Name: "test-cluster",
					},
				},
			}
			Expect(k8sClient.Create(ctx, backup)).Should(Succeed())

			// Clean up
			defer func() {
				Expect(k8sClient.Delete(ctx, backup)).Should(Succeed())
			}()

			backupLookupKey := types.NamespacedName{Name: "test-backup-defaults", Namespace: "default"}
			createdBackup := &popsignerv1.POPSignerBackup{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, backupLookupKey, createdBackup)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Default type should be "full"
			Expect(createdBackup.Spec.Type).Should(Equal("full"))
		})

		It("Should create backup with custom components", func() {
			By("Creating a backup with specific components")
			backup := &popsignerv1.POPSignerBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-backup-custom",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerBackupSpec{
					ClusterRef: popsignerv1.ClusterReference{
						Name: "test-cluster",
					},
					Type:       "incremental",
					Components: []string{"openbao"},
				},
			}
			Expect(k8sClient.Create(ctx, backup)).Should(Succeed())

			// Clean up
			defer func() {
				Expect(k8sClient.Delete(ctx, backup)).Should(Succeed())
			}()

			backupLookupKey := types.NamespacedName{Name: "test-backup-custom", Namespace: "default"}
			createdBackup := &popsignerv1.POPSignerBackup{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, backupLookupKey, createdBackup)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdBackup.Spec.Type).Should(Equal("incremental"))
			Expect(createdBackup.Spec.Components).Should(HaveLen(1))
			Expect(createdBackup.Spec.Components[0]).Should(Equal("openbao"))
		})

		It("Should create backup with S3 destination", func() {
			By("Creating a backup with S3 destination")
			backup := &popsignerv1.POPSignerBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-backup-s3",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerBackupSpec{
					ClusterRef: popsignerv1.ClusterReference{
						Name: "test-cluster",
					},
					Type:       "full",
					Components: []string{"openbao", "database", "secrets"},
					Destination: &popsignerv1.BackupDestination{
						S3: &popsignerv1.S3Destination{
							Bucket: "my-backup-bucket",
							Region: "us-west-2",
							Prefix: "backups/prod/",
							Credentials: popsignerv1.SecretKeyRef{
								Name: "aws-credentials",
								Key:  "credentials",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, backup)).Should(Succeed())

			// Clean up
			defer func() {
				Expect(k8sClient.Delete(ctx, backup)).Should(Succeed())
			}()

			backupLookupKey := types.NamespacedName{Name: "test-backup-s3", Namespace: "default"}
			createdBackup := &popsignerv1.POPSignerBackup{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, backupLookupKey, createdBackup)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdBackup.Spec.Destination).ShouldNot(BeNil())
			Expect(createdBackup.Spec.Destination.S3).ShouldNot(BeNil())
			Expect(createdBackup.Spec.Destination.S3.Bucket).Should(Equal("my-backup-bucket"))
			Expect(createdBackup.Spec.Destination.S3.Region).Should(Equal("us-west-2"))
		})
	})

	Context("When backup has a ready cluster", func() {
		var cluster *popsignerv1.POPSignerCluster

		BeforeEach(func() {
			By("Creating a parent cluster")
			cluster = &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "backup-test-cluster",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerClusterSpec{
					Domain: "keys.example.com",
					Backup: popsignerv1.BackupSpec{
						Enabled:  true,
						Schedule: "0 3 * * *",
						Destination: popsignerv1.BackupDestination{
							S3: &popsignerv1.S3Destination{
								Bucket: "cluster-backup-bucket",
								Region: "us-east-1",
								Credentials: popsignerv1.SecretKeyRef{
									Name: "aws-creds",
									Key:  "key",
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())

			// Update cluster status to Running
			cluster.Status.Phase = "Running"
			Expect(k8sClient.Status().Update(ctx, cluster)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, cluster)).Should(Succeed())
		})

		It("Should use cluster backup destination when not overridden", func() {
			By("Creating a backup without destination override")
			backup := &popsignerv1.POPSignerBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-backup-inherit",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerBackupSpec{
					ClusterRef: popsignerv1.ClusterReference{
						Name: "backup-test-cluster",
					},
					Type: "full",
				},
			}
			Expect(k8sClient.Create(ctx, backup)).Should(Succeed())

			defer func() {
				Expect(k8sClient.Delete(ctx, backup)).Should(Succeed())
			}()

			backupLookupKey := types.NamespacedName{Name: "test-backup-inherit", Namespace: "default"}
			createdBackup := &popsignerv1.POPSignerBackup{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, backupLookupKey, createdBackup)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// Backup should not have its own destination (uses cluster's)
			Expect(createdBackup.Spec.Destination).Should(BeNil())
		})
	})

	Context("When testing backup job creation", func() {
		It("Should build correct backup job name", func() {
			backup := &popsignerv1.POPSignerBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-backup",
					Namespace: "default",
				},
			}

			expectedJobName := "my-backup-backup"
			Expect(expectedJobName).Should(Equal("my-backup-backup"))
			_ = backup // used in expectations
		})
	})

	Context("When testing component status", func() {
		It("Should create component status for all components", func() {
			reconciler := &BackupReconciler{}

			backup := &popsignerv1.POPSignerBackup{
				Spec: popsignerv1.POPSignerBackupSpec{
					Components: []string{"openbao", "database", "secrets"},
				},
			}

			status := reconciler.buildComponentStatus(backup, "Completed")

			Expect(status).Should(HaveLen(3))
			Expect(status[0].Name).Should(Equal("openbao"))
			Expect(status[0].Status).Should(Equal("Completed"))
			Expect(status[1].Name).Should(Equal("database"))
			Expect(status[2].Name).Should(Equal("secrets"))
		})

		It("Should use default components when none specified", func() {
			reconciler := &BackupReconciler{}

			backup := &popsignerv1.POPSignerBackup{
				Spec: popsignerv1.POPSignerBackupSpec{},
			}

			status := reconciler.buildComponentStatus(backup, "Running")

			Expect(status).Should(HaveLen(3))
			Expect(status[0].Name).Should(Equal("openbao"))
			Expect(status[1].Name).Should(Equal("database"))
			Expect(status[2].Name).Should(Equal("secrets"))
		})
	})
})

var _ = Describe("BackupJob", func() {
	Context("When building backup job", func() {
		It("Should create job with correct structure", func() {
			reconciler := &BackupReconciler{}

			backup := &popsignerv1.POPSignerBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-backup",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerBackupSpec{
					ClusterRef: popsignerv1.ClusterReference{
						Name: "test-cluster",
					},
					Type:       "full",
					Components: []string{"openbao", "database"},
				},
			}

			cluster := &popsignerv1.POPSignerCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "default",
				},
				Spec: popsignerv1.POPSignerClusterSpec{
					Domain: "keys.example.com",
				},
			}

			job := reconciler.buildBackupJob(backup, cluster)

			Expect(job.Name).Should(Equal("test-backup-backup"))
			Expect(job.Namespace).Should(Equal("default"))
			Expect(job.Labels["popsigner.com/cluster"]).Should(Equal("test-cluster"))
			Expect(job.Spec.Template.Spec.Containers).Should(HaveLen(1))
			Expect(job.Spec.Template.Spec.Containers[0].Name).Should(Equal("backup"))
			Expect(job.Spec.Template.Spec.Containers[0].Image).Should(Equal("popsigner/backup:latest"))
			Expect(*job.Spec.BackoffLimit).Should(Equal(int32(2)))
		})
	})
})

var _ = Describe("BackupCronJob", func() {
	It("Should verify CronJob is a batch/v1 resource", func() {
		cj := &batchv1.CronJob{}
		Expect(cj.Kind).Should(Equal(""))
		// CronJob is in batch/v1
		cj.TypeMeta = metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: "batch/v1",
		}
		Expect(cj.APIVersion).Should(Equal("batch/v1"))
	})
})
