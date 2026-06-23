package service

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
)

// baseTime is an arbitrary fixed time used as an anchor for test samples.
var baseTime = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

// makeSample is a convenience constructor for test TrafficSample rows.
func makeSample(secsOffset int, r2xx, r3xx, r4xx, r5xx, rOther, bIn, bOut int64,
	dCount int64, dSum float64, buckets map[string]int64, inFlight int64,
) models.TrafficSample {
	b := make(models.DurationBucketsField, len(buckets))
	for k, v := range buckets {
		b[k] = v
	}
	return models.TrafficSample{
		CollectedAt:     baseTime.Add(time.Duration(secsOffset) * time.Second),
		Req2xx:          r2xx,
		Req3xx:          r3xx,
		Req4xx:          r4xx,
		Req5xx:          r5xx,
		ReqOther:        rOther,
		BytesIn:         bIn,
		BytesOut:        bOut,
		DurationCount:   dCount,
		DurationSum:     dSum,
		DurationBuckets: b,
		InFlight:        inFlight,
	}
}

// knownBuckets are the standard Caddy histogram le labels used in tests.
var knownBuckets = []string{
	"0.005", "0.01", "0.025", "0.05", "0.1",
	"0.25", "0.5", "1", "2.5", "5", "10", "+Inf",
}

// zeroBuckets returns a map with all known le labels set to 0.
func zeroBuckets() map[string]int64 {
	m := make(map[string]int64, len(knownBuckets))
	for _, le := range knownBuckets {
		m[le] = 0
	}
	return m
}

// addBuckets returns a copy of base with values added from delta.
func addBuckets(base, delta map[string]int64) map[string]int64 {
	m := make(map[string]int64, len(base))
	for k, v := range base {
		m[k] = v
	}
	for k, v := range delta {
		m[k] += v
	}
	return m
}

// ─── Range validation ────────────────────────────────────────────────────────

func TestComputeSeries_InvalidRange(t *testing.T) {
	_, err := computeSeries(nil, "invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid range")
}

func TestComputeSeries_EmptyRange_DefaultsTo1h(t *testing.T) {
	// computeSeries with "" range should error (caller's GetTraffic handles default).
	_, err := computeSeries(nil, "")
	require.Error(t, err)
}

func TestGetTraffic_InvalidRange(t *testing.T) {
	svc := NewTrafficMetricsService(nil, nil)
	// Range validation fires before any repo access, so nil repo is safe here.
	_, err := svc.GetTraffic("bad")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRange), "error must wrap ErrInvalidRange")
	assert.Contains(t, err.Error(), "invalid range")
}

// ─── Empty / single sample ───────────────────────────────────────────────────

func TestComputeSeries_EmptySamples(t *testing.T) {
	series, err := computeSeries([]models.TrafficSample{}, "1h")
	require.NoError(t, err)
	assert.Equal(t, "1h", series.Range)
	assert.Equal(t, 30, series.StepSeconds)
	assert.Empty(t, series.Points)
}

func TestComputeSeries_SingleSample(t *testing.T) {
	// With only one sample there are no consecutive pairs, so no points.
	samples := []models.TrafficSample{
		makeSample(0, 100, 5, 10, 2, 1, 5000, 8000, 118, 3.5, zeroBuckets(), 3),
	}
	series, err := computeSeries(samples, "1h")
	require.NoError(t, err)
	assert.Empty(t, series.Points)
}

// ─── Delta math + downsample ─────────────────────────────────────────────────

func TestComputeSeries_DeltaMath_1h(t *testing.T) {
	// Two samples 30 s apart → one delta → one output point (1h step=30s).
	b0 := map[string]int64{"0.05": 3, "0.1": 7, "0.25": 9, "+Inf": 10}
	b1 := map[string]int64{"0.05": 5, "0.1": 9, "0.25": 11, "+Inf": 13}

	samples := []models.TrafficSample{
		makeSample(0, 100, 5, 10, 2, 1, 5000, 8000, 10, 1.0, b0, 3),
		makeSample(30, 115, 6, 12, 3, 2, 5300, 8200, 13, 1.3, b1, 4),
	}

	series, err := computeSeries(samples, "1h")
	require.NoError(t, err)
	require.Len(t, series.Points, 1)
	pt := series.Points[0]

	assert.Equal(t, int64(15), pt.Req2xx)
	assert.Equal(t, int64(1), pt.Req3xx)
	assert.Equal(t, int64(2), pt.Req4xx)
	assert.Equal(t, int64(1), pt.Req5xx)
	assert.Equal(t, int64(1), pt.ReqOther)
	assert.Equal(t, int64(300), pt.BytesIn)
	assert.Equal(t, int64(200), pt.BytesOut)
	assert.Equal(t, int64(4), pt.InFlight)
}

