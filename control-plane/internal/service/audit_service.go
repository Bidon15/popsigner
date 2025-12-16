// Package service provides business logic implementations.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"

	"github.com/Bidon15/popsigner/control-plane/internal/models"
	"github.com/Bidon15/popsigner/control-plane/internal/repository"
)

// AuditService defines the interface for audit log operations.
type AuditService interface {
	// Log creates a new audit log entry.
	Log(ctx context.Context, entry AuditEntry) error

	// Query retrieves audit logs with filtering and pagination.
	Query(ctx context.Context, orgID uuid.UUID, filter AuditFilter) ([]*models.AuditLog, string, error)

	// CountForPeriod counts audit logs for an organization within a time period.
	CountForPeriod(ctx context.Context, orgID uuid.UUID, start, end time.Time) (int64, error)

	// CleanupOldLogs removes audit logs older than the retention period for each org.
	CleanupOldLogs(ctx context.Context) (int64, error)

	// GetByID retrieves a single audit log by ID.
	GetByID(ctx context.Context, id uuid.UUID) (*models.AuditLog, error)
}

// AuditEntry represents the input for creating an audit log.
type AuditEntry struct {
	OrgID        uuid.UUID
	Event        models.AuditEvent
	ActorID      *uuid.UUID
	ActorType    models.ActorType
	ResourceType *models.ResourceType
	ResourceID   *uuid.UUID
	IPAddress    string
	UserAgent    string
	Metadata     map[string]any
}

// AuditFilter defines query parameters for listing audit logs.
type AuditFilter struct {
	Event        *models.AuditEvent
	ActorID      *uuid.UUID
	ResourceType *models.ResourceType
	ResourceID   *uuid.UUID
	StartTime    *time.Time
	EndTime      *time.Time
	Limit        int
	Cursor       string
}

type auditService struct {
	auditRepo repository.AuditRepository
	orgRepo   repository.OrgRepository
}

// NewAuditService creates a new audit service.
func NewAuditService(
	auditRepo repository.AuditRepository,
	orgRepo repository.OrgRepository,
) AuditService {
	return &auditService{
		auditRepo: auditRepo,
		orgRepo:   orgRepo,
	}
}

// Log creates a new audit log entry.
func (s *auditService) Log(ctx context.Context, entry AuditEntry) error {
	// Build the audit log
	log := &models.AuditLog{
		ID:           uuid.New(),
		OrgID:        entry.OrgID,
		Event:        entry.Event,
		ActorID:      entry.ActorID,
		ActorType:    entry.ActorType,
		ResourceType: entry.ResourceType,
		ResourceID:   entry.ResourceID,
	}

	// Parse IP address if provided
	if entry.IPAddress != "" {
		if ip := net.ParseIP(entry.IPAddress); ip != nil {
			log.IPAddress = &ip
		}
	}

	// Set user agent if provided
	if entry.UserAgent != "" {
		log.UserAgent = &entry.UserAgent
	}

	// Marshal metadata if provided
	if len(entry.Metadata) > 0 {
		metadata, err := json.Marshal(entry.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		log.Metadata = metadata
	}

	return s.auditRepo.Create(ctx, log)
}

// Query retrieves audit logs with filtering and pagination.
func (s *auditService) Query(ctx context.Context, orgID uuid.UUID, filter AuditFilter) ([]*models.AuditLog, string, error) {
	// Set default and max limits
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	// Build query from filter
	query := models.AuditLogQuery{
		OrgID:        orgID,
		Event:        filter.Event,
		ActorID:      filter.ActorID,
		ResourceType: filter.ResourceType,
		ResourceID:   filter.ResourceID,
		StartTime:    filter.StartTime,
		EndTime:      filter.EndTime,
		Limit:        limit + 1, // Fetch one extra to determine if there's a next page
		Cursor:       filter.Cursor,
	}

	logs, err := s.auditRepo.List(ctx, query)
	if err != nil {
		return nil, "", fmt.Errorf("failed to query audit logs: %w", err)
	}

	// Determine next cursor
	var nextCursor string
	if len(logs) > limit {
		// There's more data - set the cursor to the last item's ID
		nextCursor = logs[limit-1].ID.String()
		logs = logs[:limit] // Trim to requested limit
	}

	return logs, nextCursor, nil
}

// CountForPeriod counts audit logs for an organization within a time period.
func (s *auditService) CountForPeriod(ctx context.Context, orgID uuid.UUID, start, end time.Time) (int64, error) {
	return s.auditRepo.CountByOrgAndPeriod(ctx, orgID, start, end)
}

// CleanupOldLogs removes audit logs older than the retention period for each org.
func (s *auditService) CleanupOldLogs(ctx context.Context) (int64, error) {
	// Get all organizations
	orgs, err := s.orgRepo.ListUserOrgs(ctx, uuid.Nil)
	if err != nil {
		// If ListUserOrgs requires a valid userID, we need a different approach
		// For now, we'll return 0 and log the issue
		return 0, fmt.Errorf("cleanup requires ListAll method on org repo: %w", err)
	}

	var totalDeleted int64

	for _, org := range orgs {
		// Get plan limits to determine retention period
		limits := models.GetPlanLimits(org.Plan)
		cutoff := time.Now().AddDate(0, 0, -limits.AuditRetentionDays)

		deleted, err := s.auditRepo.DeleteBefore(ctx, org.ID, cutoff)
		if err != nil {
			// Log error but continue with other orgs
			continue
		}
		totalDeleted += deleted
	}

	return totalDeleted, nil
}

// GetByID retrieves a single audit log by ID.
func (s *auditService) GetByID(ctx context.Context, id uuid.UUID) (*models.AuditLog, error) {
	log, err := s.auditRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}
	return log, nil
}

