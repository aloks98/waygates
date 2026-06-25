package metrics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixture mirrors the real Caddy /metrics exposition: caddy_http_requests_total
// carries NO `code` label (only handler/server), so the status-class breakdown
// must come from the duration histogram's per-series _count keyed by `code`.
// The requests_total value below is deliberately large and unrelated so a
// regression that buckets it would produce visibly wrong numbers.
const fixture = `# HELP caddy_http_requests_total Counter of HTTP(S) requests made.
# TYPE caddy_http_requests_total counter
caddy_http_requests_total{handler="reverse_proxy",server="srv0"} 9999
# HELP caddy_http_request_size_bytes Histogram of the total number of bytes received from clients.
# TYPE caddy_http_request_size_bytes histogram
caddy_http_request_size_bytes_bucket{handler="static",server="srv0",le="1000"} 80
caddy_http_request_size_bytes_bucket{handler="static",server="srv0",le="+Inf"} 90
caddy_http_request_size_bytes_sum{handler="static",server="srv0"} 45000
caddy_http_request_size_bytes_count{handler="static",server="srv0"} 90
caddy_http_request_size_bytes_bucket{handler="api",server="srv0",le="1000"} 40
caddy_http_request_size_bytes_bucket{handler="api",server="srv0",le="+Inf"} 60
caddy_http_request_size_bytes_sum{handler="api",server="srv0"} 30000
caddy_http_request_size_bytes_count{handler="api",server="srv0"} 60
# HELP caddy_http_response_size_bytes Histogram of the number of bytes sent to clients.
# TYPE caddy_http_response_size_bytes histogram
caddy_http_response_size_bytes_bucket{handler="static",server="srv0",le="1000"} 70
caddy_http_response_size_bytes_bucket{handler="static",server="srv0",le="+Inf"} 90
caddy_http_response_size_bytes_sum{handler="static",server="srv0"} 90000
caddy_http_response_size_bytes_count{handler="static",server="srv0"} 90
caddy_http_response_size_bytes_bucket{handler="api",server="srv0",le="1000"} 30
caddy_http_response_size_bytes_bucket{handler="api",server="srv0",le="+Inf"} 60
caddy_http_response_size_bytes_sum{handler="api",server="srv0"} 60000
caddy_http_response_size_bytes_count{handler="api",server="srv0"} 60
# HELP caddy_http_request_duration_seconds Histogram of round-trip request durations.
# TYPE caddy_http_request_duration_seconds histogram
caddy_http_request_duration_seconds_bucket{code="200",handler="rp",method="GET",server="srv0",le="0.005"} 50
caddy_http_request_duration_seconds_bucket{code="200",handler="rp",method="GET",server="srv0",le="0.01"} 70
caddy_http_request_duration_seconds_bucket{code="200",handler="rp",method="GET",server="srv0",le="+Inf"} 100
caddy_http_request_duration_seconds_sum{code="200",handler="rp",method="GET",server="srv0"} 5.0
caddy_http_request_duration_seconds_count{code="200",handler="rp",method="GET",server="srv0"} 100
caddy_http_request_duration_seconds_bucket{code="204",handler="rp",method="POST",server="srv0",le="0.005"} 10
caddy_http_request_duration_seconds_bucket{code="204",handler="rp",method="POST",server="srv0",le="0.01"} 20
caddy_http_request_duration_seconds_bucket{code="204",handler="rp",method="POST",server="srv0",le="+Inf"} 50
caddy_http_request_duration_seconds_sum{code="204",handler="rp",method="POST",server="srv0"} 3.0
caddy_http_request_duration_seconds_count{code="204",handler="rp",method="POST",server="srv0"} 50
caddy_http_request_duration_seconds_bucket{code="304",handler="rp",method="GET",server="srv0",le="0.005"} 5
caddy_http_request_duration_seconds_bucket{code="304",handler="rp",method="GET",server="srv0",le="0.01"} 10
caddy_http_request_duration_seconds_bucket{code="304",handler="rp",method="GET",server="srv0",le="+Inf"} 15
caddy_http_request_duration_seconds_sum{code="304",handler="rp",method="GET",server="srv0"} 0.5
caddy_http_request_duration_seconds_count{code="304",handler="rp",method="GET",server="srv0"} 15
caddy_http_request_duration_seconds_bucket{code="404",handler="rp",method="GET",server="srv0",le="0.005"} 10
caddy_http_request_duration_seconds_bucket{code="404",handler="rp",method="GET",server="srv0",le="0.01"} 15
caddy_http_request_duration_seconds_bucket{code="404",handler="rp",method="GET",server="srv0",le="+Inf"} 23
caddy_http_request_duration_seconds_sum{code="404",handler="rp",method="GET",server="srv0"} 1.0
caddy_http_request_duration_seconds_count{code="404",handler="rp",method="GET",server="srv0"} 23
caddy_http_request_duration_seconds_bucket{code="500",handler="rp",method="GET",server="srv0",le="0.005"} 3
caddy_http_request_duration_seconds_bucket{code="500",handler="rp",method="GET",server="srv0",le="0.01"} 5
caddy_http_request_duration_seconds_bucket{code="500",handler="rp",method="GET",server="srv0",le="+Inf"} 9
caddy_http_request_duration_seconds_sum{code="500",handler="rp",method="GET",server="srv0"} 0.3
caddy_http_request_duration_seconds_count{code="500",handler="rp",method="GET",server="srv0"} 9
caddy_http_request_duration_seconds_bucket{code="unknown",handler="rp",method="GET",server="srv0",le="0.005"} 0
caddy_http_request_duration_seconds_bucket{code="unknown",handler="rp",method="GET",server="srv0",le="0.01"} 1
caddy_http_request_duration_seconds_bucket{code="unknown",handler="rp",method="GET",server="srv0",le="+Inf"} 1
caddy_http_request_duration_seconds_sum{code="unknown",handler="rp",method="GET",server="srv0"} 0.01
caddy_http_request_duration_seconds_count{code="unknown",handler="rp",method="GET",server="srv0"} 1
# HELP caddy_http_requests_in_flight Current number of active connections.
# TYPE caddy_http_requests_in_flight gauge
caddy_http_requests_in_flight{handler="static",server="srv0"} 3
caddy_http_requests_in_flight{handler="api",server="srv0"} 5
`

