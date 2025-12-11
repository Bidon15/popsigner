# Agent 13F: Backup/Restore Controller

## Overview

Implement Backup and Restore controllers for disaster recovery. Supports scheduled backups via CronJob and on-demand backups/restores via custom resources.

> **Requires:** Agent 13A (Operator Foundation) complete

---

## Deliverables

### 1. Backup Controller

```go
// controllers/backup_controller.go
package controllers

import (
    "context"
    "fmt"
    "time"

    batchv1 "k8s.io/api/batch/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/log"

    banhbaoringv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
)

type BackupReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

func (r *BackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)

    backup := &banhbaoringv1.BanhBaoRingBackup{}
    if err := r.Get(ctx, req.NamespacedName, backup); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Skip if already completed or failed
    if backup.Status.Phase == "Completed" || backup.Status.Phase == "Failed" {
        return ctrl.Result{}, nil
    }

    // Get parent cluster
    cluster := &banhbaoringv1.BanhBaoRingCluster{}
    if err := r.Get(ctx, client.ObjectKey{
        Name:      backup.Spec.ClusterRef.Name,
        Namespace: backup.Namespace,
    }, cluster); err != nil {
        return r.updateBackupStatus(ctx, backup, "Failed", err.Error())
    }

    // Create backup job
    if backup.Status.Phase == "" || backup.Status.Phase == "Pending" {
        job := r.buildBackupJob(backup, cluster)
        if err := r.Create(ctx, job); err != nil {
            return r.updateBackupStatus(ctx, backup, "Failed", err.Error())
        }
        now := metav1.Now()
        backup.Status.StartedAt = &now
        return r.updateBackupStatus(ctx, backup, "Running", "")
    }

    // Check job status
    job := &batchv1.Job{}
    jobName := fmt.Sprintf("%s-backup", backup.Name)
    if err := r.Get(ctx, client.ObjectKey{Name: jobName, Namespace: backup.Namespace}, job); err != nil {
        return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
    }

    if job.Status.Succeeded > 0 {
        now := metav1.Now()
        backup.Status.CompletedAt = &now
        return r.updateBackupStatus(ctx, backup, "Completed", "")
    }

    if job.Status.Failed > 0 {
        return r.updateBackupStatus(ctx, backup, "Failed", "Backup job failed")
    }

    return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *BackupReconciler) buildBackupJob(backup *banhbaoringv1.BanhBaoRingBackup, cluster *banhbaoringv1.BanhBaoRingCluster) *batchv1.Job {
    name := fmt.Sprintf("%s-backup", backup.Name)
    backoffLimit := int32(2)

    // Build backup script
    script := r.buildBackupScript(backup, cluster)

    return &batchv1.Job{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: backup.Namespace,
        },
        Spec: batchv1.JobSpec{
            BackoffLimit: &backoffLimit,
            Template: corev1.PodTemplateSpec{
                Spec: corev1.PodSpec{
                    RestartPolicy: corev1.RestartPolicyNever,
                    Containers: []corev1.Container{{
                        Name:    "backup",
                        Image:   "banhbaoring/backup:latest",
                        Command: []string{"/bin/sh", "-c", script},
                        Env:     r.buildBackupEnv(backup, cluster),
                    }},
                },
            },
        },
    }
}

func (r *BackupReconciler) buildBackupScript(backup *banhbaoringv1.BanhBaoRingBackup, cluster *banhbaoringv1.BanhBaoRingCluster) string {
    return `#!/bin/sh
set -e

TIMESTAMP=$(date +%Y%m%d-%H%M%S)

# Backup OpenBao (Raft snapshot)
if echo "$COMPONENTS" | grep -q "openbao"; then
    echo "Backing up OpenBao..."
    vault operator raft snapshot save /tmp/openbao-${TIMESTAMP}.snap
    aws s3 cp /tmp/openbao-${TIMESTAMP}.snap s3://${S3_BUCKET}/${S3_PREFIX}openbao-${TIMESTAMP}.snap
fi

# Backup PostgreSQL
if echo "$COMPONENTS" | grep -q "database"; then
    echo "Backing up PostgreSQL..."
    pg_dump $DATABASE_URL | gzip > /tmp/postgres-${TIMESTAMP}.sql.gz
    aws s3 cp /tmp/postgres-${TIMESTAMP}.sql.gz s3://${S3_BUCKET}/${S3_PREFIX}postgres-${TIMESTAMP}.sql.gz
fi

echo "Backup completed successfully"
`
}

func (r *BackupReconciler) updateBackupStatus(ctx context.Context, backup *banhbaoringv1.BanhBaoRingBackup, phase, message string) (ctrl.Result, error) {
    backup.Status.Phase = phase
    if err := r.Status().Update(ctx, backup); err != nil {
        return ctrl.Result{}, err
    }
    if phase == "Running" {
        return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
    }
    return ctrl.Result{}, nil
}
```

