// Package web provides HTTP handlers for the web dashboard.
package web

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Bidon15/popsigner/control-plane/internal/models"
	"github.com/Bidon15/popsigner/control-plane/internal/service"
	"github.com/Bidon15/popsigner/control-plane/templates/layouts"
	"github.com/Bidon15/popsigner/control-plane/templates/pages"
)

// Ensure imports are used
var (
	_ layouts.DashboardData
	_ = service.AuditFilter{}
)

// ============================================
// Audit Log Handlers Implementation
// ============================================

// AuditLog renders the audit log page.
func (h *WebHandler) AuditLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	session, _ := h.sessionStore.Get(r, SessionCookieName)
	userID, _ := session.Values["user_id"].(string)
	orgID, _ := session.Values["org_id"].(string)

	uid, _ := uuid.Parse(userID)
	oid, _ := uuid.Parse(orgID)

	user, _ := h.authService.GetUserByID(ctx, uid)
	org, _ := h.orgService.Get(ctx, oid)

	// Parse filters from query params
	event := r.URL.Query().Get("event")
	period := r.URL.Query().Get("period")
	actor := r.URL.Query().Get("actor")
	cursor := r.URL.Query().Get("cursor")

	// Calculate time range based on period
	var startTime *time.Time
	now := time.Now()
	switch period {
	case "30d":
		t := now.AddDate(0, 0, -30)
		startTime = &t
	case "90d":
		t := now.AddDate(0, 0, -90)
		startTime = &t
	default: // 7d
		t := now.AddDate(0, 0, -7)
		startTime = &t
		period = "7d"
	}

	// Build query
	query := models.AuditLogQuery{
		OrgID:     oid,
		StartTime: startTime,
		Limit:     50,
		Cursor:    cursor,
	}

	if event != "" {
		e := models.AuditEvent(event)
		query.Event = &e
	}

	if actor != "" {
		// Would filter by actor type
	}

	// Get audit logs
	filter := service.AuditFilter{
		Event:     query.Event,
		StartTime: query.StartTime,
		Limit:     query.Limit,
		Cursor:    query.Cursor,
	}
	logs, nextCursor, _ := h.auditService.Query(ctx, oid, filter)

	filters := pages.AuditFilters{
		Event:  event,
		Period: period,
		Actor:  actor,
	}

	// If this is an HTMX request for pagination or filtering, return just the list
	if r.Header.Get("HX-Request") == "true" && cursor != "" {
		component := pages.AuditList(logs, nextCursor)
		templ.Handler(component).ServeHTTP(w, r)
		return
	}

	dashboardData := buildDashboardData(user, org, "/audit")

	data := pages.AuditPageData{
		DashboardData: dashboardData,
		Logs:          logs,
		NextCursor:    nextCursor,
		Filters:       filters,
	}

	// For HTMX filter changes, return just the list
	if r.Header.Get("HX-Request") == "true" {
		component := pages.AuditList(logs, nextCursor)
		templ.Handler(component).ServeHTTP(w, r)
		return
	}

	component := pages.AuditPage(data)
	templ.Handler(component).ServeHTTP(w, r)
}

// AuditLogDetail renders the audit log detail modal.
func (h *WebHandler) AuditLogDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logID := chi.URLParam(r, "id")
	lid, err := uuid.Parse(logID)
	if err != nil {
		http.Error(w, "Invalid log ID", http.StatusBadRequest)
		return
	}

	session, _ := h.sessionStore.Get(r, SessionCookieName)
	orgID, _ := session.Values["org_id"].(string)
	oid, _ := uuid.Parse(orgID)
	_ = oid // TODO: Verify log belongs to this org

	// Get the specific audit log entry
	log, err := h.auditService.GetByID(ctx, lid)
	if err != nil {
		http.Error(w, "Audit log not found", http.StatusNotFound)
		return
	}

	component := pages.AuditDetailModal(log)
	templ.Handler(component).ServeHTTP(w, r)
}

