-- Create traffic_samples table for storing scraped Caddy metrics
CREATE TABLE IF NOT EXISTS traffic_samples (
    id                SERIAL PRIMARY KEY,
    collected_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- HTTP request counters bucketed by response-code class
    req_2xx           BIGINT NOT NULL DEFAULT 0,
    req_3xx           BIGINT NOT NULL DEFAULT 0,
    req_4xx           BIGINT NOT NULL DEFAULT 0,
    req_5xx           BIGINT NOT NULL DEFAULT 0,
    req_other         BIGINT NOT NULL DEFAULT 0,

    -- Bytes transferred
    bytes_in          BIGINT NOT NULL DEFAULT 0,
    bytes_out         BIGINT NOT NULL DEFAULT 0,

    -- Request duration histogram
    duration_buckets  TEXT NOT NULL DEFAULT '{}',
    duration_sum      DOUBLE PRECISION NOT NULL DEFAULT 0,
    duration_count    BIGINT NOT NULL DEFAULT 0,

    -- In-flight requests gauge
    in_flight         BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX idx_traffic_samples_collected_at ON traffic_samples(collected_at);
