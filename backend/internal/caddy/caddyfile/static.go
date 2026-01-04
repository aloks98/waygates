package caddyfile

import (
	"fmt"
	"strings"

	"github.com/aloks98/waygates/backend/internal/models"
)

// buildStaticBlock generates a static file server site block
func (b *Builder) buildStaticBlock(proxy *models.Proxy) (string, error) {
	if len(proxy.StaticConfig) == 0 {
		return "", fmt.Errorf("static proxy requires static configuration")
	}

	rootPath, _ := proxy.StaticConfig["root_path"].(string)
	if rootPath == "" {
		return "", fmt.Errorf("static proxy requires root_path")
	}

	var sb strings.Builder

	// Site block with hostname (use http:// prefix if SSL disabled)
	siteAddr := formatSiteAddress(proxy.Hostname, proxy.SSLEnabled)
	sb.WriteString(fmt.Sprintf("%s {\n", siteAddr))

	// Import security snippet if block exploits is enabled
	if proxy.BlockExploits {
		sb.WriteString("\timport /etc/caddy/snippets/security.caddy\n\n")
	}

	// Root directive
	sb.WriteString(fmt.Sprintf("\troot * %s\n", rootPath))

	// Template rendering (must come before file_server)
	if templateRendering, ok := proxy.StaticConfig["template_rendering"].(bool); ok && templateRendering {
		sb.WriteString("\ttemplates\n")
	}

	// SPA support with try_files
	if tryFiles, ok := proxy.StaticConfig["try_files"].([]interface{}); ok && len(tryFiles) > 0 {
		var files []string
		for _, f := range tryFiles {
			if s, ok := f.(string); ok {
				files = append(files, s)
			}
		}
		if len(files) > 0 {
			sb.WriteString(fmt.Sprintf("\ttry_files {path} %s\n", strings.Join(files, " ")))
		}
	}

	// File server directive
	sb.WriteString("\tfile_server")

	// Add browse option if directory listing is enabled
	if browse, ok := proxy.StaticConfig["browse"].(bool); ok && browse {
		sb.WriteString(" browse")
	}

	sb.WriteString(" {\n")

	// Index file
	if indexFile, ok := proxy.StaticConfig["index_file"].(string); ok && indexFile != "" {
		sb.WriteString(fmt.Sprintf("\t\tindex %s\n", indexFile))
	}

	sb.WriteString("\t}\n") // Close file_server
	sb.WriteString("}\n")   // Close site block

	return sb.String(), nil
}