### 2. Scheduled Backup CronJob

```go
// internal/resources/backup/cronjob.go
package backup

import (
    "fmt"

    batchv1 "k8s.io/api/batch/v1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

    banhbaoringv1 "github.com/Bidon15/banhbaoring/operator/api/v1"
)

func CronJob(cluster *banhbaoringv1.BanhBaoRingCluster) *batchv1.CronJob {
    spec := cluster.Spec.Backup
    name := fmt.Sprintf("%s-backup", cluster.Name)

    schedule := spec.Schedule
    if schedule == "" {
        schedule = "0 2 * * *" // Daily at 2 AM
    }

    return &batchv1.CronJob{
        ObjectMeta: metav1.ObjectMeta{
            Name:      name,
            Namespace: cluster.Namespace,
        },
        Spec: batchv1.CronJobSpec{
            Schedule:          schedule,
            ConcurrencyPolicy: batchv1.ForbidConcurrent,
            JobTemplate: batchv1.JobTemplateSpec{
                Spec: batchv1.JobSpec{
                    Template: corev1.PodTemplateSpec{
                        Spec: corev1.PodSpec{
                            RestartPolicy: corev1.RestartPolicyNever,
                            Containers: []corev1.Container{{
                                Name:    "backup",
                                Image:   "banhbaoring/backup:latest",
                                Command: []string{"/backup.sh"},
                                Env:     buildEnv(cluster),
                            }},
                        },
                    },
                },
            },
        },
    }
}

func buildEnv(cluster *banhbaoringv1.BanhBaoRingCluster) []corev1.EnvVar {
    dest := cluster.Spec.Backup.Destination
    var env []corev1.EnvVar

    if dest.S3 != nil {
        env = append(env,
            corev1.EnvVar{Name: "S3_BUCKET", Value: dest.S3.Bucket},
            corev1.EnvVar{Name: "S3_PREFIX", Value: dest.S3.Prefix},
            corev1.EnvVar{Name: "AWS_REGION", Value: dest.S3.Region},
        )
    }

    return env
}
```

### 3. Restore Controller

```go
// controllers/restore_controller.go
package controllers

type RestoreReconciler struct {
    client.Client
    Scheme *runtime.Scheme
}

func (r *RestoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)

    restore := &banhbaoringv1.BanhBaoRingRestore{}
    if err := r.Get(ctx, req.NamespacedName, restore); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    if restore.Status.Phase == "Completed" || restore.Status.Phase == "Failed" {
        return ctrl.Result{}, nil
    }

    cluster := &banhbaoringv1.BanhBaoRingCluster{}
    if err := r.Get(ctx, client.ObjectKey{
        Name:      restore.Spec.ClusterRef.Name,
        Namespace: restore.Namespace,
    }, cluster); err != nil {
        return r.updateRestoreStatus(ctx, restore, "Failed")
    }

    switch restore.Status.Phase {
    case "", "Pending":
        if restore.Spec.Options.StopApplications {
            if err := r.scaleDownApps(ctx, cluster); err != nil {
                return r.updateRestoreStatus(ctx, restore, "Failed")
            }
        }
        return r.updateRestoreStatus(ctx, restore, "Stopping")

    case "Stopping":
        if r.appsScaledDown(ctx, cluster) {
            return r.updateRestoreStatus(ctx, restore, "Restoring")
        }
        return ctrl.Result{RequeueAfter: 10 * time.Second}, nil

    case "Restoring":
        if err := r.runRestore(ctx, restore, cluster); err != nil {
            return r.updateRestoreStatus(ctx, restore, "Failed")
        }
        return r.updateRestoreStatus(ctx, restore, "Starting")

    case "Starting":
        if err := r.scaleUpApps(ctx, cluster); err != nil {
            return r.updateRestoreStatus(ctx, restore, "Failed")
        }
        return r.updateRestoreStatus(ctx, restore, "Completed")
    }

    return ctrl.Result{}, nil
}

func (r *RestoreReconciler) scaleDownApps(ctx context.Context, cluster *banhbaoringv1.BanhBaoRingCluster) error {
    // Scale API and Dashboard to 0
    zero := int32(0)
    // Update deployments with replicas: 0
    return nil
}

func (r *RestoreReconciler) runRestore(ctx context.Context, restore *banhbaoringv1.BanhBaoRingRestore, cluster *banhbaoringv1.BanhBaoRingCluster) error {
    // Create restore job
    return nil
}
```

---

## Test Commands

```bash
cd operator
go build ./...
go test ./controllers/... -v -run TestBackup
go test ./controllers/... -v -run TestRestore
```

---

## Acceptance Criteria

- [ ] Backup controller with Job creation
- [ ] Scheduled backup via CronJob
- [ ] S3/GCS destination support
- [ ] Restore controller with phased workflow
- [ ] Application scale down/up during restore
- [ ] Status tracking for both resources

