package caddyfile

import (
	"fmt"
	"strings"

	"github.com/aloks98/waygates/backend/internal/models"
)

// buildRedirectBlock generates a redirect site block
func (b *Builder) buildRedirectBlock(proxy *models.Proxy) (string, error) {
	if len(proxy.RedirectConfig) == 0 {
		return "", fmt.Errorf("redirect proxy requires redirect configuration")
	}

	target, _ := proxy.RedirectConfig["target"].(string)
	if target == "" {
		return "", fmt.Errorf("redirect proxy requires target URL")
	}

	var sb strings.Builder

	// Site block with hostname (use http:// prefix if SSL disabled)
	siteAddr := formatSiteAddress(proxy.Hostname, proxy.SSLEnabled)
	sb.WriteString(fmt.Sprintf("%s {\n", siteAddr))

	// Build redirect target with optional path/query preservation
	redirectTarget := b.buildRedirectTarget(proxy.RedirectConfig)

	// Status code (default to 301 permanent)
	statusCode := 301
	if sc, ok := proxy.RedirectConfig["status_code"].(float64); ok && sc > 0 {
		statusCode = int(sc)
	}

	// Map status code to Caddy keyword or use number
	statusKeyword := mapRedirectStatus(statusCode)

	sb.WriteString(fmt.Sprintf("\tredir %s %s\n", redirectTarget, statusKeyword))
	sb.WriteString("}\n")

	return sb.String(), nil
}

// buildRedirectTarget constructs the redirect target URL with optional placeholders
func (b *Builder) buildRedirectTarget(redirect models.JSONField) string {
	target, _ := redirect["target"].(string)

	preservePath, _ := redirect["preserve_path"].(bool)
	preserveQuery, _ := redirect["preserve_query"].(bool)

	// Add path preservation
	if preservePath {
		// Remove trailing slash from target to avoid double slashes
		target = strings.TrimSuffix(target, "/")
		target += "{uri}"
	}

	// If only preserving query (not path), append query placeholder
	if !preservePath && preserveQuery {
		if !strings.Contains(target, "?") {
			target += "?{query}"
		} else {
			target += "&{query}"
		}
	}

	return target
}

// mapRedirectStatus maps HTTP status codes to Caddy keywords
func mapRedirectStatus(code int) string {
	switch code {
	case 301:
		return "permanent"
	case 302:
		return "temporary"
	case 303:
		return "303"
	case 307:
		return "307"
	case 308:
		return "308"
	default:
		return fmt.Sprintf("%d", code)
	}
}
