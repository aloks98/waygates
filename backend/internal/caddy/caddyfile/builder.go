package caddyfile

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/models"
)

// Builder generates Caddyfile content for different proxy types
type Builder struct {
	logger     *zap.Logger
	aclBuilder *ACLBuilder
}

// BuilderOptions holds configuration options for creating a new Builder
type BuilderOptions struct {
	Logger            *zap.Logger
	WaygatesVerifyURL string // URL for Waygates forward auth verification
}

// NewBuilder creates a new Caddyfile builder
func NewBuilder(logger *zap.Logger) *Builder {
	return &Builder{
		logger:     logger,
		aclBuilder: nil, // No ACL support by default for backward compatibility
	}
}

// NewBuilderWithOptions creates a new Caddyfile builder with full options
func NewBuilderWithOptions(opts BuilderOptions) *Builder {
	var aclBuilder *ACLBuilder
	if opts.WaygatesVerifyURL != "" {
		aclBuilder = NewACLBuilder(opts.WaygatesVerifyURL)
	}

	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Builder{
		logger:     logger,
		aclBuilder: aclBuilder,
	}
}

// MainCaddyfileOptions holds options for building the main Caddyfile
type MainCaddyfileOptions struct {
	Email        string // Email for ACME certificate notifications
	ACMEProvider string // DNS provider: off, http, cloudflare, route53, duckdns, digitalocean, hetzner, porkbun, azure, vultr, namecheap, ovh
}

