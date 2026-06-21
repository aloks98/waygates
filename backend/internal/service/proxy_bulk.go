package service

// BulkItemResult is the per-id outcome of a bulk operation.
// Status is either "ok" or "error"; Error carries the failure message when Status is "error".
type BulkItemResult struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// BulkResult summarizes the outcome of a bulk operation across a set of ids.
type BulkResult struct {
	Requested int              `json:"requested"`
	Succeeded int              `json:"succeeded"`
	Failed    int              `json:"failed"`
	Results   []BulkItemResult `json:"results"`
}

// Bulk operation status values.
const (
	bulkStatusOK    = "ok"
	bulkStatusError = "error"
)

// BulkSetActive enables or disables each proxy in ids. It is best-effort: a
// failure on one id never aborts the batch. Each id contributes one entry to
// Results with its individual outcome.
func (s *ProxyService) BulkSetActive(ids []int, enable bool) BulkResult {
	op := s.DisableProxy
	if enable {
		op = s.EnableProxy
	}
	return s.runBulk(ids, op)
}

// BulkDelete deletes each proxy in ids. It is best-effort: a failure on one id
// never aborts the batch.
func (s *ProxyService) BulkDelete(ids []int) BulkResult {
	return s.runBulk(ids, s.DeleteProxy)
}

// runBulk applies op to every id, collecting a per-id result and aggregate counts.
func (s *ProxyService) runBulk(ids []int, op func(id int) error) BulkResult {
	result := BulkResult{
		Requested: len(ids),
		Results:   make([]BulkItemResult, 0, len(ids)),
	}

	for _, id := range ids {
		if err := op(id); err != nil {
			result.Failed++
			result.Results = append(result.Results, BulkItemResult{
				ID:     id,
				Status: bulkStatusError,
				Error:  err.Error(),
			})
			continue
		}
		result.Succeeded++
		result.Results = append(result.Results, BulkItemResult{
			ID:     id,
			Status: bulkStatusOK,
		})
	}

	return result
}
