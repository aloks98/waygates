package models

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/schema"
)

// TestTrafficSampleColumnNames guards against GORM naming drift between the
// TrafficSample model and the 000012_create_traffic_samples migration.
//
// GORM's default snake_case conversion does NOT insert an underscore before a
// digit, so Req2xx maps to "req2xx" unless an explicit column tag is set — but
// the migration defines "req_2xx". Without the column tags this test fails, and
// in production every INSERT errors with `column "req2xx" ... does not exist`.
func TestTrafficSampleColumnNames(t *testing.T) {
	s, err := schema.Parse(&TrafficSample{}, &sync.Map{}, schema.NamingStrategy{})
	require.NoError(t, err)

	// Field name -> column name as defined in the SQL migration.
	want := map[string]string{
		"ID":              "id",
		"CollectedAt":     "collected_at",
		"Req2xx":          "req_2xx",
		"Req3xx":          "req_3xx",
		"Req4xx":          "req_4xx",
		"Req5xx":          "req_5xx",
		"ReqOther":        "req_other",
		"BytesIn":         "bytes_in",
		"BytesOut":        "bytes_out",
		"DurationBuckets": "duration_buckets",
		"DurationSum":     "duration_sum",
		"DurationCount":   "duration_count",
		"InFlight":        "in_flight",
	}

	for field, col := range want {
		f := s.LookUpField(field)
		require.NotNilf(t, f, "field %s not found in TrafficSample schema", field)
		assert.Equalf(t, col, f.DBName, "field %s maps to the wrong column", field)
	}
}