func TestParseSnapshot_Aggregation(t *testing.T) {
	snap, err := ParseSnapshot(strings.NewReader(fixture))
	require.NoError(t, err)
	require.NotNil(t, snap)

	// Request classes come from the duration histogram _count by `code`, NOT
	// from caddy_http_requests_total (9999, no code — must be ignored).
	assert.Equal(t, int64(150), snap.Req2xx, "req_2xx") // 100 (200) + 50 (204)
	assert.Equal(t, int64(15), snap.Req3xx, "req_3xx")  // 304
	assert.Equal(t, int64(23), snap.Req4xx, "req_4xx")  // 404
	assert.Equal(t, int64(9), snap.Req5xx, "req_5xx")   // 500
	assert.Equal(t, int64(1), snap.ReqOther, "req_other")

	// bytes_in: histogram _sum summed across both request-size handlers
	assert.Equal(t, int64(75000), snap.BytesIn, "bytes_in")
	// bytes_out: histogram _sum summed across both response-size handlers
	assert.Equal(t, int64(150000), snap.BytesOut, "bytes_out")

	// duration_sum: sum of the per-code _sum values
	assert.InDelta(t, 9.81, snap.DurationSum, 1e-9, "duration_sum")
	// duration_count: total requests across all codes
	assert.Equal(t, int64(198), snap.DurationCount, "duration_count")

	// duration_buckets: summed per le across all code series
	assert.Equal(t, int64(50+10+5+10+3+0), snap.DurationBuckets["0.005"], "bucket le=0.005")
	assert.Equal(t, int64(70+20+10+15+5+1), snap.DurationBuckets["0.01"], "bucket le=0.01")
	assert.Equal(t, int64(198), snap.DurationBuckets["+Inf"], "bucket le=+Inf")

	// in_flight: sum of both handler gauges
	assert.Equal(t, int64(8), snap.InFlight, "in_flight")
}

func TestParseSnapshot_EmptyBody(t *testing.T) {
	snap, err := ParseSnapshot(strings.NewReader(""))
	require.NoError(t, err)
	require.NotNil(t, snap)

	assert.Equal(t, int64(0), snap.Req2xx)
	assert.Equal(t, int64(0), snap.InFlight)
	assert.Empty(t, snap.DurationBuckets)
}

func TestParseSnapshot_CodeBucketing(t *testing.T) {
	// Boundary cases driven by the duration histogram _count: a teapot (418 ->
	// 4xx), a 502 (-> 5xx), and a non-standard 9xx code (-> req_other).
	input := `# HELP caddy_http_request_duration_seconds Histogram
# TYPE caddy_http_request_duration_seconds histogram
caddy_http_request_duration_seconds_bucket{code="200",server="srv0",le="+Inf"} 10
caddy_http_request_duration_seconds_sum{code="200",server="srv0"} 1.0
caddy_http_request_duration_seconds_count{code="200",server="srv0"} 10
caddy_http_request_duration_seconds_bucket{code="301",server="srv0",le="+Inf"} 5
caddy_http_request_duration_seconds_sum{code="301",server="srv0"} 0.5
caddy_http_request_duration_seconds_count{code="301",server="srv0"} 5
caddy_http_request_duration_seconds_bucket{code="418",server="srv0",le="+Inf"} 2
caddy_http_request_duration_seconds_sum{code="418",server="srv0"} 0.2
caddy_http_request_duration_seconds_count{code="418",server="srv0"} 2
caddy_http_request_duration_seconds_bucket{code="502",server="srv0",le="+Inf"} 1
caddy_http_request_duration_seconds_sum{code="502",server="srv0"} 0.1
caddy_http_request_duration_seconds_count{code="502",server="srv0"} 1
caddy_http_request_duration_seconds_bucket{code="999",server="srv0",le="+Inf"} 3
caddy_http_request_duration_seconds_sum{code="999",server="srv0"} 0.3
caddy_http_request_duration_seconds_count{code="999",server="srv0"} 3
`
	snap, err := ParseSnapshot(strings.NewReader(input))
	require.NoError(t, err)

	assert.Equal(t, int64(10), snap.Req2xx)
	assert.Equal(t, int64(5), snap.Req3xx)
	assert.Equal(t, int64(2), snap.Req4xx)
	assert.Equal(t, int64(1), snap.Req5xx)
	assert.Equal(t, int64(3), snap.ReqOther, "9xx should fall into req_other")
}

// TestParseSnapshot_RequestsTotalIgnored locks in that caddy_http_requests_total
// (which has no `code` label in real Caddy) does not contribute to any
// request-class counter — the regression that put every request into "other".
func TestParseSnapshot_RequestsTotalIgnored(t *testing.T) {
	input := `# HELP caddy_http_requests_total Counter
# TYPE caddy_http_requests_total counter
caddy_http_requests_total{handler="reverse_proxy",server="srv0"} 500
`
	snap, err := ParseSnapshot(strings.NewReader(input))
	require.NoError(t, err)

	assert.Equal(t, int64(0), snap.Req2xx)
	assert.Equal(t, int64(0), snap.ReqOther, "requests_total must not be bucketed")
	assert.Equal(t, int64(0), snap.DurationCount)
}
