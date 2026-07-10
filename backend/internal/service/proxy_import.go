package service

import (
	"errors"
	"fmt"

	"github.com/aloks98/waygates/backend/internal/models"
)

// Import per-item statuses.
const (
	ImportStatusValid           = "valid"
	ImportStatusConflict        = "conflict"
	ImportStatusInvalid         = "invalid"
	ImportStatusCreated         = "created"
	ImportStatusSkippedConflict = "skipped_conflict"
	ImportStatusFailed          = "failed"
)

// ImportInput is one decoded item. Proxy is nil when the raw item could not be
// decoded, in which case DecodeError explains why.
type ImportInput struct {
	Proxy       *models.Proxy
	DecodeError string
}

type ImportItemResult struct {
	Index    int    `json:"index"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
}

type ImportSummary struct {
	Total      int `json:"total"`
	Importable int `json:"importable"`
	Conflicts  int `json:"conflicts"`
	Invalid    int `json:"invalid"`
	Created    int `json:"created"`
	Failed     int `json:"failed"`
}

type ImportReport struct {
	Summary ImportSummary      `json:"summary"`
	Items   []ImportItemResult `json:"items"`
}

// ImportProxies validates and (when dryRun is false) creates the given proxies,
// skipping conflicts and invalid items. Best-effort: one failure never aborts
// the rest. Dry-run performs the identical validation/conflict checks but writes
// nothing, so it is an exact preview of apply.
func (s *ProxyService) ImportProxies(inputs []ImportInput, dryRun bool, userID int) ImportReport {
	report := ImportReport{Items: make([]ImportItemResult, 0, len(inputs))}
	seen := make(map[string]bool) // hostnames already accepted (valid/created) in this batch

	// Look up which hostnames already exist in ONE query rather than calling
	// HostnameExists per item (N+1). hostErr is surfaced per-item below so the
	// report shape (and best-effort semantics) is preserved.
	hostnames := make([]string, 0, len(inputs))
	for _, in := range inputs {
		if in.Proxy != nil {
			hostnames = append(hostnames, in.Proxy.Hostname)
		}
	}
	existing, hostErr := s.repo.ExistingHostnames(hostnames)

	conflictStatus := ImportStatusConflict
	if !dryRun {
		conflictStatus = ImportStatusSkippedConflict
	}

	for i, in := range inputs {
		res := ImportItemResult{Index: i}

		if in.Proxy == nil {
			res.Status = ImportStatusInvalid
			res.Reason = in.DecodeError
			report.Items = append(report.Items, res)
			continue
		}

		proxy := in.Proxy
		res.Name = proxy.Name
		res.Type = proxy.Type

		// materializeHostname must run before Validate, same as CreateProxy /
		// UpdateProxy: a label-addressed item has no raw Hostname until this
		// resolves it against its group's base_domain.
		if err := s.materializeHostname(proxy); err != nil {
			res.Hostname = proxy.Hostname
			res.Status = ImportStatusInvalid
			res.Reason = err.Error()
			report.Items = append(report.Items, res)
			continue
		}
		res.Hostname = proxy.Hostname

		if err := proxy.Validate(); err != nil {
			res.Status = ImportStatusInvalid
			res.Reason = err.Error()
			report.Items = append(report.Items, res)
			continue
		}

		if seen[proxy.Hostname] {
			res.Status = conflictStatus
			res.Reason = "duplicate hostname in import"
			report.Items = append(report.Items, res)
			continue
		}
		if hostErr != nil {
			res.Status = ImportStatusFailed
			res.Reason = fmt.Sprintf("failed to check hostname: %v", hostErr)
			report.Items = append(report.Items, res)
			continue
		}
		if existing[proxy.Hostname] {
			res.Status = conflictStatus
			res.Reason = "hostname already exists"
			report.Items = append(report.Items, res)
			continue
		}

		if dryRun {
			res.Status = ImportStatusValid
			seen[proxy.Hostname] = true
			report.Items = append(report.Items, res)
			continue
		}

		if err := s.CreateProxy(proxy, userID); err != nil {
			if errors.Is(err, ErrHostnameConflict) {
				res.Status = ImportStatusSkippedConflict
				res.Reason = "hostname already exists"
			} else {
				res.Status = ImportStatusFailed
				res.Reason = err.Error()
			}
			report.Items = append(report.Items, res)
			continue
		}
		res.Status = ImportStatusCreated
		seen[proxy.Hostname] = true
		report.Items = append(report.Items, res)
	}

	report.Summary = summarizeImport(report.Items, len(inputs))
	return report
}

func summarizeImport(items []ImportItemResult, total int) ImportSummary {
	s := ImportSummary{Total: total}
	for _, it := range items {
		switch it.Status {
		case ImportStatusValid:
			s.Importable++
		case ImportStatusCreated:
			s.Importable++
			s.Created++
		case ImportStatusFailed:
			s.Importable++
			s.Failed++
		case ImportStatusConflict, ImportStatusSkippedConflict:
			s.Conflicts++
		case ImportStatusInvalid:
			s.Invalid++
		}
	}
	return s
}
