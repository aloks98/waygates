package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	chimw "github.com/aloks98/goauth/middleware/chi"
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
	auditService    service.AuditServiceInterface
	settingsRepo    repository.SettingsRepositoryInterface
	config          *config.Config
	logger          *zap.Logger
}

// OAuthHandlerConfig holds configuration for creating an OAuthHandler
type OAuthHandlerConfig struct {
	ProviderManager *auth.OAuthProviderManager
	ACLService      service.ACLServiceInterface
	UserRepo        repository.UserRepositoryInterface
	AuditService    service.AuditServiceInterface
	SettingsRepo    repository.SettingsRepositoryInterface
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
		auditService:    cfg.AuditService,
		settingsRepo:    cfg.SettingsRepo,
		config:          cfg.Config,
		logger:          logger.Named("oauth-handler"),
	}
}

// OAuth state cookie name
const oauthStateCookieName = "waygates_oauth_state"

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
// Returns a list of OAuth providers that are both available AND enabled (public endpoint for login page)
func (h *OAuthHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	// Get all available providers (env vars configured)
	allProviders := h.providerManager.GetAvailableProvidersPublic()

	// Get enabled state from settings
	enabledMap := h.getEnabledProvidersMap()

	// Filter to only return providers that are both available AND enabled
	var enabledProviders []auth.OAuthProviderPublic
	for _, p := range allProviders {
		if p.Available && enabledMap[string(p.ID)] {
			p.Enabled = true
			enabledProviders = append(enabledProviders, p)
		}
	}

	// Return array directly for frontend compatibility
	utils.Success(w, enabledProviders, "")
}

// =============================================================================
// Admin Providers Management (Protected)
// =============================================================================

// GetOAuthProviders handles GET /api/acl/oauth/providers
// Returns all OAuth providers with their available and enabled status (protected endpoint for admin)
func (h *OAuthHandler) GetOAuthProviders(w http.ResponseWriter, r *http.Request) {
	// Get all providers (not just available ones)
	allProviders := h.providerManager.GetAllProvidersPublic()

	// Get enabled state from settings
	enabledMap := h.getEnabledProvidersMap()

	// Set the enabled status for each provider
	for i := range allProviders {
		allProviders[i].Enabled = enabledMap[string(allProviders[i].ID)]
	}

	utils.Success(w, allProviders, "")
}

// UpdateOAuthProviderRequest is the request body for updating an OAuth provider
type UpdateOAuthProviderRequest struct {
	Enabled bool `json:"enabled"`
}

// UpdateOAuthProvider handles PUT /api/acl/oauth/providers/{id}
// Enables or disables an OAuth provider (protected endpoint for admin)
func (h *OAuthHandler) UpdateOAuthProvider(w http.ResponseWriter, r *http.Request) {
	providerID := chi.URLParam(r, "id")

	// Validate provider ID
	if !auth.ValidateProviderID(providerID) {
		utils.BadRequest(w, "Invalid OAuth provider ID", nil)
		return
	}

	// Check if provider is available (env vars configured)
	if !h.providerManager.IsAvailable(auth.OAuthProviderID(providerID)) {
		utils.BadRequest(w, "OAuth provider is not available (environment variables not configured)", nil)
		return
	}

	// Parse request body
	var req UpdateOAuthProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}

	// Get current enabled map
	enabledMap := h.getEnabledProvidersMap()

	// Update the provider's enabled state
	enabledMap[providerID] = req.Enabled

	// Save to settings
	if err := h.saveEnabledProvidersMap(enabledMap); err != nil {
		h.logger.Error("Failed to save OAuth provider settings", zap.Error(err))
		utils.InternalError(w, "Failed to save settings")
		return
	}

	// Log the action
	if h.auditService != nil {
		userIDStr := chimw.UserID(r)
		userID, _ := strconv.Atoi(userIDStr)
		if userID > 0 {
			oldVal := "disabled"
			newVal := "enabled"
			if !req.Enabled {
				oldVal = "enabled"
				newVal = "disabled"
			}
			_ = h.auditService.LogSettingsUpdate(r.Context(), userID, "oauth_provider."+providerID, oldVal, newVal, getClientIP(r), r.UserAgent())
		}
	}

	// Return updated provider info
	provider, _ := h.providerManager.GetProviderByString(providerID)
	result := provider.ToPublic()
	result.Enabled = req.Enabled

	utils.Success(w, result, fmt.Sprintf("OAuth provider %s", map[bool]string{true: "enabled", false: "disabled"}[req.Enabled]))
}

// =============================================================================
// Helper functions for OAuth provider settings
// =============================================================================

// getEnabledProvidersMap retrieves the enabled state map from settings
func (h *OAuthHandler) getEnabledProvidersMap() map[string]bool {
	enabledMap := make(map[string]bool)

	if h.settingsRepo == nil {
		return enabledMap
	}

	setting, err := h.settingsRepo.Get(models.SettingOAuthProviders)
	if err != nil || setting == nil {
		return enabledMap
	}

	if err := json.Unmarshal([]byte(setting.Value), &enabledMap); err != nil {
		h.logger.Warn("Failed to parse OAuth providers settings", zap.Error(err))
		return make(map[string]bool)
	}

	return enabledMap
}

// saveEnabledProvidersMap saves the enabled state map to settings
func (h *OAuthHandler) saveEnabledProvidersMap(enabledMap map[string]bool) error {
	if h.settingsRepo == nil {
		return fmt.Errorf("settings repository not configured")
	}

	data, err := json.Marshal(enabledMap)
	if err != nil {
		return err
	}

	return h.settingsRepo.Set(models.SettingOAuthProviders, string(data))
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

	// Get redirect URL from query parameter
	redirectURL := r.URL.Query().Get("redirect")
	if redirectURL == "" {
		redirectURL = "/"
	}

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

	// Build OAuth2 config
	oauth2Config := h.buildOAuth2Config(provider)

	// Generate authorization URL
	authURL := oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)

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

	// Extract redirect URL from state
	redirectURL := h.extractRedirectFromState(stateParam)

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

	// Build OAuth2 config
	oauth2Config := h.buildOAuth2Config(provider)

	// Exchange code for token
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	token, err := oauth2Config.Exchange(ctx, code)
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
		zap.String("username", user.Username))

	// Redirect to original URL
	if redirectURL == "" {
		redirectURL = "/"
	}
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}

// =============================================================================
// Helper Functions
// =============================================================================

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
	return h.parseUserInfo(provider.ID, resp.Body)
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

		// GitHub may not return email in user info, would need to fetch from /user/emails
		// For simplicity, we'll use the username as email prefix if email is empty
		if userInfo.Email == "" && userInfo.Username != "" {
			// In production, you should fetch from /user/emails endpoint
			h.logger.Warn("GitHub user has no public email, using placeholder",
				zap.String("username", userInfo.Username))
		}

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
	if userInfo.Email == "" {
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
func (h *OAuthHandler) findOrCreateUser(ctx context.Context, providerID auth.OAuthProviderID, userInfo *OAuthUserInfo) (*models.User, error) {
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

	// Build error redirect URL
	if redirectURL == "" {
		redirectURL = "/"
	}

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
