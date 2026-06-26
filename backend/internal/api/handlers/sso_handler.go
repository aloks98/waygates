package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"

	"github.com/aloks98/waygates/backend/internal/config"
	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
	"github.com/aloks98/waygates/backend/internal/service"
	"github.com/aloks98/waygates/backend/internal/utils"
)

// SSOHandler serves the admin OIDC login + config endpoints.
type SSOHandler struct {
	svc      *service.SSOService
	settings repository.SettingsRepositoryInterface
	config   *config.Config
	logger   *zap.Logger
}

// NewSSOHandler constructs an SSOHandler.
func NewSSOHandler(svc *service.SSOService, settings repository.SettingsRepositoryInterface, cfg *config.Config, logger *zap.Logger) *SSOHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SSOHandler{svc: svc, settings: settings, config: cfg, logger: logger}
}

// Status (public): { enabled, label } for the login page.
func (h *SSOHandler) Status(w http.ResponseWriter, _ *http.Request) {
	utils.Success(w, map[string]any{
		"enabled": h.svc.Enabled(),
		"label":   h.svc.ButtonLabel(),
	}, "")
}

// Lookup (public, rate-limited): routes an email to "sso" or "password".
func (h *SSOHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}
	utils.Success(w, map[string]string{"method": h.svc.LookupMethod(req.Email)}, "")
}

// Login (public): redirect to the IdP.
func (h *SSOHandler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.svc.Enabled() {
		http.NotFound(w, r)
		return
	}
	state, err := generateOpaque()
	if err != nil {
		h.fail(w, r, "sso_failed", "generate state", err)
		return
	}
	verifier, err := generateCodeVerifier()
	if err != nil {
		h.fail(w, r, "sso_failed", "generate verifier", err)
		return
	}
	challenge := generateCodeChallenge(verifier)
	h.setFlowCookie(w, oauthStateCookieName, state)
	h.setFlowCookie(w, oauthPKCECookieName, verifier)

	authURL, err := h.svc.AuthCodeURL(r.Context(), h.redirectURI(r), state, challenge, r.URL.Query().Get("login_hint"))
	if err != nil {
		h.fail(w, r, "sso_failed", "auth code url", err)
		return
	}
	//nolint:gosec // G710: authURL is the IdP's discovered authorize endpoint; only the URL-encoded login_hint is request-derived, not the redirect host
	http.Redirect(w, r, authURL, http.StatusFound)
}

// Callback (public): verify, match/provision, mint, hand off via one-time code.
func (h *SSOHandler) Callback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie(oauthStateCookieName)
	if err != nil || stateCookie.Value == "" || stateCookie.Value != r.URL.Query().Get("state") {
		h.fail(w, r, "state_mismatch", "state mismatch", err)
		return
	}
	pkceCookie, err := r.Cookie(oauthPKCECookieName)
	if err != nil || pkceCookie.Value == "" {
		h.fail(w, r, "state_mismatch", "missing pkce", err)
		return
	}
	h.clearFlowCookie(w, oauthStateCookieName)
	h.clearFlowCookie(w, oauthPKCECookieName)

	pair, err := h.svc.CompleteLogin(r.Context(), h.redirectURI(r), r.URL.Query().Get("code"), pkceCookie.Value)
	if err != nil {
		h.fail(w, r, ssoErrorCode(err), "complete login", err)
		return
	}
	code, err := h.svc.Codes().Issue(pair)
	if err != nil {
		h.fail(w, r, "sso_failed", "issue code", err)
		return
	}
	http.Redirect(w, r, "/auth/sso/callback?code="+url.QueryEscape(code), http.StatusFound)
}

// Exchange (public): one-time code -> JWT pair.
func (h *SSOHandler) Exchange(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}
	pair, ok := h.svc.Codes().Consume(req.Code)
	if !ok {
		utils.Unauthorized(w, "Invalid or expired code")
		return
	}
	utils.Success(w, map[string]string{
		"access_token":  pair.AccessToken,
		"refresh_token": pair.RefreshToken,
	}, "Login successful")
}

// GetConfig (protected, settings:read): returns the SSO configuration (no secret).
func (h *SSOHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	stored := h.settings.GetValue(models.SettingSSOOIDCClientSecret, "")
	utils.Success(w, map[string]any{
		"enabled":           h.settings.GetValue(models.SettingSSOEnabled, "false") == "true",
		"issuer":            h.settings.GetValue(models.SettingSSOOIDCIssuer, ""),
		"client_id":         h.settings.GetValue(models.SettingSSOOIDCClientID, ""),
		"has_client_secret": stored != "",
		"auto_provision":    h.settings.GetValue(models.SettingSSOAutoProvision, "false") == "true",
		"default_role":      h.settings.GetValue(models.SettingSSODefaultRole, ""),
		"button_label":      h.settings.GetValue(models.SettingSSOButtonLabel, ""),
		"base_url":          h.settings.GetValue(models.SettingSSOBaseURL, ""),
		"redirect_uri":      h.redirectURI(r),
	}, "")
}

