package caddyfile

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aloks98/waygates/backend/internal/models"
)

// ACLBuilder generates Caddy ACL directives for proxy configurations.
// It supports various authentication methods including IP rules, basic auth,
// forward auth (Waygates), and external providers (Authelia, Authentik).
type ACLBuilder struct {
	waygatesVerifyURL string // e.g., http://localhost:8080 (internal URL for Caddy)
	waygatesLoginURL  string // e.g., https://waygates.company.com/auth/login (external URL for users)
}

// NewACLBuilder creates a new ACL builder with the specified Waygates URLs.
func NewACLBuilder(waygatesVerifyURL, waygatesLoginURL string) *ACLBuilder {
	return &ACLBuilder{
		waygatesVerifyURL: waygatesVerifyURL,
		waygatesLoginURL:  waygatesLoginURL,
	}
}

// Default headers to copy from Waygates forward auth responses
var waygatesDefaultHeaders = []string{
	"X-Auth-User",
	"X-Auth-User-ID",
	"X-Auth-User-Email",
}

// Static asset extensions that bypass ACL authentication
// These are common static files that don't need authentication
var staticAssetExtensions = []string{
	".ico", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".avif",
	".css", ".js", ".mjs",
	".woff", ".woff2", ".ttf", ".eot", ".otf",
	".webmanifest", ".map",
}

// Static asset paths that bypass ACL authentication
var staticAssetPaths = []string{
	"/favicon.ico",
	"/robots.txt",
	"/sitemap.xml",
}

// Provider-specific default headers
var providerDefaultHeaders = map[string][]string{
	models.ACLProviderTypeAuthelia: {
		"Remote-User",
		"Remote-Groups",
		"Remote-Name",
		"Remote-Email",
	},
	models.ACLProviderTypeAuthentik: {
		"X-authentik-username",
		"X-authentik-groups",
		"X-authentik-email",
		"X-authentik-name",
		"X-authentik-uid",
	},
}

