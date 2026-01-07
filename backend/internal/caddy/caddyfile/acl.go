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
	waygatesVerifyURL string // e.g., http://localhost:8080/api/auth/acl/verify
}

// NewACLBuilder creates a new ACL builder with the specified Waygates verify URL.
func NewACLBuilder(waygatesVerifyURL string) *ACLBuilder {
	return &ACLBuilder{
		waygatesVerifyURL: waygatesVerifyURL,
	}
}

// Default headers to copy from Waygates forward auth responses
var waygatesDefaultHeaders = []string{
	"X-Auth-User",
	"X-Auth-User-ID",
	"X-Auth-User-Email",
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

	// 4. Handle remaining requests with authentication
	hasForwardAuth := group.WaygatesAuth != nil && group.WaygatesAuth.Enabled
	hasExternalAuth := len(group.ExternalProviders) > 0
	hasBasicAuth := len(group.BasicAuthUsers) > 0

	if hasBasicAuth && !hasForwardAuth && !hasExternalAuth {
		// Only basic auth configured
		sb.WriteString(b.buildBasicAuthBlock(proxy, group, pathPattern, matcherPrefix, bypassIPs, allowIPs))
	} else if hasForwardAuth || hasExternalAuth {
		// Forward auth (Waygates or external provider)
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
	hasExternalAuth := len(group.ExternalProviders) > 0
	hasBasicAuth := len(group.BasicAuthUsers) > 0

	if hasBasicAuth && !hasForwardAuth && !hasExternalAuth {
		sb.WriteString(b.buildBasicAuthBlockAll(proxy, group, pathPattern, matcherPrefix, allAllowedIPs))
	} else if hasForwardAuth || hasExternalAuth {
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

// buildBasicAuthBlock generates basic auth configuration
func (b *ACLBuilder) buildBasicAuthBlock(proxy *models.Proxy, group *models.ACLGroup, pathPattern, matcherPrefix string, bypassIPs, allowIPs []string) string {
	var sb strings.Builder

	matcherName := fmt.Sprintf("@%s_basic_auth", matcherPrefix)

	// Build matcher that excludes already handled IPs
	sb.WriteString(fmt.Sprintf("\t%s {\n", matcherName))
	if pathPattern != "" && pathPattern != "/*" {
		sb.WriteString(fmt.Sprintf("\t\tpath %s\n", pathPattern))
	}

	// Exclude bypass and allow IPs
	excludeIPs := make([]string, 0, len(bypassIPs)+len(allowIPs))
	excludeIPs = append(excludeIPs, bypassIPs...)
	excludeIPs = append(excludeIPs, allowIPs...)
	if len(excludeIPs) > 0 {
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

	sb.WriteString(fmt.Sprintf("\t%s {\n", matcherName))
	if pathPattern != "" && pathPattern != "/*" {
		sb.WriteString(fmt.Sprintf("\t\tpath %s\n", pathPattern))
	}
	if len(allowedIPs) > 0 {
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

	// Build matcher that excludes already handled IPs
	sb.WriteString(fmt.Sprintf("\t%s {\n", matcherName))
	if pathPattern != "" && pathPattern != "/*" {
		sb.WriteString(fmt.Sprintf("\t\tpath %s\n", pathPattern))
	}

	// Exclude bypass and allow IPs
	excludeIPs := make([]string, 0, len(bypassIPs)+len(allowIPs))
	excludeIPs = append(excludeIPs, bypassIPs...)
	excludeIPs = append(excludeIPs, allowIPs...)
	if len(excludeIPs) > 0 {
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

	sb.WriteString(fmt.Sprintf("\t%s {\n", matcherName))
	if pathPattern != "" && pathPattern != "/*" {
		sb.WriteString(fmt.Sprintf("\t\tpath %s\n", pathPattern))
	}
	if len(allowedIPs) > 0 {
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