// Compile-time check to ensure auditService implements AuditService.
var _ AuditService = (*auditService)(nil)

// Helper functions for creating common audit entries

// LogKeyCreated creates an audit log for key creation.
func LogKeyCreated(s AuditService, ctx context.Context, orgID, keyID uuid.UUID, actorID *uuid.UUID, actorType models.ActorType, keyName string, ip, userAgent string) error {
	rt := models.ResourceTypeKey
	return s.Log(ctx, AuditEntry{
		OrgID:        orgID,
		Event:        models.AuditEventKeyCreated,
		ActorID:      actorID,
		ActorType:    actorType,
		ResourceType: &rt,
		ResourceID:   &keyID,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Metadata: map[string]any{
			"key_name": keyName,
		},
	})
}

// LogKeyDeleted creates an audit log for key deletion.
func LogKeyDeleted(s AuditService, ctx context.Context, orgID, keyID uuid.UUID, actorID *uuid.UUID, actorType models.ActorType, keyName string, ip, userAgent string) error {
	rt := models.ResourceTypeKey
	return s.Log(ctx, AuditEntry{
		OrgID:        orgID,
		Event:        models.AuditEventKeyDeleted,
		ActorID:      actorID,
		ActorType:    actorType,
		ResourceType: &rt,
		ResourceID:   &keyID,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Metadata: map[string]any{
			"key_name": keyName,
		},
	})
}

// LogKeySigned creates an audit log for signing operations.
func LogKeySigned(s AuditService, ctx context.Context, orgID, keyID uuid.UUID, actorID *uuid.UUID, actorType models.ActorType, ip, userAgent string) error {
	rt := models.ResourceTypeKey
	return s.Log(ctx, AuditEntry{
		OrgID:        orgID,
		Event:        models.AuditEventKeySigned,
		ActorID:      actorID,
		ActorType:    actorType,
		ResourceType: &rt,
		ResourceID:   &keyID,
		IPAddress:    ip,
		UserAgent:    userAgent,
	})
}

// LogKeyExported creates an audit log for key export.
func LogKeyExported(s AuditService, ctx context.Context, orgID, keyID uuid.UUID, actorID *uuid.UUID, actorType models.ActorType, ip, userAgent string) error {
	rt := models.ResourceTypeKey
	return s.Log(ctx, AuditEntry{
		OrgID:        orgID,
		Event:        models.AuditEventKeyExported,
		ActorID:      actorID,
		ActorType:    actorType,
		ResourceType: &rt,
		ResourceID:   &keyID,
		IPAddress:    ip,
		UserAgent:    userAgent,
	})
}

