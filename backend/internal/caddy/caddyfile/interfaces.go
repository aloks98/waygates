package caddyfile

import "github.com/aloks98/waygates/backend/internal/models"

// BuilderInterface defines the interface for Caddyfile generation
type BuilderInterface interface {
	BuildMainCaddyfile(opts MainCaddyfileOptions) string
	BuildProxyFile(proxy *models.Proxy) (string, error)
	BuildCatchAllFile(settings *models.NotFoundSettings) string
	GetProxyFilename(proxy *models.Proxy) string
}

// Ensure Builder implements BuilderInterface
var _ BuilderInterface = (*Builder)(nil)
