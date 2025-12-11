// Package conditions provides helpers for managing Kubernetes conditions.
package conditions

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition types
const (
	TypeReady          = "Ready"
	TypeTLSReady       = "TLSReady"
	TypeDatabaseReady  = "DatabaseReady"
	TypeRedisReady     = "RedisReady"
	TypeOpenBaoReady   = "OpenBaoReady"
	TypeAPIReady       = "APIReady"
	TypeDashboardReady = "DashboardReady"
	TypeBackupSuccess  = "BackupSucceeded"
	TypeRestoreSuccess = "RestoreSucceeded"
	TypeTenantReady    = "TenantReady"
)

// Condition reasons
const (
	ReasonInitializing    = "Initializing"
	ReasonProgressing     = "Progressing"
	ReasonAvailable       = "Available"
	ReasonDegraded        = "Degraded"
	ReasonFailed          = "Failed"
	ReasonNotReady        = "NotReady"
	ReasonReady           = "Ready"
	ReasonBackupRunning   = "BackupRunning"
	ReasonBackupComplete  = "BackupComplete"
	ReasonBackupFailed    = "BackupFailed"
	ReasonRestoreRunning  = "RestoreRunning"
	ReasonRestoreComplete = "RestoreComplete"
	ReasonRestoreFailed   = "RestoreFailed"
)

// SetCondition adds or updates a condition in the conditions slice.
func SetCondition(conditions *[]metav1.Condition, condType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.Now()

	for i, c := range *conditions {
		if c.Type == condType {
			if c.Status != status {
				(*conditions)[i].Status = status
				(*conditions)[i].LastTransitionTime = now
			}
			(*conditions)[i].Reason = reason
			(*conditions)[i].Message = message
			(*conditions)[i].ObservedGeneration = c.ObservedGeneration
			return
		}
	}

	*conditions = append(*conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	})
}

// SetConditionWithGeneration adds or updates a condition with observed generation.
func SetConditionWithGeneration(conditions *[]metav1.Condition, condType string, status metav1.ConditionStatus, reason, message string, generation int64) {
	now := metav1.Now()

	for i, c := range *conditions {
		if c.Type == condType {
			if c.Status != status {
				(*conditions)[i].Status = status
				(*conditions)[i].LastTransitionTime = now
			}
			(*conditions)[i].Reason = reason
			(*conditions)[i].Message = message
			(*conditions)[i].ObservedGeneration = generation
			return
		}
	}

	*conditions = append(*conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: generation,
	})
}

// GetCondition returns the condition with the given type, or nil if not found.
func GetCondition(conditions []metav1.Condition, condType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == condType {
			return &conditions[i]
		}
	}
	return nil
}

// IsConditionTrue returns true if the condition with the given type is True.
func IsConditionTrue(conditions []metav1.Condition, condType string) bool {
	cond := GetCondition(conditions, condType)
	return cond != nil && cond.Status == metav1.ConditionTrue
}

// IsConditionFalse returns true if the condition with the given type is False.
func IsConditionFalse(conditions []metav1.Condition, condType string) bool {
	cond := GetCondition(conditions, condType)
	return cond != nil && cond.Status == metav1.ConditionFalse
}

// RemoveCondition removes the condition with the given type from the conditions slice.
func RemoveCondition(conditions *[]metav1.Condition, condType string) {
	for i, c := range *conditions {
		if c.Type == condType {
			*conditions = append((*conditions)[:i], (*conditions)[i+1:]...)
			return
		}
	}
}
