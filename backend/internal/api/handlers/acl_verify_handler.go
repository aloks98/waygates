package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/net/publicsuffix"

	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// ACLVerifyHandler handles ACL forward auth verification and login
type ACLVerifyHandler struct {
	aclService   service.ACLServiceInterface
	userRepo     repository.UserRepositoryInterface
	auditService service.AuditServiceInterface
	logger       *zap.Logger
	cookieSecure bool
}

// ACLVerifyHandlerConfig holds configuration for ACL verify handler
type ACLVerifyHandlerConfig struct {
	ACLService   service.ACLServiceInterface
	UserRepo     repository.UserRepositoryInterface
	AuditService service.AuditServiceInterface
	Logger       *zap.Logger
	CookieSecure bool // Whether cookies should be secure-only
}

// NewACLVerifyHandler creates a new ACL verify handler
func NewACLVerifyHandler(cfg ACLVerifyHandlerConfig) *ACLVerifyHandler {
	return &ACLVerifyHandler{
		aclService:   cfg.ACLService,
		userRepo:     cfg.UserRepo,
		auditService: cfg.AuditService,
		logger:       cfg.Logger,
		cookieSecure: cfg.CookieSecure,
	}
}

// ACLSessionCookieName is the name of the cookie used for ACL sessions
const ACLSessionCookieName = "waygates_acl_session"

// =============================================================================
// Request/Response Types
// =============================================================================

// ACLLoginRequest is the request body for ACL login
type ACLLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Redirect string `json:"redirect,omitempty"`
}

// ACLLoginResponse is the response body for ACL login
type ACLLoginResponse struct {
	Success     bool   `json:"success"`
	RedirectURL string `json:"redirect_url,omitempty"`
	Message     string `json:"message,omitempty"`
}