// forbiddenHTML is the HTML template shown when access is denied (403)
const forbiddenHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Access Denied</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #e2e8f0;
        }
        .container {
            text-align: center;
            padding: 2rem;
            max-width: 500px;
        }
        .icon {
            width: 80px;
            height: 80px;
            margin: 0 auto 1.5rem;
            background: rgba(239, 68, 68, 0.1);
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .icon svg {
            width: 40px;
            height: 40px;
            color: #ef4444;
        }
        h1 {
            font-size: 1.875rem;
            font-weight: 600;
            margin-bottom: 0.75rem;
            color: #f8fafc;
        }
        .code {
            font-size: 4rem;
            font-weight: 700;
            color: #ef4444;
            margin-bottom: 0.5rem;
        }
        p {
            color: #94a3b8;
            margin-bottom: 1.5rem;
            line-height: 1.6;
        }
        .btn {
            display: inline-block;
            padding: 0.75rem 1.5rem;
            background: #3b82f6;
            color: white;
            text-decoration: none;
            border-radius: 0.5rem;
            font-weight: 500;
            transition: background 0.2s;
        }
        .btn:hover {
            background: #2563eb;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
            </svg>
        </div>
        <div class="code">403</div>
        <h1>Access Denied</h1>
        <p>You don't have permission to access this resource. Please contact your administrator if you believe this is an error.</p>
        <a href="javascript:history.back()" class="btn">Go Back</a>
    </div>
</body>
</html>`

// unauthorizedHTML is the HTML template shown when authentication is required (401)
const unauthorizedHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Authentication Required</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #e2e8f0;
        }
        .container {
            text-align: center;
            padding: 2rem;
            max-width: 500px;
        }
        .icon {
            width: 80px;
            height: 80px;
            margin: 0 auto 1.5rem;
            background: rgba(234, 179, 8, 0.1);
            border-radius: 50%;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .icon svg {
            width: 40px;
            height: 40px;
            color: #eab308;
        }
        h1 {
            font-size: 1.875rem;
            font-weight: 600;
            margin-bottom: 0.75rem;
            color: #f8fafc;
        }
        .code {
            font-size: 4rem;
            font-weight: 700;
            color: #eab308;
            margin-bottom: 0.5rem;
        }
        p {
            color: #94a3b8;
            margin-bottom: 1.5rem;
            line-height: 1.6;
        }
        .btn {
            display: inline-block;
            padding: 0.75rem 1.5rem;
            background: #3b82f6;
            color: white;
            text-decoration: none;
            border-radius: 0.5rem;
            font-weight: 500;
            transition: background 0.2s;
        }
        .btn:hover {
            background: #2563eb;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
            </svg>
        </div>
        <div class="code">401</div>
        <h1>Authentication Required</h1>
        <p>You need to sign in to access this resource. Please authenticate to continue.</p>
        <a href="javascript:history.back()" class="btn">Go Back</a>
    </div>
</body>
</html>`

// ACLConfig holds the complete ACL configuration for a proxy
type ACLConfig struct {
	Proxy       *models.Proxy
	Assignments []models.ProxyACLAssignment
}

// BuildACLConfig generates ACL configuration for a proxy.
// Returns an empty string if no ACL assignments exist.
func (b *ACLBuilder) BuildACLConfig(proxy *models.Proxy, assignments []models.ProxyACLAssignment) string {
	if len(assignments) == 0 {
		return ""
	}

	// Filter enabled assignments and sort by priority (lower = higher priority)
	enabledAssignments := filterEnabledAssignments(assignments)
	if len(enabledAssignments) == 0 {
		return ""
	}

	sort.Slice(enabledAssignments, func(i, j int) bool {
		return enabledAssignments[i].Priority < enabledAssignments[j].Priority
	})

	var sb strings.Builder

	// Process each assignment
	for idx, assignment := range enabledAssignments {
		if assignment.ACLGroup == nil {
			continue
		}

		config := b.buildAssignmentConfig(proxy, assignment, idx)
		if config != "" {
			sb.WriteString(config)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// filterEnabledAssignments returns only enabled assignments with loaded ACL groups
func filterEnabledAssignments(assignments []models.ProxyACLAssignment) []models.ProxyACLAssignment {
	var enabled []models.ProxyACLAssignment
	for _, a := range assignments {
		if a.Enabled && a.ACLGroup != nil {
			enabled = append(enabled, a)
		}
	}
	return enabled
}

// buildAssignmentConfig generates config for a single ACL assignment
func (b *ACLBuilder) buildAssignmentConfig(proxy *models.Proxy, assignment models.ProxyACLAssignment, idx int) string {
	group := assignment.ACLGroup
	pathPattern := assignment.PathPattern

	// Analyze what authentication methods are configured
	hasIPRules := len(group.IPRules) > 0
	hasBasicAuth := len(group.BasicAuthUsers) > 0
	hasWaygatesAuth := group.WaygatesAuth != nil && group.WaygatesAuth.Enabled
	hasExternalProviders := len(group.ExternalProviders) > 0

	// If no auth methods configured, skip
	if !hasIPRules && !hasBasicAuth && !hasWaygatesAuth && !hasExternalProviders {
		return ""
	}

	var sb strings.Builder
	matcherPrefix := fmt.Sprintf("acl_%d", idx)

	// Generate config based on combination mode
	switch group.CombinationMode {
	case models.ACLCombinationModeAny:
		sb.WriteString(b.buildAnyModeConfig(proxy, group, pathPattern, matcherPrefix))
	case models.ACLCombinationModeAll:
		sb.WriteString(b.buildAllModeConfig(proxy, group, pathPattern, matcherPrefix))
	case models.ACLCombinationModeIPBypass:
		sb.WriteString(b.buildIPBypassModeConfig(proxy, group, pathPattern, matcherPrefix))
	default:
		// Default to "any" mode
		sb.WriteString(b.buildAnyModeConfig(proxy, group, pathPattern, matcherPrefix))
	}

	return sb.String()
}

// buildAnyModeConfig generates config where ANY auth method can grant access (OR logic)
// IP bypass rules skip auth entirely
// IP allow rules grant access without further auth
// Otherwise, forward_auth is checked
func (b *ACLBuilder) buildAnyModeConfig(proxy *models.Proxy, group *models.ACLGroup, pathPattern, matcherPrefix string) string {
	var sb strings.Builder

	// Categorize IP rules
	bypassIPs, allowIPs, denyIPs := categorizeIPRules(group.IPRules)

	// 1. Handle IP deny rules first (highest priority)
	if len(denyIPs) > 0 {
		sb.WriteString(b.buildIPDenyBlock(pathPattern, matcherPrefix, denyIPs))
	}

	// 2. Handle IP bypass rules (skip all auth)
	if len(bypassIPs) > 0 {
		sb.WriteString(b.buildIPBypassBlock(proxy, pathPattern, matcherPrefix, bypassIPs))
	}

	// 3. Handle IP allow rules (grant access without auth)
	if len(allowIPs) > 0 {
		sb.WriteString(b.buildIPAllowBlock(proxy, pathPattern, matcherPrefix, allowIPs))
	}

	// 4. Handle static assets bypass (skip auth for common static files)
	sb.WriteString(b.buildStaticAssetsBypassBlock(proxy, matcherPrefix))

	// 5. Handle remaining requests with authentication
	hasForwardAuth := group.WaygatesAuth != nil && group.WaygatesAuth.Enabled
	hasOAuthRestrictions := len(group.OAuthProviderRestrictions) > 0
	hasExternalAuth := len(group.ExternalProviders) > 0
	hasBasicAuth := len(group.BasicAuthUsers) > 0

	// Use forward auth if Waygates auth, OAuth restrictions, or external providers are configured.
	// Basic auth is only used when it's the only auth method (more secure methods override it).
	hasSecureAuth := hasForwardAuth || hasOAuthRestrictions || hasExternalAuth
	if hasBasicAuth && !hasSecureAuth {
		// Only basic auth configured (no more secure auth methods)
		sb.WriteString(b.buildBasicAuthBlock(proxy, group, pathPattern, matcherPrefix, bypassIPs, allowIPs))
	} else if hasSecureAuth {
		// Forward auth (Waygates, OAuth, or external provider)
		sb.WriteString(b.buildForwardAuthBlock(proxy, group, pathPattern, matcherPrefix, bypassIPs, allowIPs))
	}

	return sb.String()
}

// buildAllModeConfig generates config where ALL auth methods must pass (AND logic)
// IP rules must match AND auth must pass
func (b *ACLBuilder) buildAllModeConfig(proxy *models.Proxy, group *models.ACLGroup, pathPattern, matcherPrefix string) string {
	var sb strings.Builder

	// Categorize IP rules
	bypassIPs, allowIPs, denyIPs := categorizeIPRules(group.IPRules)

	// 1. Handle IP deny rules first
	if len(denyIPs) > 0 {
		sb.WriteString(b.buildIPDenyBlock(pathPattern, matcherPrefix, denyIPs))
	}

	// 2. For ALL mode with IP rules, requests must come from allowed IPs AND pass auth
	allAllowedIPs := make([]string, 0, len(bypassIPs)+len(allowIPs))
	allAllowedIPs = append(allAllowedIPs, bypassIPs...)
	allAllowedIPs = append(allAllowedIPs, allowIPs...)
	if len(allAllowedIPs) > 0 {
		// Deny requests not from allowed IPs
		sb.WriteString(b.buildIPDenyNotInListBlock(pathPattern, matcherPrefix, allAllowedIPs))
	}

	// 3. Requests from allowed IPs still need auth check
	hasForwardAuth := group.WaygatesAuth != nil && group.WaygatesAuth.Enabled
	hasOAuthRestrictions := len(group.OAuthProviderRestrictions) > 0
	hasExternalAuth := len(group.ExternalProviders) > 0
	hasBasicAuth := len(group.BasicAuthUsers) > 0

	// Use forward auth if Waygates auth, OAuth restrictions, or external providers are configured.
	// Basic auth is only used when it's the only auth method (more secure methods override it).
	hasSecureAuth := hasForwardAuth || hasOAuthRestrictions || hasExternalAuth
	if hasBasicAuth && !hasSecureAuth {
		sb.WriteString(b.buildBasicAuthBlockAll(proxy, group, pathPattern, matcherPrefix, allAllowedIPs))
	} else if hasSecureAuth {
		sb.WriteString(b.buildForwardAuthBlockAll(proxy, group, pathPattern, matcherPrefix, allAllowedIPs))
	}

	return sb.String()
}

// buildIPBypassModeConfig generates config where IP rules can bypass auth
// IP bypass: skip auth entirely
// IP allow: pre-authenticated but may need group check
// Others: forward_auth required
func (b *ACLBuilder) buildIPBypassModeConfig(proxy *models.Proxy, group *models.ACLGroup, pathPattern, matcherPrefix string) string {
	// IP bypass mode is similar to ANY mode with specific IP bypass handling
	return b.buildAnyModeConfig(proxy, group, pathPattern, matcherPrefix)
}

// categorizeIPRules separates IP rules by type
func categorizeIPRules(rules []models.ACLIPRule) (bypass, allow, deny []string) {
	for _, rule := range rules {
		switch rule.RuleType {
		case models.ACLIPRuleTypeBypass:
			bypass = append(bypass, rule.CIDR)
		case models.ACLIPRuleTypeAllow:
			allow = append(allow, rule.CIDR)
		case models.ACLIPRuleTypeDeny:
			deny = append(deny, rule.CIDR)
		}
	}
	return
}

// buildIPDenyBlock generates configuration to deny requests from specific IPs
func (b *ACLBuilder) buildIPDenyBlock(pathPattern, matcherPrefix string, denyIPs []string) string {
	var sb strings.Builder

	matcherName := fmt.Sprintf("@%s_denied_ips", matcherPrefix)

	sb.WriteString(fmt.Sprintf("\t%s {\n", matcherName))
	if pathPattern != "" && pathPattern != "/*" {
		sb.WriteString(fmt.Sprintf("\t\tpath %s\n", pathPattern))
	}
	sb.WriteString(fmt.Sprintf("\t\tremote_ip %s\n", strings.Join(denyIPs, " ")))
	sb.WriteString("\t}\n")
	sb.WriteString(fmt.Sprintf("\trespond %s \"Forbidden\" 403\n\n", matcherName))

	return sb.String()
}

// buildIPDenyNotInListBlock generates configuration to deny requests NOT from allowed IPs
func (b *ACLBuilder) buildIPDenyNotInListBlock(pathPattern, matcherPrefix string, allowedIPs []string) string {
	var sb strings.Builder

	matcherName := fmt.Sprintf("@%s_not_allowed_ip", matcherPrefix)

	sb.WriteString(fmt.Sprintf("\t%s {\n", matcherName))
	if pathPattern != "" && pathPattern != "/*" {
		sb.WriteString(fmt.Sprintf("\t\tpath %s\n", pathPattern))
	}
	sb.WriteString(fmt.Sprintf("\t\tnot remote_ip %s\n", strings.Join(allowedIPs, " ")))
	sb.WriteString("\t}\n")
	sb.WriteString(fmt.Sprintf("\trespond %s \"Forbidden\" 403\n\n", matcherName))

	return sb.String()
}

// buildIPBypassBlock generates configuration for IP bypass (skip all auth)
func (b *ACLBuilder) buildIPBypassBlock(proxy *models.Proxy, pathPattern, matcherPrefix string, bypassIPs []string) string {
	var sb strings.Builder

	matcherName := fmt.Sprintf("@%s_bypass_ip", matcherPrefix)

	sb.WriteString(fmt.Sprintf("\t%s {\n", matcherName))
	if pathPattern != "" && pathPattern != "/*" {
		sb.WriteString(fmt.Sprintf("\t\tpath %s\n", pathPattern))
	}
	sb.WriteString(fmt.Sprintf("\t\tremote_ip %s\n", strings.Join(bypassIPs, " ")))
	sb.WriteString("\t}\n")
	sb.WriteString(fmt.Sprintf("\thandle %s {\n", matcherName))
	sb.WriteString(b.buildReverseProxyDirective(proxy, "\t\t"))
	sb.WriteString("\t}\n\n")

	return sb.String()
}

// buildIPAllowBlock generates configuration for IP allow (access without auth)
func (b *ACLBuilder) buildIPAllowBlock(proxy *models.Proxy, pathPattern, matcherPrefix string, allowIPs []string) string {
	var sb strings.Builder

	matcherName := fmt.Sprintf("@%s_allowed_ip", matcherPrefix)

	sb.WriteString(fmt.Sprintf("\t%s {\n", matcherName))
	if pathPattern != "" && pathPattern != "/*" {
		sb.WriteString(fmt.Sprintf("\t\tpath %s\n", pathPattern))
	}
	sb.WriteString(fmt.Sprintf("\t\tremote_ip %s\n", strings.Join(allowIPs, " ")))
	sb.WriteString("\t}\n")
	sb.WriteString(fmt.Sprintf("\thandle %s {\n", matcherName))
	sb.WriteString(b.buildReverseProxyDirective(proxy, "\t\t"))
	sb.WriteString("\t}\n\n")

	return sb.String()
}

// buildStaticAssetsBypassBlock generates configuration to bypass auth for static assets
// This allows common static files (images, CSS, JS, fonts, etc.) to be served without authentication
func (b *ACLBuilder) buildStaticAssetsBypassBlock(proxy *models.Proxy, matcherPrefix string) string {
	var sb strings.Builder

	matcherName := fmt.Sprintf("@%s_static_assets", matcherPrefix)

	// Build path patterns for static assets
	var pathPatterns []string

	// Add file extension patterns
	for _, ext := range staticAssetExtensions {
		pathPatterns = append(pathPatterns, fmt.Sprintf("*%s", ext))
	}

	// Add specific paths
	pathPatterns = append(pathPatterns, staticAssetPaths...)

	sb.WriteString(fmt.Sprintf("\t%s {\n", matcherName))
	sb.WriteString(fmt.Sprintf("\t\tpath %s\n", strings.Join(pathPatterns, " ")))
	sb.WriteString("\t}\n")
	sb.WriteString(fmt.Sprintf("\thandle %s {\n", matcherName))
	sb.WriteString(b.buildReverseProxyDirective(proxy, "\t\t"))
	sb.WriteString("\t}\n\n")

	return sb.String()
}

// buildBasicAuthBlock generates basic auth configuration
func (b *ACLBuilder) buildBasicAuthBlock(proxy *models.Proxy, group *models.ACLGroup, pathPattern, matcherPrefix string, bypassIPs, allowIPs []string) string {
	var sb strings.Builder

	matcherName := fmt.Sprintf("@%s_basic_auth", matcherPrefix)

	// Exclude bypass and allow IPs
	excludeIPs := make([]string, 0, len(bypassIPs)+len(allowIPs))
	excludeIPs = append(excludeIPs, bypassIPs...)
	excludeIPs = append(excludeIPs, allowIPs...)

	hasPathCondition := pathPattern != "" && pathPattern != "/*"
	hasIPCondition := len(excludeIPs) > 0

	// Build matcher that excludes already handled IPs
	sb.WriteString(fmt.Sprintf("\t%s {\n", matcherName))
	if hasPathCondition {
		sb.WriteString(fmt.Sprintf("\t\tpath %s\n", pathPattern))
	} else if !hasIPCondition {
		// If no conditions at all, add wildcard path to match everything
		sb.WriteString("\t\tpath *\n")
	}

	if hasIPCondition {
		sb.WriteString(fmt.Sprintf("\t\tnot remote_ip %s\n", strings.Join(excludeIPs, " ")))
	}
	sb.WriteString("\t}\n")

	sb.WriteString(fmt.Sprintf("\thandle %s {\n", matcherName))
	sb.WriteString("\t\tbasicauth {\n")
	for _, user := range group.BasicAuthUsers {
		sb.WriteString(fmt.Sprintf("\t\t\t%s %s\n", user.Username, user.PasswordHash))
	}
	sb.WriteString("\t\t}\n")
	sb.WriteString(b.buildReverseProxyDirective(proxy, "\t\t"))
	sb.WriteString("\t}\n\n")

	return sb.String()
}

// buildBasicAuthBlockAll generates basic auth configuration for ALL mode (requires IP match)
func (b *ACLBuilder) buildBasicAuthBlockAll(proxy *models.Proxy, group *models.ACLGroup, pathPattern, matcherPrefix string, allowedIPs []string) string {
	var sb strings.Builder

	matcherName := fmt.Sprintf("@%s_basic_auth_all", matcherPrefix)

	hasPathCondition := pathPattern != "" && pathPattern != "/*"
	hasIPCondition := len(allowedIPs) > 0

	sb.WriteString(fmt.Sprintf("\t%s {\n", matcherName))
	if hasPathCondition {
		sb.WriteString(fmt.Sprintf("\t\tpath %s\n", pathPattern))
	} else if !hasIPCondition {
		// If no conditions at all, add wildcard path to match everything
		sb.WriteString("\t\tpath *\n")
	}
	if hasIPCondition {
		sb.WriteString(fmt.Sprintf("\t\tremote_ip %s\n", strings.Join(allowedIPs, " ")))
	}
	sb.WriteString("\t}\n")

	sb.WriteString(fmt.Sprintf("\thandle %s {\n", matcherName))
	sb.WriteString("\t\tbasicauth {\n")
	for _, user := range group.BasicAuthUsers {
		sb.WriteString(fmt.Sprintf("\t\t\t%s %s\n", user.Username, user.PasswordHash))
	}
	sb.WriteString("\t\t}\n")
	sb.WriteString(b.buildReverseProxyDirective(proxy, "\t\t"))
	sb.WriteString("\t}\n\n")

	return sb.String()
}

// buildForwardAuthBlock generates forward auth configuration
func (b *ACLBuilder) buildForwardAuthBlock(proxy *models.Proxy, group *models.ACLGroup, pathPattern, matcherPrefix string, bypassIPs, allowIPs []string) string {
	var sb strings.Builder

	matcherName := fmt.Sprintf("@%s_forward_auth", matcherPrefix)

	// Exclude bypass and allow IPs
	excludeIPs := make([]string, 0, len(bypassIPs)+len(allowIPs))
	excludeIPs = append(excludeIPs, bypassIPs...)
	excludeIPs = append(excludeIPs, allowIPs...)

	hasPathCondition := pathPattern != "" && pathPattern != "/*"
	hasIPCondition := len(excludeIPs) > 0

	// Build matcher that excludes already handled IPs
	sb.WriteString(fmt.Sprintf("\t%s {\n", matcherName))
	if hasPathCondition {
		sb.WriteString(fmt.Sprintf("\t\tpath %s\n", pathPattern))
	} else if !hasIPCondition {
		// If no conditions at all, add wildcard path to match everything
		// An empty matcher {} matches nothing in Caddy
		sb.WriteString("\t\tpath *\n")
	}

	if hasIPCondition {
		sb.WriteString(fmt.Sprintf("\t\tnot remote_ip %s\n", strings.Join(excludeIPs, " ")))
	}
	sb.WriteString("\t}\n")

	sb.WriteString(fmt.Sprintf("\thandle %s {\n", matcherName))

	// Determine which forward auth to use
	if len(group.ExternalProviders) > 0 {
		// Use first external provider
		provider := group.ExternalProviders[0]
		sb.WriteString(b.buildExternalForwardAuth(provider, "\t\t"))
	} else if group.WaygatesAuth != nil && group.WaygatesAuth.Enabled {
		sb.WriteString(b.buildWaygatesForwardAuth("\t\t"))
	}

	sb.WriteString(b.buildReverseProxyDirective(proxy, "\t\t"))
	sb.WriteString("\t}\n\n")

	return sb.String()
}

// buildForwardAuthBlockAll generates forward auth configuration for ALL mode
func (b *ACLBuilder) buildForwardAuthBlockAll(proxy *models.Proxy, group *models.ACLGroup, pathPattern, matcherPrefix string, allowedIPs []string) string {
	var sb strings.Builder

	matcherName := fmt.Sprintf("@%s_forward_auth_all", matcherPrefix)

	hasPathCondition := pathPattern != "" && pathPattern != "/*"
	hasIPCondition := len(allowedIPs) > 0

	sb.WriteString(fmt.Sprintf("\t%s {\n", matcherName))
	if hasPathCondition {
		sb.WriteString(fmt.Sprintf("\t\tpath %s\n", pathPattern))
	} else if !hasIPCondition {
		// If no conditions at all, add wildcard path to match everything
		sb.WriteString("\t\tpath *\n")
	}
	if hasIPCondition {
		sb.WriteString(fmt.Sprintf("\t\tremote_ip %s\n", strings.Join(allowedIPs, " ")))
	}
	sb.WriteString("\t}\n")

	sb.WriteString(fmt.Sprintf("\thandle %s {\n", matcherName))

	// Determine which forward auth to use
	if len(group.ExternalProviders) > 0 {
		provider := group.ExternalProviders[0]
		sb.WriteString(b.buildExternalForwardAuth(provider, "\t\t"))
	} else if group.WaygatesAuth != nil && group.WaygatesAuth.Enabled {
		sb.WriteString(b.buildWaygatesForwardAuth("\t\t"))
	}

	sb.WriteString(b.buildReverseProxyDirective(proxy, "\t\t"))
	sb.WriteString("\t}\n\n")

	return sb.String()
}

// buildWaygatesForwardAuth generates Waygates forward auth directive
func (b *ACLBuilder) buildWaygatesForwardAuth(indent string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%sforward_auth %s {\n", indent, b.waygatesVerifyURL))
	sb.WriteString(fmt.Sprintf("%s\turi /api/auth/acl/verify\n", indent))
	sb.WriteString(fmt.Sprintf("%s\tcopy_headers %s\n", indent, strings.Join(waygatesDefaultHeaders, " ")))

	// Handle 401 response
	sb.WriteString(fmt.Sprintf("%s\t@unauthorized status 401\n", indent))
	sb.WriteString(fmt.Sprintf("%s\thandle_response @unauthorized {\n", indent))
	if b.waygatesLoginURL != "" {
		// Redirect to login page with original URL
		// {scheme}://{host}{uri} captures the original URL the user was trying to access
		sb.WriteString(fmt.Sprintf("%s\t\tredir %s?redirect={scheme}://{host}{uri} 302\n", indent, b.waygatesLoginURL))
	} else {
		// No login URL configured, show error page
		sb.WriteString(fmt.Sprintf("%s\t\theader Content-Type text/html\n", indent))
		sb.WriteString(fmt.Sprintf("%s\t\trespond <<HTML\n", indent))
		sb.WriteString(unauthorizedHTML)
		sb.WriteString(fmt.Sprintf("\n%s\t\tHTML 401\n", indent))
	}
	sb.WriteString(fmt.Sprintf("%s\t}\n", indent))

	// Handle 403 response - redirect to error page with denial details
	sb.WriteString(fmt.Sprintf("%s\t@forbidden status 403\n", indent))
	sb.WriteString(fmt.Sprintf("%s\thandle_response @forbidden {\n", indent))
	if b.waygatesLoginURL != "" {
		// Extract base URL from login URL (remove /auth/login path)
		errorPageBase := b.waygatesLoginURL
		if idx := strings.Index(errorPageBase, "/auth/login"); idx != -1 {
			errorPageBase = errorPageBase[:idx]
		}
		// Redirect to error page with reason from upstream header
		// The {rp.header.X-Auth-Denial-Reason} placeholder captures the denial reason
		sb.WriteString(fmt.Sprintf("%s\t\tredir %s/auth/forbidden?reason={rp.header.X-Auth-Denial-Reason}&email={rp.header.X-Auth-User-Email}&provider={rp.header.X-Auth-Provider}&url={scheme}://{host}{uri} 302\n", indent, errorPageBase))
	} else {
		// No login URL configured, show static error page
		sb.WriteString(fmt.Sprintf("%s\t\theader Content-Type text/html\n", indent))
		sb.WriteString(fmt.Sprintf("%s\t\trespond <<HTML\n", indent))
		sb.WriteString(forbiddenHTML)
		sb.WriteString(fmt.Sprintf("\n%s\t\tHTML 403\n", indent))
	}
	sb.WriteString(fmt.Sprintf("%s\t}\n", indent))

	sb.WriteString(fmt.Sprintf("%s}\n", indent))

	return sb.String()
}

// buildExternalForwardAuth generates external provider forward auth directive
func (b *ACLBuilder) buildExternalForwardAuth(provider models.ACLExternalProvider, indent string) string {
	var sb strings.Builder

	// Build the verify URL with optional redirect
	verifyURL := provider.VerifyURL
	if provider.AuthRedirectURL != nil && *provider.AuthRedirectURL != "" {
		// Some providers need the redirect URL as a query parameter
		if strings.Contains(verifyURL, "?") {
			verifyURL = fmt.Sprintf("%s&rd=%s", verifyURL, *provider.AuthRedirectURL)
		} else {
			verifyURL = fmt.Sprintf("%s?rd=%s", verifyURL, *provider.AuthRedirectURL)
		}
	}

	sb.WriteString(fmt.Sprintf("%sforward_auth %s {\n", indent, verifyURL))

	// Use custom headers if specified, otherwise use provider defaults
	headers := provider.HeadersToCopy
	if len(headers) == 0 {
		if defaultHeaders, ok := providerDefaultHeaders[provider.ProviderType]; ok {
			headers = defaultHeaders
		}
	}

	if len(headers) > 0 {
		sb.WriteString(fmt.Sprintf("%s\tcopy_headers %s\n", indent, strings.Join(headers, " ")))
	}

	sb.WriteString(fmt.Sprintf("%s}\n", indent))

	return sb.String()
}

// buildReverseProxyDirective generates the reverse_proxy directive for the proxy
func (b *ACLBuilder) buildReverseProxyDirective(proxy *models.Proxy, indent string) string {
	if proxy.Upstreams == nil {
		return ""
	}

	upstreams, ok := proxy.Upstreams.([]interface{})
	if !ok || len(upstreams) == 0 {
		return ""
	}

	var sb strings.Builder

	// Build upstream addresses
	addresses := make([]string, 0, len(upstreams))
	var hasHTTPS bool

	for _, up := range upstreams {
		upstreamMap, ok := up.(map[string]interface{})
		if !ok {
			continue
		}

		host, _ := upstreamMap["host"].(string)
		port, _ := upstreamMap["port"].(float64)
		scheme, _ := upstreamMap["scheme"].(string)

		if scheme == "https" {
			hasHTTPS = true
		}

		addr := fmt.Sprintf("%s:%d", host, int(port))
		addresses = append(addresses, addr)
	}

	sb.WriteString(fmt.Sprintf("%sreverse_proxy %s {\n", indent, strings.Join(addresses, " ")))

	// Transport config for HTTPS upstreams
	if hasHTTPS || proxy.TLSInsecureSkipVerify {
		sb.WriteString(fmt.Sprintf("%s\ttransport http {\n", indent))
		if hasHTTPS {
			sb.WriteString(fmt.Sprintf("%s\t\ttls\n", indent))
		}
		if proxy.TLSInsecureSkipVerify {
			sb.WriteString(fmt.Sprintf("%s\t\ttls_insecure_skip_verify\n", indent))
		}
		sb.WriteString(fmt.Sprintf("%s\t}\n", indent))
	}

	// Standard headers
	sb.WriteString(fmt.Sprintf("%s\theader_up X-Real-IP {remote_host}\n", indent))
	sb.WriteString(fmt.Sprintf("%s\theader_up X-Forwarded-For {remote_host}\n", indent))
	sb.WriteString(fmt.Sprintf("%s\theader_up X-Forwarded-Proto {scheme}\n", indent))
	sb.WriteString(fmt.Sprintf("%s\theader_up X-Forwarded-Host {host}\n", indent))

	// Custom headers
	if len(proxy.CustomHeaders) > 0 {
		for key, value := range proxy.CustomHeaders {
			if strVal, ok := value.(string); ok {
				sb.WriteString(fmt.Sprintf("%s\theader_up %s %q\n", indent, key, strVal))
			}
		}
	}

	sb.WriteString(fmt.Sprintf("%s}\n", indent))

	return sb.String()
}

// HasACLConfig checks if proxy has any enabled ACL assignments
func HasACLConfig(assignments []models.ProxyACLAssignment) bool {
	for _, a := range assignments {
		if a.Enabled && a.ACLGroup != nil {
			return true
		}
	}
	return false
}

// =============================================================================
// Union ACL Config Builder
// =============================================================================

// deduplicateCIDRs removes duplicate CIDR entries while preserving order.
func deduplicateCIDRs(cidrs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(cidrs))
	for _, cidr := range cidrs {
		if !seen[cidr] {
			seen[cidr] = true
			result = append(result, cidr)
		}
	}
	return result
}

// collectUnionIPRules collects all IP rules from all enabled assignments grouped by type.
// Returns deduplicated slices of deny, bypass, and allow CIDRs.
func collectUnionIPRules(assignments []models.ProxyACLAssignment) (denyRules, bypassRules, allowRules []string) {
	for _, assignment := range assignments {
		if !assignment.Enabled || assignment.ACLGroup == nil {
			continue
		}
		for _, rule := range assignment.ACLGroup.IPRules {
			switch rule.RuleType {
			case models.ACLIPRuleTypeDeny:
				denyRules = append(denyRules, rule.CIDR)
			case models.ACLIPRuleTypeBypass:
				bypassRules = append(bypassRules, rule.CIDR)
			case models.ACLIPRuleTypeAllow:
				allowRules = append(allowRules, rule.CIDR)
			}
		}
	}
	return deduplicateCIDRs(denyRules), deduplicateCIDRs(bypassRules), deduplicateCIDRs(allowRules)
}

// BuildUnionACLConfig generates Caddyfile config combining IP rules from all ACL groups
// into unified matchers. This creates a single set of deny, bypass, and forward_auth
// directives that represent the union of all assigned ACL groups.
//
// The generated config follows this order:
//  1. Deny matcher - blocks requests from any denied IP across all groups
//  2. Bypass matcher - allows requests from bypass IPs to skip authentication
//  3. Forward auth - requires authentication for all other requests
//
// Example output for two groups with deny 10.0.10.0/24 and deny 10.0.12.0/24, bypass 192.168.1.0/24:
//
//	@denied_ips {
//	    remote_ip 10.0.10.0/24
//	    remote_ip 10.0.12.0/24
//	}
//	respond @denied_ips 403
//
//	@bypass_ips {
//	    remote_ip 192.168.1.0/24
//	}
//
//	@needs_auth {
//	    not {
//	        remote_ip 192.168.1.0/24
//	    }
//	}
//
//	forward_auth @needs_auth localhost:8080 {
//	    uri /api/auth/acl/verify
//	    copy_headers Remote-User Remote-Groups Remote-Email X-Forwarded-User
//	}
func (b *ACLBuilder) BuildUnionACLConfig(assignments []models.ProxyACLAssignment) string {
	if len(assignments) == 0 {
		return ""
	}

	// Check if any assignment is enabled
	hasEnabled := false
	for _, a := range assignments {
		if a.Enabled && a.ACLGroup != nil {
			hasEnabled = true
			break
		}
	}
	if !hasEnabled {
		return ""
	}

	denyRules, bypassRules, _ := collectUnionIPRules(assignments)

	var config strings.Builder

	// Generate deny matcher if there are deny rules
	if len(denyRules) > 0 {
		config.WriteString("\t@denied_ips {\n")
		for _, cidr := range denyRules {
			config.WriteString(fmt.Sprintf("\t\tremote_ip %s\n", cidr))
		}
		config.WriteString("\t}\n")
		config.WriteString("\trespond @denied_ips 403\n\n")
	}

	// Generate bypass matcher if there are bypass rules
	if len(bypassRules) > 0 {
		config.WriteString("\t@bypass_ips {\n")
		for _, cidr := range bypassRules {
			config.WriteString(fmt.Sprintf("\t\tremote_ip %s\n", cidr))
		}
		config.WriteString("\t}\n\n")

		// Generate needs_auth matcher (not in bypass list)
		config.WriteString("\t@needs_auth {\n")
		config.WriteString("\t\tnot {\n")
		for _, cidr := range bypassRules {
			config.WriteString(fmt.Sprintf("\t\t\tremote_ip %s\n", cidr))
		}
		config.WriteString("\t\t}\n")
		config.WriteString("\t}\n\n")

		// Forward auth only for IPs that need it
		config.WriteString(fmt.Sprintf("\tforward_auth @needs_auth %s {\n", b.waygatesVerifyURL))
		config.WriteString("\t\turi /api/auth/acl/verify\n")
		config.WriteString("\t\tcopy_headers Remote-User Remote-Groups Remote-Email X-Forwarded-User\n")
		config.WriteString("\t}\n")
	} else {
		// No bypass rules, forward auth for all requests
		config.WriteString(fmt.Sprintf("\tforward_auth %s {\n", b.waygatesVerifyURL))
		config.WriteString("\t\turi /api/auth/acl/verify\n")
		config.WriteString("\t\tcopy_headers Remote-User Remote-Groups Remote-Email X-Forwarded-User\n")
		config.WriteString("\t}\n")
	}

	return config.String()
}

// GetDefaultWaygatesHeaders returns the default headers copied from Waygates auth
func GetDefaultWaygatesHeaders() []string {
	return waygatesDefaultHeaders
}

// GetProviderDefaultHeaders returns the default headers for a provider type
func GetProviderDefaultHeaders(providerType string) []string {
	if headers, ok := providerDefaultHeaders[providerType]; ok {
		return headers
	}
	return nil
}
