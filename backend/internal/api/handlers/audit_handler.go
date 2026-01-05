package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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

// UpdateConfig updates the audit configuration
func (h *AuditHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var config models.AuditConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}

	if err := h.auditService.SetConfig(&config); err != nil {
		utils.InternalError(w, "Failed to update audit configuration")
		return
	}

	utils.Success(w, &config, "Audit configuration updated successfully")
}

// GetEventGroups returns the available audit event groups for configuration UI
func (h *AuditHandler) GetEventGroups(w http.ResponseWriter, r *http.Request) {
	groups := models.GetAuditEventGroups()
	utils.Success(w, groups, "Audit event groups retrieved successfully")
}

// Supported filter operators
const (
	OpEq         = "eq"          // equals (default)
	OpNot        = "not"         // not equals
	OpIn         = "in"          // in list
	OpNotIn      = "not_in"      // not in list
	OpContains   = "contains"    // contains (text)
	OpStartsWith = "starts_with" // starts with (text)
	OpEndsWith   = "ends_with"   // ends with (text)
)

// filterValue holds parsed operator and value from query param
type filterValue struct {
	Operator string
	Value    string
	Values   []string // For in/not_in operators
}

// parseFilterParam parses a filter parameter in format "operator:value" or just "value"
// If no operator is specified, defaults to "eq" for single values
// Examples:
//   - "ends_with:121" -> {Operator: "ends_with", Value: "121"}
//   - "in:a,b,c" -> {Operator: "in", Values: ["a", "b", "c"]}
//   - "success" -> {Operator: "eq", Value: "success"}
//   - "http://example.com" -> {Operator: "eq", Value: "http://example.com"} (colon in URL preserved)
func parseFilterParam(param string) filterValue {
	if param == "" {
		return filterValue{}
	}

	// Check for operator prefix (operator:value format)
	// Only split on first colon, and only if it looks like a known operator
	colonIdx := strings.Index(param, ":")
	if colonIdx > 0 {
		possibleOp := param[:colonIdx]
		// Check if it's a valid operator
		switch possibleOp {
		case OpEq, OpNot, OpIn, OpNotIn, OpContains, OpStartsWith, OpEndsWith:
			value := param[colonIdx+1:]
			fv := filterValue{Operator: possibleOp, Value: value}
			// For in/not_in operators, split into multiple values
			if possibleOp == OpIn || possibleOp == OpNotIn {
				fv.Values = splitAndTrim(value)
			}
			return fv
		}
	}

	// No valid operator found, default to "eq"
	return filterValue{Operator: OpEq, Value: param}
}

// parseAuditListParams parses query parameters for audit log listing
// Filter format: field=operator:value (e.g., ip_address=ends_with:121)
// If no operator specified, defaults to "eq" (e.g., status=success)
// Supported operators: eq, not, in, not_in, contains, starts_with, ends_with
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

	// Parse search (simple text search, no operator)
	params.Search = r.URL.Query().Get("search")

	// Parse action filter
	if action := r.URL.Query().Get("action"); action != "" {
		fv := parseFilterParam(action)
		switch fv.Operator {
		case OpIn, OpEq:
			if len(fv.Values) > 0 {
				params.Actions = fv.Values
			} else {
				params.Actions = splitAndTrim(fv.Value)
			}
		case OpNotIn, OpNot:
			if len(fv.Values) > 0 {
				params.ActionsExclude = fv.Values
			} else {
				params.ActionsExclude = splitAndTrim(fv.Value)
			}
		default:
			return params, fmt.Errorf("invalid operator '%s' for action filter", fv.Operator)
		}
	}

	// Parse resource_type filter
	if resourceType := r.URL.Query().Get("resource_type"); resourceType != "" {
		fv := parseFilterParam(resourceType)
		switch fv.Operator {
		case OpIn, OpEq:
			if len(fv.Values) > 0 {
				params.ResourceTypes = fv.Values
			} else {
				params.ResourceTypes = splitAndTrim(fv.Value)
			}
		case OpNotIn, OpNot:
			if len(fv.Values) > 0 {
				params.ResourceTypesExclude = fv.Values
			} else {
				params.ResourceTypesExclude = splitAndTrim(fv.Value)
			}
		default:
			return params, fmt.Errorf("invalid operator '%s' for resource_type filter", fv.Operator)
		}
	}

	// Parse ip_address filter
	if ipAddress := r.URL.Query().Get("ip_address"); ipAddress != "" {
		fv := parseFilterParam(ipAddress)
		switch fv.Operator {
		case OpEq:
			params.IPAddress = fv.Value
		case OpNot:
			params.IPAddressNot = fv.Value
		case OpContains:
			params.IPAddressContains = fv.Value
		case OpStartsWith:
			params.IPAddressStartsWith = fv.Value
		case OpEndsWith:
			params.IPAddressEndsWith = fv.Value
		default:
			return params, fmt.Errorf("invalid operator '%s' for ip_address filter", fv.Operator)
		}
	}

	// Parse user_id filter
	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		fv := parseFilterParam(userIDStr)
		if fv.Operator != OpEq {
			return params, fmt.Errorf("user_id only supports 'eq' operator")
		}
		userID, err := strconv.Atoi(fv.Value)
		if err != nil {
			return params, fmt.Errorf("invalid user_id")
		}
		params.UserID = &userID
	}

	// Parse status filter
	if status := r.URL.Query().Get("status"); status != "" {
		fv := parseFilterParam(status)
		// Validate status value
		statusVal := fv.Value
		if statusVal != "success" && statusVal != "failure" {
			return params, fmt.Errorf("status must be 'success' or 'failure'")
		}
		switch fv.Operator {
		case OpEq:
			params.Status = statusVal
		case OpNot:
			params.StatusExclude = statusVal
		default:
			return params, fmt.Errorf("invalid operator '%s' for status filter", fv.Operator)
		}
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

// splitAndTrim splits a comma-separated string and trims whitespace from each part
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