// ACLSessionResponse is the response body for session info
type ACLSessionResponse struct {
	Authenticated bool   `json:"authenticated"`
	UserID        *int   `json:"user_id,omitempty"`
	Username      string `json:"username,omitempty"`
	Email         string `json:"email,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	// OAuth user info (for OAuth-only sessions)
	OAuthEmail    string `json:"oauth_email,omitempty"`
	OAuthProvider string `json:"oauth_provider,omitempty"`
}

// =============================================================================
// Verify Handler (Forward Auth)
// =============================================================================

// Verify handles GET /api/auth/acl/verify
// This endpoint is called by Caddy's forward_auth directive to verify access
func (h *ACLVerifyHandler) Verify(w http.ResponseWriter, r *http.Request) {
	// Extract request information from Caddy forward_auth headers
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}

	uri := r.Header.Get("X-Forwarded-Uri")
	if uri == "" {
		uri = r.URL.Path
	}

	method := r.Header.Get("X-Forwarded-Method")
	if method == "" {
		method = r.Method
	}

	// Get client IP from X-Forwarded-For or fall back to remote address
	remoteIP := r.Header.Get("X-Forwarded-For")
	if remoteIP == "" {
		remoteIP = getClientIP(r)
	} else {
		// X-Forwarded-For can contain multiple IPs, take the first one (original client)
		parts := strings.Split(remoteIP, ",")
		remoteIP = strings.TrimSpace(parts[0])
	}

	// Extract session token from cookie
	var sessionToken string
	if cookie, err := r.Cookie(ACLSessionCookieName); err == nil {
		sessionToken = cookie.Value
	}

	// Extract basic auth credentials if present
	var basicAuth *service.BasicAuthCredentials
	if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Basic ") {
		if creds := parseBasicAuth(authHeader); creds != nil {
			basicAuth = creds
		}
	}

	// Build verify request
	verifyReq := &service.ACLVerifyRequest{
		Host:         host,
		Path:         uri,
		Method:       method,
		RemoteIP:     remoteIP,
		SessionToken: sessionToken,
		BasicAuth:    basicAuth,
	}

	// Verify access
	response, err := h.aclService.VerifyAccess(verifyReq)
	if err != nil {
		// Internal error - return 500
		if h.logger != nil {
			h.logger.Error("Failed to verify ACL access",
				zap.String("host", host),
				zap.String("uri", uri),
				zap.String("method", method),
				zap.String("remote_ip", remoteIP),
				zap.Error(err))
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if response.Allowed {
		// Access granted - return 200 with auth headers
		for key, value := range response.Headers {
			w.Header().Set(key, value)
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	// Access denied
	if response.RequiresAuth {
		// Authentication required but not provided or invalid
		// Return 401 with redirect header for the auth page
		if response.RedirectURL != "" {
			w.Header().Set("X-Auth-Redirect", response.RedirectURL)
		}
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Access explicitly denied (e.g., IP blocked)
	w.WriteHeader(http.StatusForbidden)
}

// =============================================================================
// Login Handler
// =============================================================================

// Login handles POST /api/auth/acl/login
// This endpoint handles ACL login using Waygates credentials
func (h *ACLVerifyHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req ACLLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body format", nil)
		return
	}

	// Validate required fields
	if req.Email == "" {
		utils.BadRequest(w, "email is required", nil)
		return
	}
	if req.Password == "" {
		utils.BadRequest(w, "password is required", nil)
		return
	}

	// Get client info for session
	clientIP := getClientIP(r)
	userAgent := r.UserAgent()

	// Look up user by email or username
	user, err := h.userRepo.GetByUsernameOrEmail(req.Email)
	if err != nil {
		// Log failed login attempt
		if h.auditService != nil {
			_ = h.auditService.LogLoginFailed(r.Context(), req.Email, clientIP, userAgent, "User not found")
		}
		if h.logger != nil {
			h.logger.Info("ACL login failed: user not found",
				zap.String("email", req.Email),
				zap.String("client_ip", clientIP))
		}
		utils.Unauthorized(w, "Invalid email or password")
		return
	}

	// Check password
	if !user.CheckPassword(req.Password) {
		// Log failed login attempt
		if h.auditService != nil {
			_ = h.auditService.LogLoginFailed(r.Context(), req.Email, clientIP, userAgent, "Invalid password")
		}
		if h.logger != nil {
			h.logger.Info("ACL login failed: invalid password",
				zap.String("email", req.Email),
				zap.String("client_ip", clientIP))
		}
		utils.Unauthorized(w, "Invalid email or password")
		return
	}

	// Create ACL session
	// Default session TTL of 24 hours (86400 seconds)
	session, err := h.aclService.CreateSession(user.ID, nil, clientIP, userAgent, 86400)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("Failed to create ACL session",
				zap.Int("user_id", user.ID),
				zap.String("client_ip", clientIP),
				zap.Error(err))
		}
		utils.InternalError(w, "Failed to create session")
		return
	}

	// Determine redirect URL
	redirectURL := req.Redirect
	if redirectURL == "" {
		redirectURL = "/"
	}

	// Extract cookie domain dynamically from the redirect URL
	// This allows the cookie to work across different domains (e.g., e412.in vs example.com)
	cookieDomain := extractCookieDomain(redirectURL)
	if h.logger != nil {
		h.logger.Debug("Setting ACL session cookie",
			zap.String("redirect_url", redirectURL),
			zap.String("cookie_domain", cookieDomain))
	}

	// Set session cookie with domain for cross-subdomain support
	http.SetCookie(w, &http.Cookie{
		Name:     ACLSessionCookieName,
		Value:    session.SessionToken,
		Path:     "/",
		Domain:   cookieDomain,
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	utils.Success(w, ACLLoginResponse{
		Success:     true,
		RedirectURL: redirectURL,
		Message:     "Login successful",
	}, "Login successful")
}

// =============================================================================
// Logout Handler
// =============================================================================

// Logout handles POST /api/auth/acl/logout
// This endpoint invalidates the current ACL session
func (h *ACLVerifyHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get session token from cookie
	cookie, err := r.Cookie(ACLSessionCookieName)
	if err != nil {
		// No session cookie - already logged out
		utils.Success(w, nil, "Already logged out")
		return
	}

	// Revoke the session - ignore errors as the session might already be expired or revoked
	// The cookie will be cleared anyway
	_ = h.aclService.RevokeSession(cookie.Value)

	// Extract cookie domain from X-Forwarded-Host (if coming through Caddy) or request host
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	cookieDomain := extractCookieDomainFromHost(host)

	// Clear the session cookie (must use same domain as when it was set)
	http.SetCookie(w, &http.Cookie{
		Name:     ACLSessionCookieName,
		Value:    "",
		Path:     "/",
		Domain:   cookieDomain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	utils.Success(w, nil, "Logged out successfully")
}

// =============================================================================
// Session Handler
// =============================================================================

// GetSession handles GET /api/auth/acl/session
// This endpoint returns information about the current session
func (h *ACLVerifyHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	// Get session token from cookie
	cookie, err := r.Cookie(ACLSessionCookieName)
	if err != nil {
		// No session cookie
		utils.Success(w, ACLSessionResponse{
			Authenticated: false,
		}, "")
		return
	}

	// Validate session
	session, err := h.aclService.ValidateSession(cookie.Value)
	if err != nil {
		// Session invalid or expired
		// Extract cookie domain from request host
		host := r.Header.Get("X-Forwarded-Host")
		if host == "" {
			host = r.Host
		}
		cookieDomain := extractCookieDomainFromHost(host)

		// Clear the invalid cookie
		http.SetCookie(w, &http.Cookie{
			Name:     ACLSessionCookieName,
			Value:    "",
			Path:     "/",
			Domain:   cookieDomain,
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   h.cookieSecure,
			SameSite: http.SameSiteLaxMode,
		})

		if errors.Is(err, service.ErrSessionExpired) {
			utils.Success(w, ACLSessionResponse{
				Authenticated: false,
			}, "Session expired")
			return
		}

		utils.Success(w, ACLSessionResponse{
			Authenticated: false,
		}, "")
		return
	}

	// Build response
	response := ACLSessionResponse{
		Authenticated: true,
		UserID:        session.UserID,
		ExpiresAt:     session.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Include user details if available
	if session.User != nil {
		response.Username = session.User.Username
		response.Email = session.User.Email
	}

	// Include OAuth info if available
	if session.Email != nil {
		response.OAuthEmail = *session.Email
	}
	if session.Provider != nil {
		response.OAuthProvider = *session.Provider
	}

	utils.Success(w, response, "")
}

// =============================================================================
// Helper Functions
// =============================================================================

// parseBasicAuth parses a Basic auth header and returns the credentials
func parseBasicAuth(authHeader string) *service.BasicAuthCredentials {
	// Remove "Basic " prefix
	encoded := strings.TrimPrefix(authHeader, "Basic ")
	if encoded == authHeader {
		return nil
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil
	}

	// Split on colon
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return nil
	}

	return &service.BasicAuthCredentials{
		Username: parts[0],
		Password: parts[1],
	}
}

// extractCookieDomain extracts the base domain from a URL string for cookie setting.
// For example:
//   - "https://nginx.e412.in/path" -> ".e412.in"
//   - "https://app.internal.company.com" -> ".internal.company.com"
//   - "https://localhost:8080" -> "" (no domain for localhost)
//
// This allows cookies to be shared across subdomains of the same base domain.
func extractCookieDomain(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	host := parsed.Hostname()
	if host == "" {
		return ""
	}

	// Don't set domain for localhost or IP addresses
	if host == "localhost" || isIPAddress(host) {
		return ""
	}

	// Use publicsuffix to get the eTLD+1 (effective TLD plus one more label)
	// This handles cases like:
	// - "nginx.e412.in" -> "e412.in"
	// - "app.internal.company.com" -> "company.com" (if .com is the eTLD)
	// - "app.internal.co.uk" -> "internal.co.uk" (because .co.uk is the eTLD)
	baseDomain, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		// Fallback: just use the host as-is (for internal domains not in public suffix list)
		// Try to extract a reasonable base domain
		return extractBaseDomainFallback(host)
	}

	// If the host is the same as the base domain (e.g., "e412.in"), use it directly
	// Otherwise, prefix with dot for subdomain sharing
	if host == baseDomain {
		return "." + baseDomain
	}

	return "." + baseDomain
}

// extractBaseDomainFallback handles domains not in the public suffix list
// by finding a reasonable base domain (last 2 segments for simple TLDs)
func extractBaseDomainFallback(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return ""
	}

	// For simple domains like "internal.mycompany.local", use the last 2 parts
	// This won't work for complex TLDs like .co.uk but covers internal use cases
	if len(parts) >= 2 {
		baseDomain := strings.Join(parts[len(parts)-2:], ".")
		return "." + baseDomain
	}

	return ""
}

// isIPAddress checks if the host is an IP address (v4 or v6)
func isIPAddress(host string) bool {
	// Simple check: contains only digits and dots (IPv4) or contains colon (IPv6)
	if strings.Contains(host, ":") {
		return true // IPv6
	}

	// Check for IPv4
	for _, c := range host {
		if c != '.' && (c < '0' || c > '9') {
			return false
		}
	}
	return len(host) > 0
}

// extractCookieDomainFromHost extracts cookie domain from a bare host string
func extractCookieDomainFromHost(host string) string {
	// Strip port if present
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		// Check if this is IPv6 (would have multiple colons or brackets)
		if !strings.Contains(host, "[") && strings.Count(host, ":") == 1 {
			host = host[:idx]
		}
	}

	// Don't set domain for localhost or IP addresses
	if host == "localhost" || isIPAddress(host) {
		return ""
	}

	// Use publicsuffix to get the eTLD+1
	baseDomain, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return extractBaseDomainFallback(host)
	}

	return "." + baseDomain
}
