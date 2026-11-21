// Package templates contains the HTML pages embedded in the Waygates binary.
package templates

import _ "embed"

// NotFoundPage is the branded page served for unmatched proxy hostnames.
//
//go:embed 404.html
var NotFoundPage string