// BuildMainCaddyfile generates a Caddyfile with global options based on ACME provider.
// The ACMEProvider option controls TLS certificate issuance:
//   - "off": Disable automatic HTTPS (default for development)
//   - "http": Use HTTP challenge (requires ports 80/443 open to internet)
//   - DNS providers: Use DNS challenge with the specified provider
func (b *Builder) BuildMainCaddyfile(opts MainCaddyfileOptions) string {
	var sb strings.Builder

	sb.WriteString("# Managed by Waygates - DO NOT EDIT MANUALLY\n")
	sb.WriteString(fmt.Sprintf("# ACME Provider: %s\n", opts.ACMEProvider))
	sb.WriteString(fmt.Sprintf("# Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	// Global options block
	sb.WriteString("{\n")

	// Persist certificates and ACME account in /data (Docker volume)
	sb.WriteString("\tstorage file_system /data\n")

	if opts.Email != "" {
		sb.WriteString(fmt.Sprintf("\temail %s\n", opts.Email))
	}

	// Configure ACME based on provider
	switch opts.ACMEProvider {
	case "off", "":
		sb.WriteString("\tauto_https off\n")
	case "http":
		// HTTP challenge - no additional config needed, Caddy handles it automatically
	default:
		// DNS challenge - add the provider-specific acme_dns directive
		acmeConfig := buildACMEDNSConfig(opts.ACMEProvider)
		if acmeConfig != "" {
			sb.WriteString(acmeConfig)
		}
	}

	sb.WriteString("\tadmin localhost:2019\n")
	sb.WriteString("}\n\n")

	// Import proxy configs
	sb.WriteString("import sites/*.conf\n\n")

	// Import catch-all (must be last)
	sb.WriteString("import catchall.conf\n")

	return sb.String()
}

// buildACMEDNSConfig returns the acme_dns directive configuration for the given provider.
// Each DNS provider has a specific configuration format as per caddy-dns plugin documentation.
// Uses {$VAR} syntax for parse-time environment variable substitution (more reliable than {env.VAR}).
func buildACMEDNSConfig(provider string) string {
	switch provider {
	case "cloudflare":
		return "\tacme_dns cloudflare {$CLOUDFLARE_API_TOKEN}\n"
	case "route53":
		// Route53 uses AWS SDK which reads credentials from environment automatically
		return "\tacme_dns route53\n"
	case "duckdns":
		return "\tacme_dns duckdns {$DUCKDNS_API_TOKEN}\n"
	case "digitalocean":
		return "\tacme_dns digitalocean {$DO_AUTH_TOKEN}\n"
	case "hetzner":
		return "\tacme_dns hetzner {$HETZNER_API_TOKEN}\n"
	case "porkbun":
		return "\tacme_dns porkbun {\n\t\tapi_key {$PORKBUN_API_KEY}\n\t\tapi_secret_key {$PORKBUN_API_SECRET_KEY}\n\t}\n"
	case "azure":
		return "\tacme_dns azure {\n\t\ttenant_id {$AZURE_TENANT_ID}\n\t\tclient_id {$AZURE_CLIENT_ID}\n\t\tclient_secret {$AZURE_CLIENT_SECRET}\n\t\tsubscription_id {$AZURE_SUBSCRIPTION_ID}\n\t\tresource_group_name {$AZURE_RESOURCE_GROUP}\n\t}\n"
	case "vultr":
		return "\tacme_dns vultr {$VULTR_API_KEY}\n"
	case "namecheap":
		return "\tacme_dns namecheap {\n\t\tapi_key {$NAMECHEAP_API_KEY}\n\t\tuser {$NAMECHEAP_API_USER}\n\t}\n"
	case "ovh":
		return "\tacme_dns ovh {\n\t\tendpoint {$OVH_ENDPOINT}\n\t\tapplication_key {$OVH_APPLICATION_KEY}\n\t\tapplication_secret {$OVH_APPLICATION_SECRET}\n\t\tconsumer_key {$OVH_CONSUMER_KEY}\n\t}\n"
	default:
		return ""
	}
}

// BuildProxyFile generates config content for a single proxy without ACL support.
// For ACL-enabled proxies, use BuildProxyFileWithACL instead.
func (b *Builder) BuildProxyFile(proxy *models.Proxy) (string, error) {
	return b.BuildProxyFileWithACL(proxy, nil)
}

// BuildProxyFileWithACL generates config content for a single proxy with optional ACL support.
// If aclAssignments is nil or empty, it generates standard proxy config without ACL.
func (b *Builder) BuildProxyFileWithACL(proxy *models.Proxy, aclAssignments []models.ProxyACLAssignment) (string, error) {
	if proxy == nil {
		return "", fmt.Errorf("proxy is nil")
	}

	var content string
	var err error

	// Check if we have ACL assignments and ACL builder is configured
	hasAssignments := len(aclAssignments) > 0
	hasBuilder := b.aclBuilder != nil
	hasConfig := HasACLConfig(aclAssignments)
	hasACL := hasAssignments && hasBuilder && hasConfig

	b.logger.Debug("Building proxy config with ACL check",
		zap.Int("proxy_id", proxy.ID),
		zap.String("proxy_name", proxy.Name),
		zap.Int("acl_assignments_count", len(aclAssignments)),
		zap.Bool("has_assignments", hasAssignments),
		zap.Bool("has_acl_builder", hasBuilder),
		zap.Bool("has_acl_config", hasConfig),
		zap.Bool("has_acl", hasACL),
	)

	switch proxy.Type {
	case models.ProxyTypeReverseProxy:
		if hasACL {
			content, err = b.buildReverseProxyBlockWithACL(proxy, aclAssignments)
		} else {
			content, err = b.buildReverseProxyBlock(proxy)
		}
	case models.ProxyTypeStatic:
		// Static proxies currently don't support ACL
		content, err = b.buildStaticBlock(proxy)
	case models.ProxyTypeRedirect:
		// Redirect proxies currently don't support ACL
		content, err = b.buildRedirectBlock(proxy)
	default:
		return "", fmt.Errorf("unknown proxy type: %s", proxy.Type)
	}

	if err != nil {
		return "", err
	}

	// Add header comment
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Proxy ID: %d\n", proxy.ID))
	sb.WriteString(fmt.Sprintf("# Name: %s\n", proxy.Name))
	sb.WriteString(fmt.Sprintf("# Type: %s\n", proxy.Type))
	if hasACL {
		sb.WriteString(fmt.Sprintf("# ACL Enabled: true (%d assignments)\n", len(aclAssignments)))
	}
	sb.WriteString(fmt.Sprintf("# Updated: %s\n\n", time.Now().Format(time.RFC3339)))
	sb.WriteString(content)

	return sb.String(), nil
}

// BuildCatchAllFile generates the catch-all 404 config
func (b *Builder) BuildCatchAllFile(settings *models.NotFoundSettings) string {
	var sb strings.Builder

	sb.WriteString("# Catch-all 404 handler\n")
	sb.WriteString(fmt.Sprintf("# Mode: %s\n", settings.Mode))
	sb.WriteString(fmt.Sprintf("# Updated: %s\n\n", time.Now().Format(time.RFC3339)))

	// Catch-all on port 80 only - specific domains handle their own HTTPS
	sb.WriteString(":80 {\n")

	if settings.Mode == "redirect" && settings.RedirectURL != "" {
		sb.WriteString(fmt.Sprintf("\tredir %s 302\n", settings.RedirectURL))
	} else {
		// Default mode: respond with 404
		sb.WriteString("\trespond \"Not Found\" 404\n")
	}

	sb.WriteString("}\n")

	return sb.String()
}

// GetProxyFilename returns the filename for a proxy
// Format: {id}_{sanitized_hostname}.conf
func (b *Builder) GetProxyFilename(proxy *models.Proxy) string {
	sanitized := sanitizeFilename(proxy.Hostname)
	return fmt.Sprintf("%d_%s.conf", proxy.ID, sanitized)
}

// GetDisabledFilename returns the disabled filename for a proxy
func (b *Builder) GetDisabledFilename(proxy *models.Proxy) string {
	return b.GetProxyFilename(proxy) + ".disabled"
}

// formatSiteAddress returns the site address for Caddyfile
// If SSL is disabled, returns http://hostname to prevent auto-HTTPS
func formatSiteAddress(hostname string, sslEnabled bool) string {
	if !sslEnabled {
		return "http://" + hostname
	}
	return hostname
}

// sanitizeFilename removes unsafe characters from filename
func sanitizeFilename(name string) string {
	// Replace dots with underscores, remove other unsafe chars
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	sanitized := reg.ReplaceAllString(name, "_")
	// Remove consecutive underscores
	reg = regexp.MustCompile(`_+`)
	sanitized = reg.ReplaceAllString(sanitized, "_")
	// Trim underscores from ends
	sanitized = strings.Trim(sanitized, "_")
	return sanitized
}
