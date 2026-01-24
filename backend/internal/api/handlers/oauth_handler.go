package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/auth"
	"github.com/aloks98/waygates/backend/internal/config"
	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// OAuthHandler handles OAuth authentication flows
type OAuthHandler struct {
	providerManager *auth.OAuthProviderManager
	aclService      service.ACLServiceInterface
	userRepo        repository.UserRepositoryInterface
	proxyRepo       repository.ProxyRepositoryInterface
	auditService    service.AuditServiceInterface
	config          *config.Config
	logger          *zap.Logger
}

// OAuthHandlerConfig holds configuration for creating an OAuthHandler
type OAuthHandlerConfig struct {
	ProviderManager *auth.OAuthProviderManager
	ACLService      service.ACLServiceInterface
	UserRepo        repository.UserRepositoryInterface
	ProxyRepo       repository.ProxyRepositoryInterface
	AuditService    service.AuditServiceInterface
	Config          *config.Config
	Logger          *zap.Logger
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(cfg OAuthHandlerConfig) *OAuthHandler {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &OAuthHandler{
		providerManager: cfg.ProviderManager,
		aclService:      cfg.ACLService,
		userRepo:        cfg.UserRepo,
		proxyRepo:       cfg.ProxyRepo,
		auditService:    cfg.AuditService,
		config:          cfg.Config,
		logger:          logger.Named("oauth-handler"),
	}
}

// OAuth state cookie name
const oauthStateCookieName = "waygates_oauth_state"

// PKCE verifier cookie name
const oauthPKCECookieName = "waygates_oauth_pkce"

// State expiration time
const stateExpiration = 10 * time.Minute

// =============================================================================
// Response Types
// =============================================================================

// OAuthProvidersResponse is the response for listing OAuth providers
type OAuthProvidersResponse struct {
	Providers []auth.OAuthProviderPublic `json:"providers"`
}

// =============================================================================
// List Providers Handler (Public)
// =============================================================================

// ListProviders handles GET /api/auth/oauth/providers
// Returns a list of OAuth providers that are available (env vars configured)
// This is a public endpoint used by the login page to show available OAuth options
func (h *OAuthHandler) ListProviders(w http.ResponseWriter, _ *http.Request) {
	// Get all available providers (env vars configured)
	providers := h.providerManager.GetAvailableProvidersPublic()

	// Mark all available providers as enabled (availability = can be used)
	for i := range providers {
		providers[i].Enabled = providers[i].Available
	}

	// Return array directly for frontend compatibility
	utils.Success(w, providers, "")
}

// =============================================================================
// Start OAuth Handler
// =============================================================================

// StartOAuth handles GET /auth/oauth/{provider}
// Initiates the OAuth flow by redirecting to the provider's authorization URL
func (h *OAuthHandler) StartOAuth(w http.ResponseWriter, r *http.Request) {
	providerID := chi.URLParam(r, "provider")

	// Validate provider ID
	if !auth.ValidateProviderID(providerID) {
		http.Error(w, "Invalid OAuth provider", http.StatusBadRequest)
		return
	}

	// Get provider configuration
	provider, ok := h.providerManager.GetProviderByString(providerID)
	if !ok || !provider.Enabled {
		http.Error(w, "OAuth provider not configured or disabled", http.StatusBadRequest)
		return
	}

	// Get redirect URL from query parameter and validate it
	rawRedirect := r.URL.Query().Get("redirect")
	h.logger.Debug("StartOAuth received redirect parameter",
		zap.String("provider", providerID),
		zap.String("raw_redirect", rawRedirect))
	redirectURL := h.validateRedirectURL(rawRedirect)
	h.logger.Debug("StartOAuth validated redirect URL",
		zap.String("validated_redirect", redirectURL))

	// Generate state parameter with redirect URL encoded
	state, err := h.generateState(redirectURL)
	if err != nil {
		h.logger.Error("Failed to generate OAuth state", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Store state in cookie
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   int(stateExpiration.Seconds()),
		HttpOnly: true,
		Secure:   h.config.ACL.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	// Generate PKCE code verifier and challenge
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		h.logger.Error("Failed to generate PKCE code verifier", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	// Store PKCE verifier in cookie
	http.SetCookie(w, &http.Cookie{
		Name:     oauthPKCECookieName,
		Value:    codeVerifier,
		Path:     "/",
		MaxAge:   int(stateExpiration.Seconds()),
		HttpOnly: true,
		Secure:   h.config.ACL.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	// Build OAuth2 config
	oauth2Config := h.buildOAuth2Config(provider)

	// Generate authorization URL with PKCE
	authURL := oauth2Config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	// Redirect to provider
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// =============================================================================
// OAuth Callback Handler
// =============================================================================

// Callback handles GET /auth/oauth/{provider}/callback
// Handles the OAuth callback from the provider
func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	providerID := chi.URLParam(r, "provider")

	// Validate provider ID
	if !auth.ValidateProviderID(providerID) {
		h.handleOAuthError(w, r, "Invalid OAuth provider", "")
		return
	}

	// Get provider configuration
	provider, ok := h.providerManager.GetProviderByString(providerID)
	if !ok || !provider.Enabled {
		h.handleOAuthError(w, r, "OAuth provider not configured or disabled", "")
		return
	}

	// Check for OAuth error from provider
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		h.logger.Warn("OAuth provider returned error",
			zap.String("provider", providerID),
			zap.String("error", errParam),
			zap.String("description", errDesc))
		h.handleOAuthError(w, r, "Authentication failed: "+errParam, "")
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		h.handleOAuthError(w, r, "Missing authorization code", "")
		return
	}

	// Validate state parameter
	stateParam := r.URL.Query().Get("state")
	stateCookie, err := r.Cookie(oauthStateCookieName)
	if err != nil || stateCookie.Value == "" {
		h.handleOAuthError(w, r, "Invalid or missing state", "")
		return
	}

	if stateParam != stateCookie.Value {
		h.handleOAuthError(w, r, "State mismatch - possible CSRF attack", "")
		return
	}

	// Extract redirect URL from state and validate it
	redirectURL := h.validateRedirectURL(h.extractRedirectFromState(stateParam))

	// Get PKCE verifier from cookie
	pkceCookie, err := r.Cookie(oauthPKCECookieName)
	if err != nil || pkceCookie.Value == "" {
		h.handleOAuthError(w, r, "Invalid or missing PKCE verifier", redirectURL)
		return
	}
	codeVerifier := pkceCookie.Value

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.config.ACL.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	// Clear PKCE cookie
	http.SetCookie(w, &http.Cookie{
		Name:     oauthPKCECookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.config.ACL.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	// Build OAuth2 config
	oauth2Config := h.buildOAuth2Config(provider)

	// Exchange code for token with PKCE verifier
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	token, err := oauth2Config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		h.logger.Error("Failed to exchange OAuth code",
			zap.String("provider", providerID),
			zap.Error(err))
		h.handleOAuthError(w, r, "Failed to authenticate with provider", redirectURL)
		return
	}

	// Fetch user info from provider
	userInfo, err := h.fetchUserInfo(ctx, provider, token)
	if err != nil {
		h.logger.Error("Failed to fetch user info",
			zap.String("provider", providerID),
			zap.Error(err))
		h.handleOAuthError(w, r, "Failed to get user information", redirectURL)
		return
	}

	// Find or create user
	user, err := h.findOrCreateUser(ctx, provider.ID, userInfo)
	if err != nil {
		h.logger.Error("Failed to find or create user",
			zap.String("provider", providerID),
			zap.Error(err))
		h.handleOAuthError(w, r, "Failed to process user account", redirectURL)
		return
	}

	// Create ACL session with OAuth info for proper restriction checks
	clientIP := getClientIP(r)
	userAgent := r.UserAgent()
	sessionTTL := int(h.config.ACL.SessionTTL.Seconds())
	if sessionTTL <= 0 {
		sessionTTL = 86400 // Default 24 hours
	}

	// Store both user ID and OAuth info in session
	// This allows OAuth restrictions to be checked even for users with Waygates accounts
	session, err := h.aclService.CreateSessionWithParams(service.CreateSessionParams{
		UserID:    &user.ID,
		ProxyID:   nil,
		IP:        clientIP,
		UserAgent: userAgent,
		TTL:       sessionTTL,
		Email:     userInfo.Email,
		Provider:  providerID,
	})
	if err != nil {
		h.logger.Error("Failed to create session",
			zap.Int("user_id", user.ID),
			zap.Error(err))
		h.handleOAuthError(w, r, "Failed to create session", redirectURL)
		return
	}

	// Log successful OAuth login
	if h.auditService != nil {
		_ = h.auditService.LogLogin(r.Context(), user.ID, user.Username+" (OAuth:"+providerID+")", clientIP, userAgent)
	}

	// Extract cookie domain dynamically from the redirect URL
	// This allows the cookie to work across different domains
	cookieDomain := extractCookieDomain(redirectURL)

	// Set session cookie
	cookie := &http.Cookie{
		Name:     ACLSessionCookieName,
		Value:    session.SessionToken,
		Path:     "/",
		Domain:   cookieDomain,
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   h.config.ACL.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, cookie)

	h.logger.Info("OAuth login successful",
		zap.String("provider", providerID),
		zap.Int("user_id", user.ID),
		zap.String("username", user.Username),
		zap.String("email", userInfo.Email),
		zap.String("cookie_domain", cookieDomain),
		zap.String("redirect_url", redirectURL),
		zap.String("session_token_prefix", session.SessionToken[:8]+"..."))

	// Redirect to original URL (already validated above)
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// =============================================================================
// Helper Functions
// =============================================================================

// validateRedirectURL validates that a redirect URL is safe to prevent open redirect attacks.
// Returns the validated URL or "/" if invalid.
func (h *OAuthHandler) validateRedirectURL(redirectURL string) string {
	if redirectURL == "" {
		return "/"
	}

	// Check for relative paths (must start with / but not //)
	// Protocol-relative URLs (//example.com) are dangerous
	if strings.HasPrefix(redirectURL, "/") && !strings.HasPrefix(redirectURL, "//") {
		return redirectURL
	}

	// Parse absolute URL
	parsed, err := url.Parse(redirectURL)
	if err != nil {
		return "/"
	}

	// Reject dangerous schemes
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "/"
	}

	// Validate against allowed callback base URL
	if h.config != nil && h.config.ACL.OAuth.CallbackBaseURL != "" {
		baseURL, err := url.Parse(h.config.ACL.OAuth.CallbackBaseURL)
		if err == nil && parsed.Host == baseURL.Host {
			return redirectURL
		}
	}

	// Check if the hostname is a configured proxy in Waygates
	// This allows OAuth redirects back to protected resources
	if h.proxyRepo != nil {
		hostname := parsed.Hostname()
		proxy, err := h.proxyRepo.GetByHostname(hostname)
		if err == nil && proxy != nil {
			h.logger.Info("Allowing redirect to configured proxy hostname",
				zap.String("hostname", hostname),
				zap.Int("proxy_id", proxy.ID))
			return redirectURL
		}
		// Log why proxy lookup failed
		h.logger.Warn("Proxy lookup failed for redirect hostname",
			zap.String("hostname", hostname),
			zap.Error(err))
	} else {
		h.logger.Warn("ProxyRepo is nil, cannot validate redirect hostname")
	}

	// For other absolute URLs, reject and return safe default
	h.logger.Warn("Rejecting redirect URL - not a configured proxy hostname",
		zap.String("redirect_url", redirectURL))
	return "/"
}

// generateState generates a secure state parameter with encoded redirect URL
func (h *OAuthHandler) generateState(redirectURL string) (string, error) {
	// Generate random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Encode: random_bytes|redirect_url
	randomPart := base64.URLEncoding.EncodeToString(bytes)
	encodedRedirect := base64.URLEncoding.EncodeToString([]byte(redirectURL))

	return randomPart + "|" + encodedRedirect, nil
}

// generateCodeVerifier generates a PKCE code verifier (43-128 chars, URL-safe)
func generateCodeVerifier() (string, error) {
	// Generate 32 random bytes (will become 43 chars when base64url encoded)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Use RawURLEncoding to avoid padding characters
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// generateCodeChallenge generates a PKCE code challenge from a verifier using S256
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// extractRedirectFromState extracts the redirect URL from the state parameter
func (h *OAuthHandler) extractRedirectFromState(state string) string {
	parts := strings.SplitN(state, "|", 2)
	if len(parts) != 2 {
		return "/"
	}

	decoded, err := base64.URLEncoding.DecodeString(parts[1])
	if err != nil {
		return "/"
	}

	return string(decoded)
}

// buildOAuth2Config builds an oauth2.Config for the given provider
func (h *OAuthHandler) buildOAuth2Config(provider *auth.OAuthProvider) *oauth2.Config {
	callbackURL := h.getCallbackURL(provider.ID)

	return &oauth2.Config{
		ClientID:     provider.ClientID,
		ClientSecret: provider.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  provider.AuthURL,
			TokenURL: provider.TokenURL,
		},
		RedirectURL: callbackURL,
		Scopes:      provider.Scopes,
	}
}

// getCallbackURL constructs the OAuth callback URL for a provider
func (h *OAuthHandler) getCallbackURL(providerID auth.OAuthProviderID) string {
	baseURL := h.config.ACL.OAuth.CallbackBaseURL
	if baseURL == "" {
		// Fallback - this should be configured in production
		baseURL = "http://localhost:8080"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return fmt.Sprintf("%s/auth/oauth/%s/callback", baseURL, providerID)
}

// fetchUserInfo fetches user information from the OAuth provider
func (h *OAuthHandler) fetchUserInfo(ctx context.Context, provider *auth.OAuthProvider, token *oauth2.Token) (*OAuthUserInfo, error) {
	// Create HTTP client with token
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	// Fetch user info
	req, err := http.NewRequestWithContext(ctx, "GET", provider.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// GitHub requires Accept header
	if provider.ID == auth.OAuthProviderGitHub {
		req.Header.Set("Accept", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching user info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("provider returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response based on provider
	userInfo, err := h.parseUserInfo(provider.ID, resp.Body)
	if err != nil {
		return nil, err
	}

	// If GitHub user has no public email, fetch from /user/emails endpoint
	if provider.ID == auth.OAuthProviderGitHub && userInfo.Email == "" {
		email, fetchErr := h.fetchGitHubPrimaryEmail(ctx, client)
		if fetchErr != nil {
			h.logger.Warn("Failed to fetch GitHub emails, using noreply address",
				zap.String("username", userInfo.Username),
				zap.Error(fetchErr))
			// Use GitHub's noreply email format as fallback
			userInfo.Email = fmt.Sprintf("%s@users.noreply.github.com", userInfo.Username)
		} else {
			userInfo.Email = email
		}
	}

	return userInfo, nil
}

// fetchGitHubPrimaryEmail fetches the primary email from GitHub's /user/emails endpoint
func (h *OAuthHandler) fetchGitHubPrimaryEmail(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching emails: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub returned status %d: %s", resp.StatusCode, string(body))
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("decoding emails: %w", err)
	}

	// Find primary verified email first
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	// Fall back to any verified email
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}

	// Fall back to any email
	if len(emails) > 0 {
		return emails[0].Email, nil
	}

	return "", fmt.Errorf("no email addresses found")
}

// OAuthUserInfo holds normalized user information from OAuth providers
type OAuthUserInfo struct {
	ProviderID string // Provider-specific user ID
	Email      string
	Name       string
	Username   string
	AvatarURL  string
}

// parseUserInfo parses provider-specific user info response
func (h *OAuthHandler) parseUserInfo(providerID auth.OAuthProviderID, body io.Reader) (*OAuthUserInfo, error) {
	var rawData map[string]interface{}
	if err := json.NewDecoder(body).Decode(&rawData); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	userInfo := &OAuthUserInfo{}

	switch providerID {
	case auth.OAuthProviderGoogle:
		userInfo.ProviderID = getString(rawData, "id")
		userInfo.Email = getString(rawData, "email")
		userInfo.Name = getString(rawData, "name")
		userInfo.Username = strings.Split(userInfo.Email, "@")[0]
		userInfo.AvatarURL = getString(rawData, "picture")

	case auth.OAuthProviderGitHub:
		userInfo.ProviderID = fmt.Sprintf("%v", rawData["id"])
		userInfo.Email = getString(rawData, "email")
		userInfo.Name = getString(rawData, "name")
		userInfo.Username = getString(rawData, "login")
		userInfo.AvatarURL = getString(rawData, "avatar_url")
		// Note: GitHub may not return email if user's email is private.
		// We'll fetch from /user/emails endpoint in fetchUserInfo if needed.

	case auth.OAuthProviderMicrosoft:
		userInfo.ProviderID = getString(rawData, "id")
		userInfo.Email = getString(rawData, "mail")
		if userInfo.Email == "" {
			userInfo.Email = getString(rawData, "userPrincipalName")
		}
		userInfo.Name = getString(rawData, "displayName")
		userInfo.Username = strings.Split(userInfo.Email, "@")[0]

	case auth.OAuthProviderGitLab:
		userInfo.ProviderID = fmt.Sprintf("%v", rawData["id"])
		userInfo.Email = getString(rawData, "email")
		userInfo.Name = getString(rawData, "name")
		userInfo.Username = getString(rawData, "username")
		userInfo.AvatarURL = getString(rawData, "avatar_url")

	default:
		return nil, fmt.Errorf("unsupported provider: %s", providerID)
	}

	// Validate required fields
	// Note: GitHub email may be empty here - it will be fetched from /user/emails in fetchUserInfo
	if userInfo.Email == "" && providerID != auth.OAuthProviderGitHub {
		return nil, fmt.Errorf("email not provided by OAuth provider")
	}

	if userInfo.Name == "" {
		userInfo.Name = userInfo.Username
	}

	if userInfo.Username == "" {
		userInfo.Username = strings.Split(userInfo.Email, "@")[0]
	}

	return userInfo, nil
}

// getString safely extracts a string from a map
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// findOrCreateUser finds an existing user by email or creates a new one
func (h *OAuthHandler) findOrCreateUser(_ context.Context, providerID auth.OAuthProviderID, userInfo *OAuthUserInfo) (*models.User, error) {
	// Try to find user by email
	user, err := h.userRepo.GetByUsernameOrEmail(userInfo.Email)
	if err == nil {
		// User exists
		return user, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("looking up user: %w", err)
	}

	// User doesn't exist - create new user
	// Generate unique username if needed
	username := h.generateUniqueUsername(userInfo.Username)

	user = &models.User{
		Name:     userInfo.Name,
		Username: username,
		Email:    userInfo.Email,
	}

	// Set a random password (user will use OAuth to login)
	randomPassword, err := generateRandomPassword(32)
	if err != nil {
		return nil, fmt.Errorf("generating random password: %w", err)
	}

	if err := user.SetPassword(randomPassword, 12); err != nil {
		return nil, fmt.Errorf("setting password: %w", err)
	}

	// Create user
	if err := h.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	h.logger.Info("Created new user from OAuth",
		zap.String("provider", string(providerID)),
		zap.Int("user_id", user.ID),
		zap.String("username", user.Username),
		zap.String("email", user.Email))

	return user, nil
}

// generateUniqueUsername generates a unique username
func (h *OAuthHandler) generateUniqueUsername(baseUsername string) string {
	username := baseUsername

	// Check if username exists
	_, err := h.userRepo.GetByUsernameOrEmail(username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return username
	}

	// Add random suffix
	for i := 0; i < 10; i++ {
		suffix := generateRandomSuffix(4)
		candidateUsername := username + "_" + suffix

		_, err := h.userRepo.GetByUsernameOrEmail(candidateUsername)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return candidateUsername
		}
	}

	// Last resort: use timestamp
	return fmt.Sprintf("%s_%d", username, time.Now().UnixNano())
}

// generateRandomPassword generates a secure random password
func generateRandomPassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// generateRandomSuffix generates a random alphanumeric suffix
func generateRandomSuffix(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano()%10000)
	}
	for i := range bytes {
		bytes[i] = charset[bytes[i]%byte(len(charset))]
	}
	return string(bytes)
}

// handleOAuthError handles OAuth errors by redirecting to an error page or the original redirect URL
func (h *OAuthHandler) handleOAuthError(w http.ResponseWriter, r *http.Request, message string, redirectURL string) {
	h.logger.Warn("OAuth error", zap.String("message", message))

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.config.ACL.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	// Clear PKCE cookie
	http.SetCookie(w, &http.Cookie{
		Name:     oauthPKCECookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.config.ACL.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	// Validate redirect URL to prevent open redirect attacks
	redirectURL = h.validateRedirectURL(redirectURL)

	// Add error message to redirect URL
	errorURL, err := url.Parse(redirectURL)
	if err != nil {
		errorURL = &url.URL{Path: "/"}
	}

	q := errorURL.Query()
	q.Set("oauth_error", message)
	errorURL.RawQuery = q.Encode()

	http.Redirect(w, r, errorURL.String(), http.StatusTemporaryRedirect)
}
