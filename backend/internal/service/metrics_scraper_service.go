package service

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/caddy"
	"github.com/aloks98/waygates/backend/internal/caddy/metrics"
	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

const (
	// retentionPeriod is the duration after which old traffic samples are deleted.
	retentionPeriod = 7 * 24 * time.Hour
)

// MetricsScraperService periodically scrapes Caddy's Prometheus metrics endpoint
// and stores one cumulative sample row per scrape in Postgres. Rows older than
// 7 days are pruned after every successful scrape.
type MetricsScraperService struct {
	repo     repository.TrafficSampleRepositoryInterface
	logger   *zap.Logger
	adminURL string
	client   *http.Client

	ticker   *time.Ticker
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// MetricsScraperConfig holds configuration for MetricsScraperService.
type MetricsScraperConfig struct {
	Repo   repository.TrafficSampleRepositoryInterface
	Logger *zap.Logger
	// AdminURL is the base Caddy admin API URL. Defaults to caddy.DefaultAdminAPIURL.
	AdminURL string
}

// NewMetricsScraperService constructs a MetricsScraperService.
func NewMetricsScraperService(cfg MetricsScraperConfig) *MetricsScraperService {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	if cfg.AdminURL == "" {
		cfg.AdminURL = caddy.DefaultAdminAPIURL
	}
	return &MetricsScraperService{
		repo:     cfg.Repo,
		logger:   cfg.Logger.Named("metrics-scraper"),
		adminURL: cfg.AdminURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		stopChan: make(chan struct{}),
	}
}

// Start launches the background scrape loop at the given interval.
func (s *MetricsScraperService) Start(interval time.Duration) {
	s.logger.Info("Starting metrics scraper", zap.Duration("interval", interval))
	s.ticker = time.NewTicker(interval)
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.ticker.C:
				if err := s.scrape(); err != nil {
					s.logger.Warn("Metrics scrape failed", zap.Error(err))
				}
			case <-s.stopChan:
				s.logger.Info("Metrics scraper stopping")
				return
			}
		}
	}()
}

// Stop signals the background goroutine to exit and waits for it to finish.
func (s *MetricsScraperService) Stop() {
	close(s.stopChan)
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.wg.Wait()
	s.logger.Info("Metrics scraper stopped")
}

// scrape performs one fetch-parse-store cycle.
func (s *MetricsScraperService) scrape() error {
	url := s.adminURL + "/metrics"
	resp, err := s.client.Get(url) //nolint:noctx
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: unexpected status %d", url, resp.StatusCode)
	}

	snap, err := metrics.ParseSnapshot(resp.Body)
	if err != nil {
		return fmt.Errorf("parse metrics: %w", err)
	}

	now := time.Now()
	sample := &models.TrafficSample{
		CollectedAt:     now,
		Req2xx:          snap.Req2xx,
		Req3xx:          snap.Req3xx,
		Req4xx:          snap.Req4xx,
		Req5xx:          snap.Req5xx,
		ReqOther:        snap.ReqOther,
		BytesIn:         snap.BytesIn,
		BytesOut:        snap.BytesOut,
		DurationBuckets: models.DurationBucketsField(snap.DurationBuckets),
		DurationSum:     snap.DurationSum,
		DurationCount:   snap.DurationCount,
		InFlight:        snap.InFlight,
	}

	if err := s.repo.Create(sample); err != nil {
		return fmt.Errorf("store traffic sample: %w", err)
	}

	cutoff := now.Add(-retentionPeriod)
	deleted, err := s.repo.DeleteOlderThan(cutoff)
	if err != nil {
		s.logger.Warn("Failed to prune old traffic samples", zap.Error(err))
	} else if deleted > 0 {
		s.logger.Debug("Pruned old traffic samples", zap.Int64("count", deleted))
	}

	s.logger.Debug("Traffic sample stored",
		zap.Time("collected_at", now),
		zap.Int64("req_2xx", snap.Req2xx),
		zap.Int64("in_flight", snap.InFlight),
	)
	return nil
}
