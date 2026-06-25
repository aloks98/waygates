package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// fakeTrafficSampleRepo is a hand-rolled fake that satisfies
// repository.TrafficSampleRepositoryInterface without any mocking framework.
type fakeTrafficSampleRepo struct {
	created      *models.TrafficSample
	deleteCutoff time.Time
}

func (f *fakeTrafficSampleRepo) Create(sample *models.TrafficSample) error {
	f.created = sample
	return nil
}

func (f *fakeTrafficSampleRepo) ListSince(_ time.Time) ([]models.TrafficSample, error) {
	return nil, nil
}

func (f *fakeTrafficSampleRepo) DeleteOlderThan(t time.Time) (int64, error) {
	f.deleteCutoff = t
	return 0, nil
}

// Compile-time assertion that the fake implements the interface.
var _ repository.TrafficSampleRepositoryInterface = (*fakeTrafficSampleRepo)(nil)

// minimalPrometheusFixture is a small valid Prometheus exposition body shaped
// like real Caddy output: the status-class counts come from the duration
// histogram's per-series _count (keyed by `code`), plus an in_flight gauge.
const minimalPrometheusFixture = `# HELP caddy_http_request_duration_seconds Histogram of round-trip request durations.
# TYPE caddy_http_request_duration_seconds histogram
caddy_http_request_duration_seconds_bucket{code="200",handler="static",method="GET",server="srv0",le="+Inf"} 42
caddy_http_request_duration_seconds_sum{code="200",handler="static",method="GET",server="srv0"} 4.2
caddy_http_request_duration_seconds_count{code="200",handler="static",method="GET",server="srv0"} 42
caddy_http_request_duration_seconds_bucket{code="404",handler="static",method="GET",server="srv0",le="+Inf"} 7
caddy_http_request_duration_seconds_sum{code="404",handler="static",method="GET",server="srv0"} 0.7
caddy_http_request_duration_seconds_count{code="404",handler="static",method="GET",server="srv0"} 7
# HELP caddy_http_requests_in_flight Current number of active connections.
# TYPE caddy_http_requests_in_flight gauge
caddy_http_requests_in_flight{handler="static",server="srv0"} 3
`

func TestMetricsScraperService_Scrape(t *testing.T) {
	// Stand up a fake Caddy admin /metrics endpoint.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(minimalPrometheusFixture))
	}))
	defer srv.Close()

	repo := &fakeTrafficSampleRepo{}

	scraper := NewMetricsScraperService(MetricsScraperConfig{
		Repo:     repo,
		AdminURL: srv.URL, // inject test server URL as admin base URL
	})

	before := time.Now()
	err := scraper.scrape()
	require.NoError(t, err, "scrape() should succeed against the test server")

	// --- Verify Create was called with expected aggregated values ---
	require.NotNil(t, repo.created, "repo.Create should have been called")

	sample := repo.created
	assert.Equal(t, int64(42), sample.Req2xx, "req_2xx: 200 responses")
	assert.Equal(t, int64(7), sample.Req4xx, "req_4xx: 404 responses")
	assert.Equal(t, int64(0), sample.Req3xx, "req_3xx: no 3xx in fixture")
	assert.Equal(t, int64(0), sample.Req5xx, "req_5xx: no 5xx in fixture")
	assert.Equal(t, int64(3), sample.InFlight, "in_flight: 3 connections")
	assert.WithinDuration(t, before, sample.CollectedAt, 5*time.Second, "CollectedAt should be close to now")

	// --- Verify DeleteOlderThan was called with a cutoff ~7 days in the past ---
	require.False(t, repo.deleteCutoff.IsZero(), "DeleteOlderThan should have been called")
	// The cutoff must be in the past by well over 6 days (retentionPeriod is 7 days).
	minExpectedAge := 6 * 24 * time.Hour
	age := time.Since(repo.deleteCutoff)
	assert.Greater(t, age, minExpectedAge,
		"cutoff should be at least 6 days in the past (retentionPeriod=7d), got age=%v", age)
}

func TestMetricsScraperService_ScrapeHTTPError(t *testing.T) {
	// Server returns a non-200 status.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	repo := &fakeTrafficSampleRepo{}
	scraper := NewMetricsScraperService(MetricsScraperConfig{
		Repo:     repo,
		AdminURL: srv.URL,
	})

	err := scraper.scrape()
	require.Error(t, err, "scrape() should fail on non-200 status")
	assert.Nil(t, repo.created, "repo.Create should NOT have been called on HTTP error")
}
