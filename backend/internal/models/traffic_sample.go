package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// DurationBucketsField is a custom type for storing a map[string]int64 as JSON
// text in Postgres. It mirrors the Valuer/Scanner pattern of JSONField but is
// typed to map[string]int64 for the histogram buckets column.
type DurationBucketsField map[string]int64

// Value implements the driver.Valuer interface for database storage.
func (d DurationBucketsField) Value() (driver.Value, error) {
	if d == nil {
		return "{}", nil
	}
	b, err := json.Marshal(map[string]int64(d))
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan implements the sql.Scanner interface for database retrieval.
func (d *DurationBucketsField) Scan(value interface{}) error {
	if value == nil {
		*d = make(DurationBucketsField)
		return nil
	}

	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("DurationBucketsField: unsupported type %T", value)
	}

	if len(raw) == 0 {
		*d = make(DurationBucketsField)
		return nil
	}

	result := make(map[string]int64)
	if err := json.Unmarshal(raw, &result); err != nil {
		return err
	}
	*d = result
	return nil
}

// TrafficSample represents one cumulative scrape from the Caddy metrics endpoint.
// One row is inserted every scrape interval (default 30 s).
type TrafficSample struct {
	ID          int       `json:"id"           gorm:"primaryKey;autoIncrement"`
	CollectedAt time.Time `json:"collected_at" gorm:"not null;index"`
	Req2xx      int64     `json:"req_2xx"      gorm:"not null;default:0"`
	Req3xx      int64     `json:"req_3xx"      gorm:"not null;default:0"`
	Req4xx      int64     `json:"req_4xx"      gorm:"not null;default:0"`
	Req5xx      int64     `json:"req_5xx"      gorm:"not null;default:0"`
	ReqOther    int64     `json:"req_other"    gorm:"not null;default:0"`
	BytesIn     int64     `json:"bytes_in"     gorm:"not null;default:0"`
	BytesOut    int64     `json:"bytes_out"    gorm:"not null;default:0"`
	// DurationBuckets is a JSON map of le-label → cumulative count.
	DurationBuckets DurationBucketsField `json:"duration_buckets" gorm:"type:text;not null;default:'{}'"`
	DurationSum     float64              `json:"duration_sum"  gorm:"not null;default:0"`
	DurationCount   int64                `json:"duration_count" gorm:"not null;default:0"`
	InFlight        int64                `json:"in_flight"    gorm:"not null;default:0"`
}

// TableName specifies the table name for GORM.
func (TrafficSample) TableName() string {
	return "traffic_samples"
}