// LogAuthLogin creates an audit log for user login.
func LogAuthLogin(s AuditService, ctx context.Context, orgID, userID uuid.UUID, ip, userAgent string, provider string) error {
	rt := models.ResourceTypeUser
	return s.Log(ctx, AuditEntry{
		OrgID:        orgID,
		Event:        models.AuditEventAuthLogin,
		ActorID:      &userID,
		ActorType:    models.ActorTypeUser,
		ResourceType: &rt,
		ResourceID:   &userID,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Metadata: map[string]any{
			"provider": provider,
		},
	})
}

// LogAPIKeyUsed creates an audit log for API key authentication.
func LogAPIKeyUsed(s AuditService, ctx context.Context, orgID, apiKeyID uuid.UUID, ip, userAgent string) error {
	rt := models.ResourceTypeAPIKey
	return s.Log(ctx, AuditEntry{
		OrgID:        orgID,
		Event:        models.AuditEventAuthAPIKeyUsed,
		ActorID:      &apiKeyID,
		ActorType:    models.ActorTypeAPIKey,
		ResourceType: &rt,
		ResourceID:   &apiKeyID,
		IPAddress:    ip,
		UserAgent:    userAgent,
	})
}

// LogMemberInvited creates an audit log for member invitation.
func LogMemberInvited(s AuditService, ctx context.Context, orgID uuid.UUID, inviterID uuid.UUID, email string, role models.Role, ip, userAgent string) error {
	return s.Log(ctx, AuditEntry{
		OrgID:     orgID,
		Event:     models.AuditEventMemberInvited,
		ActorID:   &inviterID,
		ActorType: models.ActorTypeUser,
		IPAddress: ip,
		UserAgent: userAgent,
		Metadata: map[string]any{
			"invited_email": email,
			"role":          string(role),
		},
	})
}

// LogMemberRemoved creates an audit log for member removal.
func LogMemberRemoved(s AuditService, ctx context.Context, orgID, actorID, removedUserID uuid.UUID, ip, userAgent string) error {
	rt := models.ResourceTypeUser
	return s.Log(ctx, AuditEntry{
		OrgID:        orgID,
		Event:        models.AuditEventMemberRemoved,
		ActorID:      &actorID,
		ActorType:    models.ActorTypeUser,
		ResourceType: &rt,
		ResourceID:   &removedUserID,
		IPAddress:    ip,
		UserAgent:    userAgent,
	})
}

// LogWebhookCreated creates an audit log for webhook creation.
func LogWebhookCreated(s AuditService, ctx context.Context, orgID, webhookID uuid.UUID, actorID *uuid.UUID, actorType models.ActorType, url string, ip, userAgent string) error {
	rt := models.ResourceTypeWebhook
	return s.Log(ctx, AuditEntry{
		OrgID:        orgID,
		Event:        models.AuditEventWebhookCreated,
		ActorID:      actorID,
		ActorType:    actorType,
		ResourceType: &rt,
		ResourceID:   &webhookID,
		IPAddress:    ip,
		UserAgent:    userAgent,
		Metadata: map[string]any{
			"url": url,
		},
	})
}

// LogWebhookDeleted creates an audit log for webhook deletion.
func LogWebhookDeleted(s AuditService, ctx context.Context, orgID, webhookID uuid.UUID, actorID *uuid.UUID, actorType models.ActorType, ip, userAgent string) error {
	rt := models.ResourceTypeWebhook
	return s.Log(ctx, AuditEntry{
		OrgID:        orgID,
		Event:        models.AuditEventWebhookDeleted,
		ActorID:      actorID,
		ActorType:    actorType,
		ResourceType: &rt,
		ResourceID:   &webhookID,
		IPAddress:    ip,
		UserAgent:    userAgent,
	})
}

