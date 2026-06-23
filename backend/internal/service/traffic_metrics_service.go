package service

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// ErrInvalidRange is returned when the caller supplies a range string that is
// not one of the recognized values ("1h", "24h", "7d").
var ErrInvalidRange = errors.New("invalid range")

// TrafficSeries is the top-level response returned by GetTraffic.
type TrafficSeries struct {
	Range       string         `json:"range"`
	StepSeconds int            `json:"step_seconds"`
	Points      []TrafficPoint `json:"points"`
}

// TrafficPoint holds the aggregated metrics for one output time bucket.
type TrafficPoint struct {
	T        time.Time `json:"t"`
	Req2xx   int64     `json:"req_2xx"`
	Req3xx   int64     `json:"req_3xx"`
	Req4xx   int64     `json:"req_4xx"`
	Req5xx   int64     `json:"req_5xx"`
	ReqOther int64     `json:"req_other"`
	BytesIn  int64     `json:"bytes_in"`
	BytesOut int64     `json:"bytes_out"`
	P50Ms    float64   `json:"p50_ms"`
	P95Ms    float64   `json:"p95_ms"`
	InFlight int64     `json:"in_flight"`
}

// histBucket is a (le, cumulative-count) pair used internally for quantile math.
type histBucket struct {
	le  float64
	cnt int64
}

// bucketAccumulator accumulates per-step deltas before being converted to a TrafficPoint.
type bucketAccumulator struct {
	req2xx   int64
	req3xx   int64
	req4xx   int64
	req5xx   int64
	reqOther int64
	bytesIn  int64
	bytesOut int64
	// buckets is the per-le cumulative-delta sum across all pairs landing in this step bucket.
	buckets  map[string]int64
	inFlight int64 // last observed in-flight (gauge)
}

// rangeConfig maps a range string to its duration and output step size.
type rangeConfig struct {
	duration    time.Duration
	stepSeconds int
}

var rangeConfigs = map[string]rangeConfig{
	"1h":  {duration: 1 * time.Hour, stepSeconds: 30},
	"24h": {duration: 24 * time.Hour, stepSeconds: 300},
	"7d":  {duration: 7 * 24 * time.Hour, stepSeconds: 3600},
}

// TrafficMetricsService computes the traffic time-series from stored samples.
type TrafficMetricsService struct {
	repo   repository.TrafficSampleRepositoryInterface
	logger *zap.Logger
}

// NewTrafficMetricsService creates a new TrafficMetricsService.
func NewTrafficMetricsService(repo repository.TrafficSampleRepositoryInterface, logger *zap.Logger) *TrafficMetricsService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &TrafficMetricsService{repo: repo, logger: logger}
}

// GetTraffic validates the range, fetches samples from the repository, and
// delegates to computeSeries. An invalid range returns a descriptive error.
func (s *TrafficMetricsService) GetTraffic(rng string) (*TrafficSeries, error) {
	if rng == "" {
		rng = "1h"
	}
	cfg, ok := rangeConfigs[rng]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrInvalidRange, rng)
	}
	since := time.Now().Add(-cfg.duration)
	samples, err := s.repo.ListSince(since)
	if err != nil {
		return nil, fmt.Errorf("failed to list traffic samples: %w", err)
	}
	return computeSeries(samples, rng)
}

// computeSeries turns a slice of ASC-ordered cumulative samples into a TrafficSeries.
// It is a pure function (no DB, no time.Now) so it can be unit-tested directly.
func computeSeries(samples []models.TrafficSample, rng string) (*TrafficSeries, error) {
	cfg, ok := rangeConfigs[rng]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrInvalidRange, rng)
	}
	step := time.Duration(cfg.stepSeconds) * time.Second

	// Use a slice for ordered keys and a map for O(1) access to accumulators.
	bucketKeys := make([]time.Time, 0, len(samples))
	buckets := make(map[time.Time]*bucketAccumulator)

	for i := 1; i < len(samples); i++ {
		prev := samples[i-1]
		cur := samples[i]

		// Counter-reset detection: if DurationCount dropped, treat the entire pair as zero.
		reset := cur.DurationCount < prev.DurationCount

		var (
			d2xx, d3xx, d4xx, d5xx, dOther int64
			dBytesIn, dBytesOut            int64
			dBuckets                       map[string]int64
		)

		if !reset {
			d2xx = clampNeg(cur.Req2xx - prev.Req2xx)
			d3xx = clampNeg(cur.Req3xx - prev.Req3xx)
			d4xx = clampNeg(cur.Req4xx - prev.Req4xx)
			d5xx = clampNeg(cur.Req5xx - prev.Req5xx)
			dOther = clampNeg(cur.ReqOther - prev.ReqOther)
			dBytesIn = clampNeg(cur.BytesIn - prev.BytesIn)
			dBytesOut = clampNeg(cur.BytesOut - prev.BytesOut)
			dBuckets = bucketDeltas(prev.DurationBuckets, cur.DurationBuckets)
		} else {
			dBuckets = make(map[string]int64)
		}

		// Bucket boundary: truncate cur.CollectedAt to the step boundary.
		boundary := cur.CollectedAt.Truncate(step)
		acc, exists := buckets[boundary]
		if !exists {
			acc = &bucketAccumulator{
				buckets: make(map[string]int64),
			}
			buckets[boundary] = acc
			bucketKeys = append(bucketKeys, boundary)
		}

		acc.req2xx += d2xx
		acc.req3xx += d3xx
		acc.req4xx += d4xx
		acc.req5xx += d5xx
		acc.reqOther += dOther
		acc.bytesIn += dBytesIn
		acc.bytesOut += dBytesOut
		// in_flight is a gauge: always use the latest cur value.
		acc.inFlight = cur.InFlight
		for le, cnt := range dBuckets {
			acc.buckets[le] += cnt
		}
	}

	// Sort bucket boundaries ascending.
	sort.Slice(bucketKeys, func(i, j int) bool {
		return bucketKeys[i].Before(bucketKeys[j])
	})

	points := make([]TrafficPoint, 0, len(bucketKeys))
	for _, boundary := range bucketKeys {
		acc := buckets[boundary]
		p50, p95 := computeQuantiles(acc.buckets)
		points = append(points, TrafficPoint{
			T:        boundary,
			Req2xx:   acc.req2xx,
			Req3xx:   acc.req3xx,
			Req4xx:   acc.req4xx,
			Req5xx:   acc.req5xx,
			ReqOther: acc.reqOther,
			BytesIn:  acc.bytesIn,
			BytesOut: acc.bytesOut,
			P50Ms:    p50,
			P95Ms:    p95,
			InFlight: acc.inFlight,
		})
	}

	return &TrafficSeries{
		Range:       rng,
		StepSeconds: cfg.stepSeconds,
		Points:      points,
	}, nil
}

