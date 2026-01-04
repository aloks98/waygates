package caddyfile

import (
	"fmt"
	"strings"

	"github.com/aloks98/waygates/backend/internal/models"
)

// buildReverseProxyBlock generates a reverse proxy site block
func (b *Builder) buildReverseProxyBlock(proxy *models.Proxy) (string, error) {
	if proxy.Upstreams == nil {
		return "", fmt.Errorf("reverse proxy requires at least one upstream")
	}

	// Parse upstreams from interface{}
	upstreams, ok := proxy.Upstreams.([]interface{})
	if !ok || len(upstreams) == 0 {
		return "", fmt.Errorf("reverse proxy requires at least one upstream")
	}

	var sb strings.Builder

	// Site block with hostname
	sb.WriteString(fmt.Sprintf("%s {\n", proxy.Hostname))

	// Import security snippet if block exploits is enabled
	if proxy.BlockExploits {
		sb.WriteString("\timport /etc/caddy/snippets/security.caddy\n\n")
	}

	// Build upstream list
	upstreamAddrs, hasHTTPS := b.buildUpstreamList(upstreams)

	// reverse_proxy directive
	sb.WriteString(fmt.Sprintf("\treverse_proxy %s {\n", upstreamAddrs))

	// Load balancing config
	if len(proxy.LoadBalancing) > 0 {
		b.writeLoadBalancingConfig(&sb, proxy.LoadBalancing)
	}

	// Health checks
	if len(proxy.LoadBalancing) > 0 {
		if healthChecks, ok := proxy.LoadBalancing["health_checks"].(map[string]interface{}); ok {
			if enabled, _ := healthChecks["enabled"].(bool); enabled {
				b.writeHealthCheckConfig(&sb, healthChecks)
			}
		}
	}

	// Transport config for HTTPS upstreams
	if hasHTTPS || proxy.TLSInsecureSkipVerify {
		b.writeTransportConfig(&sb, hasHTTPS, proxy.TLSInsecureSkipVerify)
	}

	// Standard headers
	sb.WriteString("\t\theader_up X-Real-IP {remote_host}\n")
	sb.WriteString("\t\theader_up X-Forwarded-For {remote_host}\n")
	sb.WriteString("\t\theader_up X-Forwarded-Proto {scheme}\n")
	sb.WriteString("\t\theader_up X-Forwarded-Host {host}\n")

	// Custom headers
	if len(proxy.CustomHeaders) > 0 {
		for key, value := range proxy.CustomHeaders {
			if strVal, ok := value.(string); ok {
				sb.WriteString(fmt.Sprintf("\t\theader_up %s %q\n", key, strVal))
			}
		}
	}

	sb.WriteString("\t}\n") // Close reverse_proxy
	sb.WriteString("}\n")   // Close site block

	return sb.String(), nil
}

// buildUpstreamList creates the upstream address list from interface{}
func (b *Builder) buildUpstreamList(upstreams []interface{}) (string, bool) {
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

	return strings.Join(addresses, " "), hasHTTPS
}

// writeLoadBalancingConfig writes load balancing configuration
func (b *Builder) writeLoadBalancingConfig(sb *strings.Builder, lb models.JSONField) {
	if strategy, ok := lb["strategy"].(string); ok && strategy != "" {
		policy := mapLBStrategy(strategy)
		fmt.Fprintf(sb, "\t\tlb_policy %s\n", policy)
	}
}

// writeHealthCheckConfig writes health check configuration
func (b *Builder) writeHealthCheckConfig(sb *strings.Builder, hc map[string]interface{}) {
	if path, ok := hc["path"].(string); ok && path != "" {
		fmt.Fprintf(sb, "\t\thealth_uri %s\n", path)
	}
	if interval, ok := hc["interval"].(string); ok && interval != "" {
		fmt.Fprintf(sb, "\t\thealth_interval %s\n", interval)
	}
	if timeout, ok := hc["timeout"].(string); ok && timeout != "" {
		fmt.Fprintf(sb, "\t\thealth_timeout %s\n", timeout)
	}
}

// writeTransportConfig writes HTTPS transport configuration
func (b *Builder) writeTransportConfig(sb *strings.Builder, hasHTTPS, insecureSkipVerify bool) {
	sb.WriteString("\t\ttransport http {\n")

	if hasHTTPS {
		sb.WriteString("\t\t\ttls\n")
	}

	if insecureSkipVerify {
		sb.WriteString("\t\t\ttls_insecure_skip_verify\n")
	}

	sb.WriteString("\t\t}\n")
}

// mapLBStrategy maps our strategy names to Caddy's lb_policy names
func mapLBStrategy(strategy string) string {
	switch strategy {
	case "round_robin":
		return "round_robin"
	case "least_conn":
		return "least_conn"
	case "random":
		return "random"
	case "first":
		return "first"
	case "ip_hash":
		return "ip_hash"
	case "uri_hash":
		return "uri_hash"
	case "header":
		return "header"
	default:
		return "round_robin"
	}
}