// AuditLogExport exports audit logs as CSV.
func (h *WebHandler) AuditLogExport(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	session, _ := h.sessionStore.Get(r, SessionCookieName)
	orgID, _ := session.Values["org_id"].(string)
	oid, _ := uuid.Parse(orgID)

	// Parse filters
	event := r.URL.Query().Get("event")
	period := r.URL.Query().Get("period")

	// Calculate time range
	var startTime *time.Time
	now := time.Now()
	switch period {
	case "30d":
		t := now.AddDate(0, 0, -30)
		startTime = &t
	case "90d":
		t := now.AddDate(0, 0, -90)
		startTime = &t
	default:
		t := now.AddDate(0, 0, -7)
		startTime = &t
	}

	query := models.AuditLogQuery{
		OrgID:     oid,
		StartTime: startTime,
		Limit:     10000, // Export up to 10k records
	}

	if event != "" {
		e := models.AuditEvent(event)
		query.Event = &e
	}

	exportFilter := service.AuditFilter{
		Event:     query.Event,
		StartTime: query.StartTime,
		Limit:     query.Limit,
	}
	logs, _, _ := h.auditService.Query(ctx, oid, exportFilter)

	// Set headers for CSV download
	filename := fmt.Sprintf("audit-logs-%s.csv", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Write CSV
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"Time", "Event", "Actor Type", "Resource Type", "Resource ID", "IP Address", "User Agent"})

	// Write rows
	for _, log := range logs {
		row := []string{
			log.CreatedAt.Format(time.RFC3339),
			string(log.Event),
			string(log.ActorType),
			resourceTypeString(log.ResourceType),
			uuidString(log.ResourceID),
			ipString(log.IPAddress),
			stringPtr(log.UserAgent),
		}
		writer.Write(row)
	}
}

// ============================================
// Usage Page Handler
// ============================================

// Usage renders the usage analytics page.
func (h *WebHandler) Usage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	session, _ := h.sessionStore.Get(r, SessionCookieName)
	userID, _ := session.Values["user_id"].(string)
	orgID, _ := session.Values["org_id"].(string)

	uid, _ := uuid.Parse(userID)
	oid, _ := uuid.Parse(orgID)

	user, _ := h.authService.GetUserByID(ctx, uid)
	org, _ := h.orgService.Get(ctx, oid)

	// Get plan limits
	limits := models.GetPlanLimits(org.Plan)

	// Calculate current billing period
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0).Add(-time.Second)

	// Generate placeholder data for usage charts
	displaySigData := generatePlaceholderData(periodStart, now)
	displayAPIData := generatePlaceholderData(periodStart, now)

	dashboardData := buildDashboardData(user, org, "/usage")

	// Get key count from key service
	keys, _ := h.keyService.List(ctx, oid, nil)
	keyCount := int64(len(keys))

	data := pages.UsagePageData{
		DashboardData:    dashboardData,
		Signatures:       0, // Usage tracking coming soon
		SignaturesLimit:  limits.SignaturesPerMonth,
		Keys:             keyCount,
		KeysLimit:        int64(limits.Keys),
		APICalls:         0, // Usage tracking coming soon
		TeamMembers:      1, // Default to 1 for the owner
		TeamMembersLimit: int64(limits.TeamMembers),
		SignaturesData:   displaySigData,
		APICallsData:     displayAPIData,
		PeriodStart:      periodStart,
		PeriodEnd:        periodEnd,
	}

	component := pages.UsagePage(data)
	templ.Handler(component).ServeHTTP(w, r)
}

// ============================================
// Helper Functions
// ============================================

func resourceTypeString(rt *models.ResourceType) string {
	if rt == nil {
		return ""
	}
	return string(*rt)
}

func uuidString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func ipString(ip interface{}) string {
	if ip == nil {
		return ""
	}
	return fmt.Sprintf("%v", ip)
}

func stringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func generatePlaceholderData(start, end time.Time) []pages.UsageDataPoint {
	var data []pages.UsageDataPoint
	current := start
	for current.Before(end) {
		data = append(data, pages.UsageDataPoint{
			Date:  current,
			Value: 0,
		})
		current = current.AddDate(0, 0, 1)
	}
	return data
}