// clampNeg returns delta if >= 0, otherwise 0.
func clampNeg(delta int64) int64 {
	if delta < 0 {
		return 0
	}
	return delta
}

// bucketDeltas returns per-le delta values (cur − prev) clamped to zero for
// any negative values. Only keys present in cur are included in the result.
func bucketDeltas(prev, cur models.DurationBucketsField) map[string]int64 {
	result := make(map[string]int64, len(cur))
	for le, curCount := range cur {
		prevCount := prev[le]
		d := curCount - prevCount
		if d < 0 {
			d = 0
		}
		result[le] = d
	}
	return result
}

// computeQuantiles computes p50 and p95 in milliseconds from cumulative
// histogram bucket deltas (le → cumulative observation count within this step).
// Returns (0, 0) when there are no observations.
func computeQuantiles(bucketDeltas map[string]int64) (p50Ms, p95Ms float64) {
	if len(bucketDeltas) == 0 {
		return 0, 0
	}

	parsed := make([]histBucket, 0, len(bucketDeltas))
	for leStr, cnt := range bucketDeltas {
		var leVal float64
		if leStr == "+Inf" {
			leVal = math.Inf(1)
		} else {
			var f float64
			if _, err := fmt.Sscanf(leStr, "%g", &f); err != nil {
				continue
			}
			leVal = f
		}
		parsed = append(parsed, histBucket{le: leVal, cnt: cnt})
	}
	if len(parsed) == 0 {
		return 0, 0
	}
	sort.Slice(parsed, func(i, j int) bool {
		return parsed[i].le < parsed[j].le
	})

	// Guard: the highest bucket must be +Inf; if it is absent (corrupt sample),
	// return zeros rather than silently computing wrong quantiles.
	if !math.IsInf(parsed[len(parsed)-1].le, 1) {
		return 0, 0
	}

	// total is the count at the +Inf bucket (highest le).
	total := parsed[len(parsed)-1].cnt
	if total == 0 {
		return 0, 0
	}

	p50Ms = histogramQuantile(0.50, parsed, total) * 1000
	p95Ms = histogramQuantile(0.95, parsed, total) * 1000
	return p50Ms, p95Ms
}

// histogramQuantile applies the standard Prometheus linear-interpolation
// algorithm. parsed must be sorted ascending by le; total = parsed[last].cnt.
func histogramQuantile(q float64, parsed []histBucket, total int64) float64 {
	rank := q * float64(total)

	// Largest finite le for clamping when rank falls in the +Inf bucket.
	largestFinite := 0.0
	for _, b := range parsed {
		if !math.IsInf(b.le, 1) && b.le > largestFinite {
			largestFinite = b.le
		}
	}

	var lowerBound float64
	var lowerCount int64
	for _, b := range parsed {
		if float64(b.cnt) >= rank {
			if math.IsInf(b.le, 1) {
				// All finite buckets had count < rank; clamp to largest finite le.
				return largestFinite
			}
			upperBound := b.le
			upperCount := b.cnt
			if upperCount == lowerCount {
				// Avoid divide-by-zero.
				return lowerBound
			}
			return lowerBound + (rank-float64(lowerCount))/float64(upperCount-lowerCount)*(upperBound-lowerBound)
		}
		lowerBound = b.le
		lowerCount = b.cnt
	}
	return largestFinite
}
