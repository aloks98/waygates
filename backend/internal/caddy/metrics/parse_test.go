package metrics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixture contains multiple series per metric (different handler/method/code
// label combinations) to verify that summation and code-class bucketing work.
const fixture = `# HELP caddy_http_requests_total Counter of HTTP(S) requests made.
# TYPE caddy_http_requests_total counter
caddy_http_requests_total{code="200",handler="static",method="GET",server="srv0"} 100
caddy_http_requests_total{code="201",handler="api",method="POST",server="srv0"} 50
caddy_http_requests_total{code="301",handler="redirect",method="GET",server="srv0"} 10
caddy_http_requests_total{code="302",handler="redirect",method="GET",server="srv0"} 5
caddy_http_requests_total{code="404",handler="static",method="GET",server="srv0"} 20
caddy_http_requests_total{code="429",handler="api",method="POST",server="srv0"} 3
caddy_http_requests_total{code="500",handler="api",method="GET",server="srv0"} 7
caddy_http_requests_total{code="503",handler="upstream",method="GET",server="srv0"} 2
caddy_http_requests_total{code="unknown",handler="api",method="GET",server="srv0"} 1
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
caddy_http_request_duration_seconds_bucket{handler="static",server="srv0",le="0.005"} 50
caddy_http_request_duration_seconds_bucket{handler="static",server="srv0",le="0.01"} 70
caddy_http_request_duration_seconds_bucket{handler="static",server="srv0",le="+Inf"} 90
caddy_http_request_duration_seconds_sum{handler="static",server="srv0"} 4.5
caddy_http_request_duration_seconds_count{handler="static",server="srv0"} 90
caddy_http_request_duration_seconds_bucket{handler="api",server="srv0",le="0.005"} 10
caddy_http_request_duration_seconds_bucket{handler="api",server="srv0",le="0.01"} 20
caddy_http_request_duration_seconds_bucket{handler="api",server="srv0",le="+Inf"} 60
caddy_http_request_duration_seconds_sum{handler="api",server="srv0"} 12.0
caddy_http_request_duration_seconds_count{handler="api",server="srv0"} 60
# HELP caddy_http_requests_in_flight Current number of active connections.
# TYPE caddy_http_requests_in_flight gauge
caddy_http_requests_in_flight{handler="static",server="srv0"} 3
caddy_http_requests_in_flight{handler="api",server="srv0"} 5
`

func TestParseSnapshot_Aggregation(t *testing.T) {
	snap, err := ParseSnapshot(strings.NewReader(fixture))
	require.NoError(t, err)
	require.NotNil(t, snap)

	// expected 2xx total 150 from 100 (code 200) plus 50 (code 201)
	assert.Equal(t, int64(150), snap.Req2xx, "req_2xx")
	// expected 3xx total 15 from 10 (code 301) plus 5 (code 302)
	assert.Equal(t, int64(15), snap.Req3xx, "req_3xx")
	// expected 4xx total 23 from 20 (code 404) plus 3 (code 429)
	assert.Equal(t, int64(23), snap.Req4xx, "req_4xx")
	// expected 5xx total 9 from 7 (code 500) plus 2 (code 503)
	assert.Equal(t, int64(9), snap.Req5xx, "req_5xx")
	// req_other: 1 ("unknown")
	assert.Equal(t, int64(1), snap.ReqOther, "req_other")

	// bytes_in: histogram _sum across both handlers: 45000 + 30000 = 75000
	assert.Equal(t, int64(75000), snap.BytesIn, "bytes_in")
	// expected bytes_out total 150000 from 90000 plus 60000
	assert.Equal(t, int64(150000), snap.BytesOut, "bytes_out")

	// expected duration_sum 16.5 from 4.5 plus 12.0
	assert.InDelta(t, 16.5, snap.DurationSum, 1e-9, "duration_sum")
	// expected duration_count 150 from 90 plus 60
	assert.Equal(t, int64(150), snap.DurationCount, "duration_count")

	// duration_buckets: summed per le
	assert.Equal(t, int64(50+10), snap.DurationBuckets["0.005"], "bucket le=0.005")
	assert.Equal(t, int64(70+20), snap.DurationBuckets["0.01"], "bucket le=0.01")
	assert.Equal(t, int64(90+60), snap.DurationBuckets["+Inf"], "bucket le=+Inf")

	// expected in_flight 8 from 3 plus 5
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
	// Verify boundary cases: unknown code, empty code handling
	input := `# HELP caddy_http_requests_total Counter
# TYPE caddy_http_requests_total counter
caddy_http_requests_total{code="200",server="srv0"} 10
caddy_http_requests_total{code="301",server="srv0"} 5
caddy_http_requests_total{code="418",server="srv0"} 2
caddy_http_requests_total{code="502",server="srv0"} 1
caddy_http_requests_total{code="999",server="srv0"} 3
`
	snap, err := ParseSnapshot(strings.NewReader(input))
	require.NoError(t, err)

	assert.Equal(t, int64(10), snap.Req2xx)
	assert.Equal(t, int64(5), snap.Req3xx)
	assert.Equal(t, int64(2), snap.Req4xx)
	assert.Equal(t, int64(1), snap.Req5xx)
	assert.Equal(t, int64(3), snap.ReqOther, "9xx should fall into req_other")
}