// UpdateConfig (protected, settings:write): writes sso.* settings. Blank client_secret keeps the existing value.
func (h *SSOHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Enabled       bool   `json:"enabled"`
		Issuer        string `json:"issuer"`
		ClientID      string `json:"client_id"`
		ClientSecret  string `json:"client_secret"`
		AutoProvision bool   `json:"auto_provision"`
		DefaultRole   string `json:"default_role"`
		ButtonLabel   string `json:"button_label"`
		BaseURL       string `json:"base_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}
	boolStr := func(b bool) string {
		if b {
			return "true"
		}
		return "false"
	}
	sets := []struct{ k, v string }{
		{models.SettingSSOEnabled, boolStr(body.Enabled)},
		{models.SettingSSOOIDCIssuer, body.Issuer},
		{models.SettingSSOOIDCClientID, body.ClientID},
		{models.SettingSSOAutoProvision, boolStr(body.AutoProvision)},
		{models.SettingSSODefaultRole, body.DefaultRole},
		{models.SettingSSOButtonLabel, body.ButtonLabel},
		{models.SettingSSOBaseURL, body.BaseURL},
	}
	for _, s := range sets {
		if err := h.settings.Set(s.k, s.v); err != nil {
			h.logger.Error("failed to save sso setting", zap.String("key", s.k), zap.Error(err))
			utils.InternalError(w, "Failed to save SSO configuration")
			return
		}
	}
	// Only overwrite the secret when a non-blank value is provided.
	if body.ClientSecret != "" {
		if err := h.settings.Set(models.SettingSSOOIDCClientSecret, body.ClientSecret); err != nil {
			h.logger.Error("failed to save sso client secret", zap.Error(err))
			utils.InternalError(w, "Failed to save SSO configuration")
			return
		}
	}
	h.svc.Invalidate()
	h.GetConfig(w, r)
}

// Test (protected, settings:write): verifies OIDC discovery against the provided (or saved) issuer.
func (h *SSOHandler) Test(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Issuer string `json:"issuer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.BadRequest(w, "Invalid request body", nil)
		return
	}
	issuer := strings.TrimSpace(body.Issuer)
	if issuer == "" {
		issuer = h.settings.GetValue(models.SettingSSOOIDCIssuer, "")
	}
	if issuer == "" {
		utils.Success(w, map[string]any{"ok": false, "error": "no issuer configured"}, "")
		return
	}
	_, err := oidc.NewProvider(r.Context(), issuer)
	if err != nil {
		h.logger.Warn("sso oidc discovery failed", zap.String("issuer", issuer), zap.Error(err))
		utils.Success(w, map[string]any{"ok": false, "error": err.Error()}, "")
		return
	}
	utils.Success(w, map[string]any{"ok": true}, "")
}

// --- helpers ---

// generateOpaque returns a 32-byte cryptographically random base64url string (no padding).
func generateOpaque() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// trimSlash removes a trailing slash from s.
func trimSlash(s string) string { return strings.TrimSuffix(s, "/") }

// redirectURI builds the SSO callback URL from the configured base_url or the request.
func (h *SSOHandler) redirectURI(r *http.Request) string {
	if base := h.settings.GetValue(models.SettingSSOBaseURL, ""); base != "" {
		return trimSlash(base) + "/api/auth/sso/callback"
	}
	scheme := "https"
	if xf := r.Header.Get("X-Forwarded-Proto"); xf != "" {
		scheme = xf
	} else if r.TLS == nil {
		scheme = "http"
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return scheme + "://" + host + "/api/auth/sso/callback"
}

// setFlowCookie writes a short-lived HttpOnly SSO flow cookie.
func (h *SSOHandler) setFlowCookie(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // G124: Secure is config-driven (ACL.CookieSecure) to allow HTTP in dev; HttpOnly+SameSite=Lax always set
		Name:     name,
		Value:    value,
		HttpOnly: true,
		Secure:   h.config.ACL.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   int(stateExpiration.Seconds()),
	})
}

// clearFlowCookie expires a SSO flow cookie.
func (h *SSOHandler) clearFlowCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // G124: Secure is config-driven (ACL.CookieSecure) to allow HTTP in dev; HttpOnly+SameSite=Lax always set
		Name:     name,
		Value:    "",
		HttpOnly: true,
		Secure:   h.config.ACL.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   -1,
	})
}

// ssoErrorCode maps service errors to login-page sso_error codes.
func ssoErrorCode(err error) string {
	switch {
	case errors.Is(err, service.ErrNoAccount):
		return "no_account"
	case errors.Is(err, service.ErrAccountDisabled):
		return "disabled"
	case errors.Is(err, service.ErrEmailUnverified):
		return "email_unverified"
	case errors.Is(err, service.ErrSSODisabled):
		return "sso_disabled"
	default:
		return "sso_failed"
	}
}

// fail logs, audits, and redirects to the login page with an sso_error code.
func (h *SSOHandler) fail(w http.ResponseWriter, r *http.Request, code, msg string, err error) {
	h.logger.Warn("sso flow failed", zap.String("reason", code), zap.String("at", msg), zap.Error(err))
	h.svc.AuditLoginFailure(r.Context(), code)
	http.Redirect(w, r, "/login?sso_error="+code, http.StatusFound)
}
