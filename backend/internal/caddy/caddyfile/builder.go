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
	logger *zap.Logger
}

// NewBuilder creates a new Caddyfile builder
func NewBuilder(logger *zap.Logger) *Builder {
	return &Builder{
		logger: logger,
	}
}

// MainCaddyfileOptions holds options for building the main Caddyfile
type MainCaddyfileOptions struct {
	Email            string
	DisableAutoHTTPS bool
}

// BuildMainCaddyfile generates the main Caddyfile with global options and imports
func (b *Builder) BuildMainCaddyfile(opts MainCaddyfileOptions) string {
	var sb strings.Builder

	sb.WriteString("# Managed by Waygates - Do not edit manually\n")
	sb.WriteString(fmt.Sprintf("# Generated: %s\n\n", time.Now().Format(time.RFC3339)))

	// Global options block
	sb.WriteString("{\n")
	if opts.Email != "" {
		sb.WriteString(fmt.Sprintf("\temail %s\n", opts.Email))
	}
	if opts.DisableAutoHTTPS {
		sb.WriteString("\tauto_https off\n")
	} else {
		sb.WriteString("\tacme_dns cloudflare {$CLOUDFLARE_API_TOKEN}\n")
	}
	sb.WriteString("\tadmin localhost:2019\n")
	sb.WriteString("}\n\n")

	// Import proxy configs
	sb.WriteString("import sites/*.conf\n\n")

	// Import catch-all (must be last)
	sb.WriteString("import catchall.conf\n")

	return sb.String()
}

// BuildProxyFile generates config content for a single proxy
func (b *Builder) BuildProxyFile(proxy *models.Proxy) (string, error) {
	if proxy == nil {
		return "", fmt.Errorf("proxy is nil")
	}

	var content string
	var err error

	switch proxy.Type {
	case models.ProxyTypeReverseProxy:
		content, err = b.buildReverseProxyBlock(proxy)
	case models.ProxyTypeStatic:
		content, err = b.buildStaticBlock(proxy)
	case models.ProxyTypeRedirect:
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
