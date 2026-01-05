package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// AuditHandler handles audit log-related HTTP requests
type AuditHandler struct {
	auditService service.AuditServiceInterface
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(auditService service.AuditServiceInterface) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

// List returns a paginated list of audit logs with filters
func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	params, err := parseAuditListParams(r)
	if err != nil {
		utils.BadRequest(w, err.Error(), nil)
		return
	}

	result, err := h.auditService.ListAuditLogs(params)
	if err != nil {
		utils.InternalError(w, "Failed to retrieve audit logs")
		return
	}

	utils.Success(w, result, "Audit logs retrieved successfully")
}

// GetByID returns a single audit log by ID
func (h *AuditHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(w, "Invalid audit log ID", nil)
		return
	}

	log, err := h.auditService.GetAuditLogByID(id)
	if err != nil {
		utils.NotFound(w, "Audit log not found")
		return
	}

	utils.Success(w, log, "Audit log retrieved successfully")
}

// GetStats returns aggregate statistics for audit logs
func (h *AuditHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.auditService.GetStats()
	if err != nil {
		utils.InternalError(w, "Failed to retrieve audit log statistics")
		return
	}

	utils.Success(w, stats, "Audit log statistics retrieved successfully")
}

// Export exports audit logs as CSV
func (h *AuditHandler) Export(w http.ResponseWriter, r *http.Request) {
	params, err := parseAuditListParams(r)
	if err != nil {
		utils.BadRequest(w, err.Error(), nil)
		return
	}

	// Override limit for export (get all matching records)
	params.Limit = 10000 // Max export limit
	params.Page = 1

	result, err := h.auditService.ListAuditLogs(params)
	if err != nil {
		utils.InternalError(w, "Failed to retrieve audit logs for export")
		return
	}

	// Set CSV headers
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=audit_logs_%s.csv", time.Now().Format("2006-01-02")))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header row
	header := []string{"ID", "Timestamp", "Action", "Resource Type", "Resource ID", "Resource Name", "User ID", "Status", "IP Address", "User Agent", "Error Message"}
	if err := writer.Write(header); err != nil {
		return
	}

	// Write data rows
	for i := range result.Items {
		log := &result.Items[i]
		row := []string{
			strconv.Itoa(log.ID),
			log.CreatedAt.Format(time.RFC3339),
			log.Action,
			stringOrEmpty(log.ResourceType),
			intPtrToString(log.ResourceID),
			stringOrEmpty(log.ResourceName),
			intPtrToString(log.UserID),
			log.Status,
			stringOrEmpty(log.IPAddress),
			stringOrEmpty(log.UserAgent),
			stringOrEmpty(log.ErrorMessage),
		}
		if err := writer.Write(row); err != nil {
			return
		}
	}
}

// GetConfig returns the audit configuration
func (h *AuditHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	config, err := h.auditService.GetConfig()
	if err != nil {
		utils.InternalError(w, "Failed to retrieve audit configuration")
		return
	}

	utils.Success(w, config, "Audit configuration retrieved successfully")
}

// UpdateConfigRequest represents the request body for updating audit config
type UpdateConfigRequest struct {
	ProxyEvents    bool `json:"proxy_events"`
	AuthEvents     bool `json:"auth_events"`
	SettingsEvents bool `json:"settings_events"`
	SyncEvents     bool `json:"sync_events"`
	SystemEvents   bool `json:"system_events"`
}

// UpdateConfig updates the audit configuration
func (h *AuditHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req UpdateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}

	config := &models.AuditConfig{
		ProxyEvents:    req.ProxyEvents,
		AuthEvents:     req.AuthEvents,
		SettingsEvents: req.SettingsEvents,
		SyncEvents:     req.SyncEvents,
		SystemEvents:   req.SystemEvents,
	}

	if err := h.auditService.SetConfig(config); err != nil {
		utils.InternalError(w, "Failed to update audit configuration")
		return
	}

	utils.Success(w, config, "Audit configuration updated successfully")
}

// parseAuditListParams parses query parameters for audit log listing
func parseAuditListParams(r *http.Request) (repository.AuditLogListParams, error) {
	params := repository.AuditLogListParams{
		Page:  1,
		Limit: 20,
	}

	// Parse page
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return params, fmt.Errorf("invalid page number")
		}
		params.Page = page
	}

	// Parse limit
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 100 {
			return params, fmt.Errorf("limit must be between 1 and 100")
		}
		params.Limit = limit
	}

	// Parse search
	params.Search = r.URL.Query().Get("search")

	// Parse action filter
	params.Action = r.URL.Query().Get("action")

	// Parse resource_type filter
	params.ResourceType = r.URL.Query().Get("resource_type")

	// Parse user_id filter
	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		userID, err := strconv.Atoi(userIDStr)
		if err != nil {
			return params, fmt.Errorf("invalid user_id")
		}
		params.UserID = &userID
	}

	// Parse status filter
	if status := r.URL.Query().Get("status"); status != "" {
		if status != "success" && status != "failure" {
			return params, fmt.Errorf("status must be 'success' or 'failure'")
		}
		params.Status = status
	}

	// Parse date_from filter
	if dateFromStr := r.URL.Query().Get("date_from"); dateFromStr != "" {
		dateFrom, err := time.Parse(time.RFC3339, dateFromStr)
		if err != nil {
			// Try simpler date format
			dateFrom, err = time.Parse("2006-01-02", dateFromStr)
			if err != nil {
				return params, fmt.Errorf("invalid date_from format (use ISO 8601 or YYYY-MM-DD)")
			}
		}
		params.DateFrom = &dateFrom
	}

	// Parse date_to filter
	if dateToStr := r.URL.Query().Get("date_to"); dateToStr != "" {
		dateTo, err := time.Parse(time.RFC3339, dateToStr)
		if err != nil {
			// Try simpler date format
			dateTo, err = time.Parse("2006-01-02", dateToStr)
			if err != nil {
				return params, fmt.Errorf("invalid date_to format (use ISO 8601 or YYYY-MM-DD)")
			}
			// Set to end of day
			dateTo = dateTo.Add(24*time.Hour - time.Second)
		}
		params.DateTo = &dateTo
	}

	// Parse sort
	params.Sort = r.URL.Query().Get("sort")

	// Parse order
	params.Order = r.URL.Query().Get("order")

	return params, nil
}

// Helper functions for CSV export
func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func intPtrToString(i *int) string {
	if i == nil {
		return ""
	}
	return strconv.Itoa(*i)
}
