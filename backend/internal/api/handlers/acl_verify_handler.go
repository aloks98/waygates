package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// ACLVerifyHandler handles ACL forward auth verification and login
type ACLVerifyHandler struct {
	aclService   service.ACLServiceInterface
	userRepo     repository.UserRepositoryInterface
	auditService service.AuditServiceInterface
}

// NewACLVerifyHandler creates a new ACL verify handler
func NewACLVerifyHandler(aclService service.ACLServiceInterface, userRepo repository.UserRepositoryInterface, auditService service.AuditServiceInterface) *ACLVerifyHandler {
	return &ACLVerifyHandler{
		aclService:   aclService,
		userRepo:     userRepo,
		auditService: auditService,
	}
}

// ACL session cookie name
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
	UserID        int    `json:"user_id,omitempty"`
	Username      string `json:"username,omitempty"`
	Email         string `json:"email,omitempty"`
	ExpiresAt     string `json:"expires_at,omitempty"`
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
		utils.Unauthorized(w, "Invalid email or password")
		return
	}

	// Check password
	if !user.CheckPassword(req.Password) {
		// Log failed login attempt
		if h.auditService != nil {
			_ = h.auditService.LogLoginFailed(r.Context(), req.Email, clientIP, userAgent, "Invalid password")
		}
		utils.Unauthorized(w, "Invalid email or password")
		return
	}

	// Create ACL session
	// Default session TTL of 24 hours (86400 seconds)
	session, err := h.aclService.CreateSession(user.ID, nil, clientIP, userAgent, 86400)
	if err != nil {
		utils.InternalError(w, "Failed to create session")
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     ACLSessionCookieName,
		Value:    session.SessionToken,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	// Determine redirect URL
	redirectURL := req.Redirect
	if redirectURL == "" {
		redirectURL = "/"
	}

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

	// Revoke the session
	if err := h.aclService.RevokeSession(cookie.Value); err != nil {
		// Log error but don't fail - the cookie will be cleared anyway
		// The session might already be expired or revoked
	}

	// Clear the session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     ACLSessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
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
		// Clear the invalid cookie
		http.SetCookie(w, &http.Cookie{
			Name:     ACLSessionCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   true,
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
