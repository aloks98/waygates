package handlers

import (
	"errors"
	"net/http"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// MetricsHandler handles traffic metrics read requests.
type MetricsHandler struct {
	svc    *service.TrafficMetricsService
	logger *zap.Logger
}

// NewMetricsHandler creates a new MetricsHandler.
func NewMetricsHandler(svc *service.TrafficMetricsService, logger *zap.Logger) *MetricsHandler {
	return &MetricsHandler{
		svc:    svc,
		logger: logger,
	}
}

// GetTraffic returns the traffic time-series for the requested range.
//
// Query parameters:
//   - range: "1h" (default), "24h", or "7d"
func (h *MetricsHandler) GetTraffic(w http.ResponseWriter, r *http.Request) {
	rng := r.URL.Query().Get("range")
	if rng == "" {
		rng = "1h"
	}

	series, err := h.svc.GetTraffic(rng)
	if err != nil {
		if errors.Is(err, service.ErrInvalidRange) {
			utils.BadRequest(w, err.Error(), nil)
			return
		}
		h.logger.Error("get traffic metrics failed", zap.Error(err))
		utils.InternalError(w, "failed to load traffic metrics")
		return
	}

	utils.Success(w, series, "Traffic metrics retrieved successfully")
}