func TestComputeSeries_MultipleDeltas_SummedInBucket(t *testing.T) {
	// Three samples 10 s apart, all truncating into the same 30 s bucket.
	// Net delta over the two pairs should be the sum.
	b := zeroBuckets()

	samples := []models.TrafficSample{
		makeSample(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, b, 0),
		makeSample(10, 5, 0, 0, 0, 0, 100, 200, 5, 0.5, b, 2),
		makeSample(20, 12, 0, 0, 0, 0, 250, 400, 12, 1.1, b, 3),
	}

	series, err := computeSeries(samples, "1h")
	require.NoError(t, err)
	require.Len(t, series.Points, 1)
	pt := series.Points[0]

	assert.Equal(t, int64(12), pt.Req2xx)
	assert.Equal(t, int64(250), pt.BytesIn)
	// in_flight = last cur.InFlight = 3
	assert.Equal(t, int64(3), pt.InFlight)
}

func TestComputeSeries_DeltasAcrossMultipleBuckets(t *testing.T) {
	// Samples at t=0, t=30, t=60 → two deltas landing in two different 30-s buckets.
	//
	//   pair 0→1: cur at t=30  → boundary = baseTime+30s (truncated to 30s)
	//   pair 1→2: cur at t=60  → boundary = baseTime+60s (truncated to 30s)
	b := zeroBuckets()

	samples := []models.TrafficSample{
		makeSample(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, b, 1),
		makeSample(30, 10, 0, 0, 0, 0, 1000, 2000, 10, 1.0, b, 2),
		makeSample(60, 25, 0, 0, 0, 0, 2500, 5000, 25, 2.5, b, 5),
	}

	series, err := computeSeries(samples, "1h")
	require.NoError(t, err)
	require.Len(t, series.Points, 2)

	step := 30 * time.Second
	wantBoundary0 := baseTime.Add(30 * time.Second).Truncate(step)
	wantBoundary1 := baseTime.Add(60 * time.Second).Truncate(step)

	// Points must be ordered ascending by boundary.
	assert.Equal(t, wantBoundary0, series.Points[0].T, "first point bucket boundary")
	assert.Equal(t, wantBoundary1, series.Points[1].T, "second point bucket boundary")

	assert.Equal(t, int64(10), series.Points[0].Req2xx)
	assert.Equal(t, int64(15), series.Points[1].Req2xx)

	// BytesIn deltas: pair 0→1 → 1000, pair 1→2 → 1500
	assert.Equal(t, int64(1000), series.Points[0].BytesIn)
	assert.Equal(t, int64(1500), series.Points[1].BytesIn)

	// BytesOut deltas: pair 0→1 → 2000, pair 1→2 → 3000
	assert.Equal(t, int64(2000), series.Points[0].BytesOut)
	assert.Equal(t, int64(3000), series.Points[1].BytesOut)
}

// ─── Counter-reset handling ───────────────────────────────────────────────────

func TestComputeSeries_CounterReset_ContributesZero(t *testing.T) {
	// A sample where DurationCount drops (counter reset) should yield no delta.
	bHigh := map[string]int64{"0.05": 100, "0.1": 200, "+Inf": 300}
	bLow := map[string]int64{"0.05": 5, "0.1": 10, "+Inf": 15} // reset

	samples := []models.TrafficSample{
		makeSample(0, 1000, 50, 20, 5, 2, 50000, 80000, 300, 30.0, bHigh, 10),
		makeSample(30, 10, 1, 0, 0, 0, 500, 1000, 15, 1.5, bLow, 2), // DurationCount 300→15: reset
	}

	series, err := computeSeries(samples, "1h")
	require.NoError(t, err)
	require.Len(t, series.Points, 1)
	pt := series.Points[0]

	// All counter deltas must be 0 for the reset pair.
	assert.Equal(t, int64(0), pt.Req2xx)
	assert.Equal(t, int64(0), pt.Req3xx)
	assert.Equal(t, int64(0), pt.Req4xx)
	assert.Equal(t, int64(0), pt.Req5xx)
	assert.Equal(t, int64(0), pt.ReqOther)
	assert.Equal(t, int64(0), pt.BytesIn)
	assert.Equal(t, int64(0), pt.BytesOut)
	// in_flight is still the cur gauge.
	assert.Equal(t, int64(2), pt.InFlight)
	// Quantiles should be zero (no histogram deltas).
	assert.Equal(t, float64(0), pt.P50Ms)
	assert.Equal(t, float64(0), pt.P95Ms)
}

