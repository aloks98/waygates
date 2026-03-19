package config

// SecurityRoutes returns predefined security routes that block common exploits.
// These routes should be added BEFORE the main proxy routes so they can intercept
// malicious requests before they reach the backend.
//
// The rules block:
// - SQL injection attempts
// - File injection/path traversal
// - XSS (Cross-Site Scripting) attacks
// - PHP globals injection
// - Malicious user agents
// - Common vulnerability scanner paths
func SecurityRoutes() []*HTTPRoute {
	return []*HTTPRoute{
		// Block SQL injection attempts in URI
		{
			Match: []MatcherSet{
				{
					"path_regexp": map[string]string{
						"name":    "sql_injection",
						"pattern": `(?i)(union.*select|select.*from|insert.*into|delete.*from|drop.*table|update.*set)`,
					},
				},
			},
			Handle: []HTTPHandler{
				{
					"handler":     "static_response",
					"status_code": 403,
					"body":        "Forbidden - SQL injection detected",
					"close":       true,
				},
			},
			Terminal: true,
		},
		// Block file injection/traversal attempts
		{
			Match: []MatcherSet{
				{
					"path_regexp": map[string]string{
						"name":    "file_injection",
						"pattern": `(\.\./|\.\.\\|%2e%2e|%252e)`,
					},
				},
			},
			Handle: []HTTPHandler{
				{
					"handler":     "static_response",
					"status_code": 403,
					"body":        "Forbidden - Path traversal detected",
					"close":       true,
				},
			},
			Terminal: true,
		},
		// Block common XSS patterns
		{
			Match: []MatcherSet{
				{
					"path_regexp": map[string]string{
						"name":    "xss_attack",
						"pattern": `(?i)(<script|%3cscript|javascript:|vbscript:|onload=|onerror=)`,
					},
				},
			},
			Handle: []HTTPHandler{
				{
					"handler":     "static_response",
					"status_code": 403,
					"body":        "Forbidden - XSS detected",
					"close":       true,
				},
			},
			Terminal: true,
		},
		// Block PHP globals injection
		{
			Match: []MatcherSet{
				{
					"path_regexp": map[string]string{
						"name":    "php_globals",
						"pattern": `(?i)(globals\[|_request\[|_get\[|_post\[|_cookie\[|_files\[|_env\[|_server\[)`,
					},
				},
			},
			Handle: []HTTPHandler{
				{
					"handler":     "static_response",
					"status_code": 403,
					"body":        "Forbidden - PHP globals injection detected",
					"close":       true,
				},
			},
			Terminal: true,
		},
		// Block malicious user agents
		{
			Match: []MatcherSet{
				{
					"header_regexp": map[string]interface{}{
						"User-Agent": map[string]string{
							"name":    "bad_user_agent",
							"pattern": `(?i)(libwww-perl|scrapy|nikto|sqlmap|nmap|masscan)`,
						},
					},
				},
			},
			Handle: []HTTPHandler{
				{
					"handler":     "static_response",
					"status_code": 403,
					"body":        "Forbidden - Blocked user agent",
					"close":       true,
				},
			},
			Terminal: true,
		},
		// Block common vulnerability scanners
		{
			Match: []MatcherSet{
				{
					"path_regexp": map[string]string{
						"name":    "vuln_scanners",
						"pattern": `(?i)(wp-admin|wp-login|xmlrpc\.php|\.env|\.git|\.svn|\.htaccess|\.htpasswd|web\.config|phpinfo|phpmyadmin|adminer)`,
					},
				},
			},
			Handle: []HTTPHandler{
				{
					"handler":     "static_response",
					"status_code": 403,
					"body":        "Forbidden - Access denied",
					"close":       true,
				},
			},
			Terminal: true,
		},
	}
}

// SecurityRoutesForHost returns security routes scoped to a specific hostname.
// This is useful when you want to apply security rules only to certain hosts.
func SecurityRoutesForHost(hostname string) []*HTTPRoute {
	routes := SecurityRoutes()
	for _, route := range routes {
		// Add host matcher to each route's matcher set
		for i := range route.Match {
			route.Match[i]["host"] = []string{hostname}
		}
	}
	return routes
}
