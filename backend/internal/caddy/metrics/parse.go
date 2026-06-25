// Package metrics provides helpers for scraping and parsing the Caddy Prometheus
// exposition format emitted at the admin API /metrics endpoint.
package metrics

import (
	"fmt"
	"io"
	"math"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

// u64ToI64 converts a Prometheus uint64 count to int64, clamping at the int64
// ceiling for overflow safety (real metric counts never approach this bound).
func u64ToI64(v uint64) int64 {
	if v > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(v)
}

// Snapshot is an aggregated view of one scrape from the Caddy metrics endpoint.
// It mirrors the numeric columns of models.TrafficSample but carries no
// database or GORM concerns.
type Snapshot struct {
	Req2xx   int64
	Req3xx   int64
	Req4xx   int64
	Req5xx   int64
	ReqOther int64

	BytesIn  int64
	BytesOut int64

	// DurationBuckets maps the le label value to the cumulative bucket count
	// summed across all series (handler/method/etc.).
	DurationBuckets map[string]int64
	DurationSum     float64
	DurationCount   int64

	InFlight int64
}

// ParseSnapshot reads a Prometheus text-exposition body and returns an
// aggregated Snapshot. Metrics from ALL label combinations (handler, method,
// code, …) are summed together.
func ParseSnapshot(r io.Reader) (*Snapshot, error) {
	parser := expfmt.NewTextParser(model.LegacyValidation)
	families, err := parser.TextToMetricFamilies(r)
	if err != nil && len(families) == 0 {
		return nil, fmt.Errorf("metrics: parse error: %w", err)
	}

	snap := &Snapshot{
		DurationBuckets: make(map[string]int64),
	}

	for name, mf := range families {
		switch name {
		case "caddy_http_request_size_bytes":
			// histogram; grab _sum
			for _, m := range mf.GetMetric() {
				if h := m.GetHistogram(); h != nil {
					snap.BytesIn += int64(h.GetSampleSum())
				}
			}

		case "caddy_http_response_size_bytes":
			// histogram; grab _sum
			for _, m := range mf.GetMetric() {
				if h := m.GetHistogram(); h != nil {
					snap.BytesOut += int64(h.GetSampleSum())
				}
			}

		case "caddy_http_request_duration_seconds":
			// The status-code breakdown is derived from this histogram's
			// per-series _count, bucketed by the first digit of the `code`
			// label. Caddy's caddy_http_requests_total has no `code` label, so
			// it cannot drive the class breakdown.
			for _, m := range mf.GetMetric() {
				h := m.GetHistogram()
				if h == nil {
					continue
				}
				count := u64ToI64(h.GetSampleCount())
				addByCodeClass(snap, codeLabel(m), count)
				snap.DurationSum += h.GetSampleSum()
				snap.DurationCount += count
				for _, b := range h.GetBucket() {
					le := fmt.Sprintf("%g", b.GetUpperBound())
					snap.DurationBuckets[le] += u64ToI64(b.GetCumulativeCount())
				}
			}

		case "caddy_http_requests_in_flight":
			for _, m := range mf.GetMetric() {
				if g := m.GetGauge(); g != nil {
					snap.InFlight += int64(g.GetValue())
				}
			}
		}
	}

	return snap, nil
}

// codeLabel returns the value of the `code` label on a metric series, or an
// empty string if the label is absent.
func codeLabel(m *dto.Metric) string {
	for _, lp := range m.GetLabel() {
		if lp.GetName() == "code" {
			return lp.GetValue()
		}
	}
	return ""
}

// addByCodeClass adds count to the snapshot's request-class counter selected by
// the first digit of the HTTP response code. An empty or non-numeric code falls
// into ReqOther.
func addByCodeClass(snap *Snapshot, code string, count int64) {
	if len(code) == 0 {
		snap.ReqOther += count
		return
	}
	switch code[0] {
	case '2':
		snap.Req2xx += count
	case '3':
		snap.Req3xx += count
	case '4':
		snap.Req4xx += count
	case '5':
		snap.Req5xx += count
	default:
		snap.ReqOther += count
	}
}