func TestComputeSeries_CounterReset_OnlyOnePairAffected(t *testing.T) {
	// Normal pair, reset pair, normal pair — only the reset pair contributes 0.
	// Samples land in separate 30-s boundaries so each pair gets its own point.
	//
	//   t=0   → t=30 : pair 0→1, boundary=t=0  truncated to :00, +10 req2xx
	//   t=30  → t=60 : pair 1→2, boundary=t=30 truncated to :30, reset  → 0
	//   t=60  → t=90 : pair 2→3, boundary=t=60 truncated to :00 of next min, +8 req2xx
	b0 := zeroBuckets()
	b1 := addBuckets(b0, map[string]int64{"0.05": 3, "+Inf": 3})
	b2 := zeroBuckets() // reset
	b3 := addBuckets(b2, map[string]int64{"0.05": 5, "+Inf": 5})

	samples := []models.TrafficSample{
		// pair 0→1: normal, +10 req2xx
		makeSample(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, b0, 0),
		makeSample(30, 10, 0, 0, 0, 0, 0, 0, 3, 0.3, b1, 1),
		// pair 1→2: reset (DurationCount drops 3→0)
		makeSample(60, 0, 0, 0, 0, 0, 0, 0, 0, 0, b2, 0),
		// pair 2→3: normal, +8 req2xx
		makeSample(90, 8, 0, 0, 0, 0, 0, 0, 5, 0.5, b3, 2),
	}

	series, err := computeSeries(samples, "1h")
	require.NoError(t, err)

	// Collect points indexed by their boundary time so the assertions are
	// order-independent (though they should be ascending).
	pointsByBoundary := make(map[time.Time]TrafficPoint, len(series.Points))
	for _, pt := range series.Points {
		pointsByBoundary[pt.T] = pt
	}

	// pair 0→1: cur sample is at baseTime+30s; boundary = (baseTime+30s).Truncate(30s)
	b01 := baseTime.Add(30 * time.Second).Truncate(30 * time.Second)
	if pt, ok := pointsByBoundary[b01]; ok {
		assert.Equal(t, int64(10), pt.Req2xx, "pair 0→1 should contribute exactly 10 req2xx")
	}

	// pair 1→2: boundary = (baseTime+60s).Truncate(30s) = baseTime+60s — reset → 0
	b12 := baseTime.Add(60 * time.Second).Truncate(30 * time.Second)
	if pt, ok := pointsByBoundary[b12]; ok {
		assert.Equal(t, int64(0), pt.Req2xx, "reset pair 1→2 should contribute 0 req2xx")
	}

	// pair 2→3: boundary = (baseTime+90s).Truncate(30s) = baseTime+90s — +8
	b23 := baseTime.Add(90 * time.Second).Truncate(30 * time.Second)
	if pt, ok := pointsByBoundary[b23]; ok {
		assert.Equal(t, int64(8), pt.Req2xx, "pair 2→3 should contribute exactly 8 req2xx")
	}

	// Totals still hold.
	var totalReq2xx int64
	for _, pt := range series.Points {
		totalReq2xx += pt.Req2xx
	}
	// Only pairs 0→1 (+10) and 2→3 (+8) contribute; pair 1→2 (reset) contributes 0.
	assert.Equal(t, int64(18), totalReq2xx)
}

// ─── Negative delta clamping (non-reset) ────────────────────────────────────

func TestComputeSeries_NegativeDelta_ClampedToZero(t *testing.T) {
	// DurationCount stays the same or rises (no reset detected),
	// but individual counters decrease slightly — clamp defensively.
	b := zeroBuckets()

	samples := []models.TrafficSample{
		makeSample(0, 100, 5, 2, 1, 0, 1000, 2000, 10, 1.0, b, 3),
		// Req4xx drops 2→1 (DurationCount rose 10→11, so no reset).
		makeSample(30, 110, 6, 1, 2, 0, 1100, 2100, 11, 1.1, b, 4),
	}

	series, err := computeSeries(samples, "1h")
	require.NoError(t, err)
	require.Len(t, series.Points, 1)

	// Req4xx delta = -1 → clamped to 0.
	assert.Equal(t, int64(0), series.Points[0].Req4xx)
	// Other deltas are normal.
	assert.Equal(t, int64(10), series.Points[0].Req2xx)
	assert.Equal(t, int64(1), series.Points[0].Req5xx)
}

