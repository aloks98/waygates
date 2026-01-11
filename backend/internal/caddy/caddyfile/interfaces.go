package caddyfile

import "github.com/aloks98/waygates/backend/internal/models"

// BuilderInterface defines the interface for Caddyfile generation
type BuilderInterface interface {
	BuildMainCaddyfile(opts MainCaddyfileOptions) string
	BuildProxyFile(proxy *models.Proxy) (string, error)
	BuildProxyFileWithACL(proxy *models.Proxy, aclAssignments []models.ProxyACLAssignment) (string, error)
	BuildCatchAllFile(settings *models.NotFoundSettings) string
	GetProxyFilename(proxy *models.Proxy) string
}

// ACLBuilderInterface defines the interface for ACL configuration generation
type ACLBuilderInterface interface {
	BuildACLConfig(proxy *models.Proxy, assignments []models.ProxyACLAssignment) string
}

// Ensure Builder implements BuilderInterface
var _ BuilderInterface = (*Builder)(nil)

// Ensure ACLBuilder implements ACLBuilderInterface
var _ ACLBuilderInterface = (*ACLBuilder)(nil)
