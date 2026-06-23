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
		case "caddy_http_requests_total":
			// Direct assignment (not +=): expfmt always emits caddy_http_requests_total
			// as exactly one family, so accumulating would double-count on future iterations.
			snap.Req2xx, snap.Req3xx, snap.Req4xx, snap.Req5xx, snap.ReqOther =
				aggregateRequestCounts(mf)

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
			for _, m := range mf.GetMetric() {
				h := m.GetHistogram()
				if h == nil {
					continue
				}
				snap.DurationSum += h.GetSampleSum()
				snap.DurationCount += u64ToI64(h.GetSampleCount())
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

// aggregateRequestCounts sums caddy_http_requests_total across all label
// combinations, bucketing by the first digit of the HTTP response code label.
func aggregateRequestCounts(mf *dto.MetricFamily) (r2xx, r3xx, r4xx, r5xx, rother int64) {
	for _, m := range mf.GetMetric() {
		var code string
		for _, lp := range m.GetLabel() {
			if lp.GetName() == "code" {
				code = lp.GetValue()
				break
			}
		}

		var count int64
		if c := m.GetCounter(); c != nil {
			count = int64(c.GetValue())
		}

		if len(code) == 0 {
			rother += count
			continue
		}
		switch code[0] {
		case '2':
			r2xx += count
		case '3':
			r3xx += count
		case '4':
			r4xx += count
		case '5':
			r5xx += count
		default:
			rother += count
		}
	}
	return
}