// ─── Histogram quantile: worked example ─────────────────────────────────────

// TestHistogramQuantile_WorkedExample validates the Prometheus linear-interpolation
// against the exact worked example stated in the task brief.
//
//	Buckets: {"0.005":0,"0.01":0,"0.025":1,"0.05":3,"0.1":7,"0.25":9,
//	           "0.5":10,"1":10,"2.5":10,"5":10,"10":10,"+Inf":10}
//
//	p50 → rank=5 → bucket 0.1 (cum=7, prev=0.05 cum=3)
//	     → 0.05 + (5-3)/(7-3)*(0.1-0.05) = 0.075s → 75ms
//	p95 → rank=9.5 → bucket 0.5 (cum=10, prev=0.25 cum=9)
//	     → 0.25 + (9.5-9)/(10-9)*(0.5-0.25) = 0.375s → 375ms
func TestHistogramQuantile_WorkedExample(t *testing.T) {
	bucketDeltaMap := map[string]int64{
		"0.005": 0,
		"0.01":  0,
		"0.025": 1,
		"0.05":  3,
		"0.1":   7,
		"0.25":  9,
		"0.5":   10,
		"1":     10,
		"2.5":   10,
		"5":     10,
		"10":    10,
		"+Inf":  10,
	}
	p50, p95 := computeQuantiles(bucketDeltaMap)
	assert.InDelta(t, 75.0, p50, 0.001, "p50 should be 75 ms")
	assert.InDelta(t, 375.0, p95, 0.001, "p95 should be 375 ms")
}

// ─── Zero-observation bucket ─────────────────────────────────────────────────

func TestComputeQuantiles_ZeroTotal(t *testing.T) {
	// +Inf count = 0 → no observations → p50=p95=0.
	b := map[string]int64{
		"0.05": 0,
		"0.1":  0,
		"+Inf": 0,
	}
	p50, p95 := computeQuantiles(b)
	assert.Equal(t, float64(0), p50)
	assert.Equal(t, float64(0), p95)
}

func TestComputeQuantiles_EmptyBuckets(t *testing.T) {
	p50, p95 := computeQuantiles(map[string]int64{})
	assert.Equal(t, float64(0), p50)
	assert.Equal(t, float64(0), p95)
}

// ─── End-to-end quantile via computeSeries ───────────────────────────────────

func TestComputeSeries_QuantileFromBucketDeltas(t *testing.T) {
	// Two consecutive samples; bucket deltas match the worked example above.
	b0 := map[string]int64{
		"0.005": 0, "0.01": 0, "0.025": 0,
		"0.05": 0, "0.1": 0, "0.25": 0,
		"0.5": 0, "1": 0, "2.5": 0,
		"5": 0, "10": 0, "+Inf": 0,
	}
	b1 := map[string]int64{
		"0.005": 0, "0.01": 0, "0.025": 1,
		"0.05": 3, "0.1": 7, "0.25": 9,
		"0.5": 10, "1": 10, "2.5": 10,
		"5": 10, "10": 10, "+Inf": 10,
	}

	samples := []models.TrafficSample{
		makeSample(0, 0, 0, 0, 0, 0, 0, 0, 0, 0.0, b0, 0),
		makeSample(30, 10, 0, 0, 0, 0, 0, 0, 10, 0.75, b1, 5),
	}

	series, err := computeSeries(samples, "1h")
	require.NoError(t, err)
	require.Len(t, series.Points, 1)
	pt := series.Points[0]

	assert.InDelta(t, 75.0, pt.P50Ms, 0.001, "p50 should be 75 ms")
	assert.InDelta(t, 375.0, pt.P95Ms, 0.001, "p95 should be 375 ms")
}

// ─── Range step metadata ─────────────────────────────────────────────────────

func TestComputeSeries_StepSeconds(t *testing.T) {
	cases := []struct {
		rng  string
		want int
	}{
		{"1h", 30},
		{"24h", 300},
		{"7d", 3600},
	}
	for _, tc := range cases {
		t.Run(tc.rng, func(t *testing.T) {
			series, err := computeSeries(nil, tc.rng)
			require.NoError(t, err)
			assert.Equal(t, tc.want, series.StepSeconds)
			assert.Equal(t, tc.rng, series.Range)
		})
	}
}

// ─── InFlight gauge ──────────────────────────────────────────────────────────

func TestComputeSeries_InFlight_LastWins(t *testing.T) {
	// Three samples in the same 30-s bucket; in_flight should be the last cur value.
	b := zeroBuckets()

	samples := []models.TrafficSample{
		makeSample(0, 0, 0, 0, 0, 0, 0, 0, 0, 0, b, 10),
		makeSample(10, 5, 0, 0, 0, 0, 0, 0, 5, 0.5, b, 7),
		makeSample(20, 10, 0, 0, 0, 0, 0, 0, 10, 1.0, b, 15),
	}

	series, err := computeSeries(samples, "1h")
	require.NoError(t, err)
	require.Len(t, series.Points, 1)
	assert.Equal(t, int64(15), series.Points[0].InFlight)
}

// ─── Fix 2: missing +Inf guard ───────────────────────────────────────────────

func TestComputeQuantiles_MissingPlusInf_ReturnsZero(t *testing.T) {
	// A DurationBuckets map with no "+Inf" key is a corrupt sample.
	// computeQuantiles must return (0, 0) instead of silently using the wrong bucket.
	buckets := map[string]int64{
		"0.05": 3,
		"0.1":  7,
		"0.25": 9,
		// "+Inf" intentionally absent
	}
	p50, p95 := computeQuantiles(buckets)
	assert.Equal(t, float64(0), p50, "p50 must be 0 when +Inf bucket is absent")
	assert.Equal(t, float64(0), p95, "p95 must be 0 when +Inf bucket is absent")
}

func TestComputeSeries_MissingPlusInfBucket_QuantilesZero(t *testing.T) {
	// End-to-end: a sample pair whose DurationBuckets lack "+Inf" should produce
	// p50=p95=0 in the resulting TrafficPoint.
	b0 := map[string]int64{"0.05": 0, "0.1": 0}
	b1 := map[string]int64{"0.05": 3, "0.1": 7}

	samples := []models.TrafficSample{
		makeSample(0, 0, 0, 0, 0, 0, 0, 0, 0, 0.0, b0, 0),
		makeSample(30, 10, 0, 0, 0, 0, 0, 0, 7, 0.5, b1, 2),
	}

	series, err := computeSeries(samples, "1h")
	require.NoError(t, err)
	require.Len(t, series.Points, 1)
	assert.Equal(t, float64(0), series.Points[0].P50Ms, "P50Ms must be 0 when +Inf absent")
	assert.Equal(t, float64(0), series.Points[0].P95Ms, "P95Ms must be 0 when +Inf absent")
}

// ─── GetTraffic integration (fake repo) ──────────────────────────────────────

// listFixtureRepo is a minimal fake repo whose ListSince returns a fixed slice.
// It is distinct from fakeTrafficSampleRepo (defined in metrics_scraper_service_test.go)
// because that one returns nil for ListSince.
type listFixtureRepo struct {
	samples []models.TrafficSample
	err     error
}

func (r *listFixtureRepo) Create(_ *models.TrafficSample) error { return nil }
func (r *listFixtureRepo) ListSince(_ time.Time) ([]models.TrafficSample, error) {
	return r.samples, r.err
}
func (r *listFixtureRepo) DeleteOlderThan(_ time.Time) (int64, error) { return 0, nil }

func TestGetTraffic_HappyPath_ReturnsSeries(t *testing.T) {
	// Two fixture samples 30 s apart in the 1h range → one output point.
	b0 := zeroBuckets()
	b1 := addBuckets(b0, map[string]int64{"0.05": 2, "+Inf": 5})

	fixtures := []models.TrafficSample{
		makeSample(0, 0, 0, 0, 0, 0, 0, 0, 0, 0.0, b0, 0),
		makeSample(30, 20, 1, 2, 0, 0, 500, 800, 5, 0.25, b1, 3),
	}

	svc := NewTrafficMetricsService(&listFixtureRepo{samples: fixtures}, nil)
	series, err := svc.GetTraffic("1h")

	require.NoError(t, err)
	require.NotNil(t, series)
	assert.Equal(t, "1h", series.Range)
	assert.Equal(t, 30, series.StepSeconds)
	require.NotEmpty(t, series.Points)
	assert.Equal(t, int64(20), series.Points[0].Req2xx)
}

func TestGetTraffic_InvalidRange_WrapsErrInvalidRange(t *testing.T) {
	svc := NewTrafficMetricsService(&listFixtureRepo{}, nil)
	_, err := svc.GetTraffic("bogus")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidRange), "error must wrap ErrInvalidRange")
}
