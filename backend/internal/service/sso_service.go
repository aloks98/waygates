package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aloks98/goauth/token"
	"github.com/coreos/go-oidc/v3/oidc"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// Typed errors mapped to login-page sso_error codes by the handler.
var (
	ErrSSODisabled     = errors.New("sso disabled")
	ErrNoAccount       = errors.New("no matching account")
	ErrAccountDisabled = errors.New("account disabled")
	ErrEmailUnverified = errors.New("email not verified")
)

// flexBool decodes JSON true/false (bool), "true"/"false" (string), and null as a boolean.
// This handles IdPs (Zitadel, Okta, Azure) that encode email_verified as a string.
type flexBool bool

func (b *flexBool) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch v := raw.(type) {
	case bool:
		*b = flexBool(v)
	case string:
		*b = flexBool(strings.EqualFold(v, "true"))
	case nil:
		*b = false
	default:
		*b = false
	}
	return nil
}

// userRepoForSSO is the subset of the user repository the SSO flow needs.
type userRepoForSSO interface {
	GetByEmail(email string) (*models.User, error)
	GetByUsernameOrEmail(identifier string) (*models.User, error)
	Create(user *models.User) error
	UpdateLastLogin(id int, t time.Time) error
}

// ssoTokenIssuer is the subset of the goauth provider the SSO flow needs.
type ssoTokenIssuer interface {
	AssignRole(ctx context.Context, userID, role string) error
	GenerateTokenPair(ctx context.Context, userID string, metadata map[string]any) (*token.Pair, error)
}

// idTokenClaims are the OIDC claims SSO consumes.
type idTokenClaims struct {
	Email         string   `json:"email"`
	EmailVerified flexBool `json:"email_verified"`
	Sub           string   `json:"sub"`
	Name          string   `json:"name"`
}

// idTokenVerifier verifies a raw ID token and returns its claims.
type idTokenVerifier interface {
	Verify(ctx context.Context, rawIDToken string) (idTokenClaims, error)
}

// SSODeps are the SSOService constructor dependencies.
type SSODeps struct {
	Settings repository.SettingsRepositoryInterface
	Users    userRepoForSSO
	Issuer   ssoTokenIssuer
	Audit    AuditServiceInterface
	Logger   *zap.Logger
}

// SSOService performs the admin OIDC login flow.
type SSOService struct {
	settings repository.SettingsRepositoryInterface
	users    userRepoForSSO
	issuer   ssoTokenIssuer
	audit    AuditServiceInterface
	logger   *zap.Logger
	codes    *OneTimeCodeStore

	mu           sync.Mutex
	cachedIssuer string
	provider     *oidc.Provider

	// test seams
	enabledFn  func() bool
	verifierFn func(cfg SSOConfig, provider *oidc.Provider) idTokenVerifier
}

// NewSSOService constructs an SSOService.
func NewSSOService(deps SSODeps) *SSOService {
	logger := deps.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SSOService{
		settings: deps.Settings,
		users:    deps.Users,
		issuer:   deps.Issuer,
		audit:    deps.Audit,
		logger:   logger.Named("sso"),
		codes:    NewOneTimeCodeStore(60 * time.Second),
	}
}

func (s *SSOService) config() SSOConfig { return LoadSSOConfig(s.settings) }

// Codes exposes the one-time-code store for the handler.
func (s *SSOService) Codes() *OneTimeCodeStore { return s.codes }

// Invalidate clears the cached OIDC provider; call after sso.* settings change.
func (s *SSOService) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.provider = nil
	s.cachedIssuer = ""
}

// Enabled reports whether SSO is on.
func (s *SSOService) Enabled() bool {
	if s.enabledFn != nil {
		return s.enabledFn()
	}
	return s.config().Enabled
}

// ButtonLabel returns the configured login-button label.
// ButtonLabel returns the configured login-button label, falling back to a
// default when it is unset or blank (the stored setting may be an empty string).
func (s *SSOService) ButtonLabel() string {
	if label := strings.TrimSpace(s.config().ButtonLabel); label != "" {
		return label
	}
	return "Sign in with SSO"
}

// LookupMethod returns "sso" when SSO is on and a passwordless account exists
// for email, else "password".
func (s *SSOService) LookupMethod(email string) string {
	if !s.Enabled() {
		return "password"
	}
	u, err := s.users.GetByEmail(email)
	if err == nil && u != nil && u.PasswordHash == "" {
		return "sso"
	}
	return "password"
}

// providerFor returns a cached *oidc.Provider for the configured issuer,
// running discovery once per issuer value.
func (s *SSOService) providerFor(ctx context.Context, cfg SSOConfig) (*oidc.Provider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.provider != nil && s.cachedIssuer == cfg.Issuer {
		return s.provider, nil
	}
	p, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery for %q: %w", cfg.Issuer, err)
	}
	s.provider = p
	s.cachedIssuer = cfg.Issuer
	return p, nil
}

func (s *SSOService) oauth2Config(p *oidc.Provider, cfg SSOConfig, redirectURI string) oauth2.Config {
	return oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     p.Endpoint(),
		RedirectURL:  redirectURI,
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}
}

// AuthCodeURL builds the IdP authorize URL (with PKCE + optional login_hint).
func (s *SSOService) AuthCodeURL(ctx context.Context, redirectURI, state, codeChallenge, loginHint string) (string, error) {
	cfg := s.config()
	if !cfg.Enabled {
		return "", ErrSSODisabled
	}
	p, err := s.providerFor(ctx, cfg)
	if err != nil {
		return "", err
	}
	oc := s.oauth2Config(p, cfg, redirectURI)
	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}
	if loginHint != "" {
		opts = append(opts, oauth2.SetAuthURLParam("login_hint", loginHint))
	}
	return oc.AuthCodeURL(state, opts...), nil
}

// CompleteLogin exchanges the code, verifies the ID token, matches/provisions
// the user, and mints the JWT pair.
func (s *SSOService) CompleteLogin(ctx context.Context, redirectURI, code, codeVerifier string) (*token.Pair, error) {
	cfg := s.config()
	if !cfg.Enabled {
		return nil, ErrSSODisabled
	}
	p, err := s.providerFor(ctx, cfg)
	if err != nil {
		return nil, err
	}
	oc := s.oauth2Config(p, cfg, redirectURI)

	oauth2Token, err := oc.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}
	rawID, ok := oauth2Token.Extra("id_token").(string)
	if !ok || rawID == "" {
		return nil, errors.New("no id_token in token response")
	}

	var verifier idTokenVerifier
	if s.verifierFn != nil {
		verifier = s.verifierFn(cfg, p)
	} else {
		verifier = &oidcVerifier{v: p.Verifier(&oidc.Config{ClientID: cfg.ClientID})}
	}
	claims, err := verifier.Verify(ctx, rawID)
	if err != nil {
		return nil, fmt.Errorf("id token verify: %w", err)
	}

	// UserInfo fallback: some IdPs (Zitadel default, Okta, Azure) omit email/
	// email_verified from the ID token and publish them only at the UserInfo endpoint.
	if claims.Email == "" || !claims.EmailVerified {
		ui, uiErr := p.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
		if uiErr != nil {
			return nil, fmt.Errorf("userinfo fetch: %w", uiErr)
		}
		// Token-substitution protection (OIDC core 5.3.2): UserInfo sub MUST match
		// the ID token sub.
		if ui.Subject != claims.Sub {
			return nil, errors.New("userinfo subject does not match id token subject")
		}
		var uc idTokenClaims
		if err := ui.Claims(&uc); err != nil {
			return nil, fmt.Errorf("userinfo claims: %w", err)
		}
		claims = mergeUserInfoClaims(claims, uc)
	}

	user, err := s.matchOrProvision(ctx, claims, cfg)
	if err != nil {
		return nil, err
	}
	return s.mint(ctx, user)
}

// matchOrProvision applies the SSO account-matching rules (pure: no token mint).
func (s *SSOService) matchOrProvision(ctx context.Context, claims idTokenClaims, cfg SSOConfig) (*models.User, error) {
	if claims.Email == "" || !claims.EmailVerified {
		return nil, ErrEmailUnverified
	}
	u, err := s.users.GetByEmail(claims.Email)
	if err == nil && u != nil {
		if !u.Active {
			return nil, ErrAccountDisabled
		}
		return u, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) && err != nil {
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	if !cfg.AutoProvision {
		return nil, ErrNoAccount
	}

	name := claims.Name
	if name == "" {
		name = claims.Email
	}
	newUser := &models.User{
		Name:               name,
		Username:           s.uniqueUsername(claims.Email),
		Email:              claims.Email,
		PasswordHash:       "",
		Active:             true,
		MustChangePassword: false,
	}
	if err := s.users.Create(newUser); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	role := cfg.DefaultRole
	if role == "" {
		role = "viewer"
	}
	if err := s.issuer.AssignRole(ctx, fmt.Sprintf("%d", newUser.ID), role); err != nil {
		return nil, fmt.Errorf("assign role: %w", err)
	}
	return newUser, nil
}

// uniqueUsername derives a username from the email local-part, avoiding collisions.
func (s *SSOService) uniqueUsername(email string) string {
	base := email
	if at := strings.IndexByte(email, '@'); at > 0 {
		base = email[:at]
	}
	candidate := base
	for i := 1; ; i++ {
		if _, err := s.users.GetByUsernameOrEmail(candidate); errors.Is(err, gorm.ErrRecordNotFound) {
			return candidate
		}
		candidate = fmt.Sprintf("%s%d", base, i)
	}
}

// AuditLoginFailure writes an SSO login-failure event to the audit log.
// It is a no-op when no audit service is configured.
func (s *SSOService) AuditLoginFailure(ctx context.Context, reason string) {
	if s.audit == nil {
		return
	}
	_ = s.audit.LogEvent(ctx, models.AuditEvent{
		UserID:  nil,
		Action:  models.AuditActionAuthSSOLogin,
		Status:  "failure",
		Details: map[string]interface{}{"reason": reason},
	})
}

// mint issues the JWT pair, updates last-login, and audits the success.
func (s *SSOService) mint(ctx context.Context, user *models.User) (*token.Pair, error) {
	pair, err := s.issuer.GenerateTokenPair(ctx, fmt.Sprintf("%d", user.ID), map[string]any{
		"username": user.Username,
		"email":    user.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("generate tokens: %w", err)
	}
	if err := s.users.UpdateLastLogin(user.ID, time.Now()); err != nil {
		s.logger.Error("update last login", zap.Int("user_id", user.ID), zap.Error(err))
	}
	if s.audit != nil {
		uid := user.ID
		_ = s.audit.LogEvent(ctx, models.AuditEvent{
			UserID:  &uid,
			Action:  models.AuditActionAuthSSOLogin,
			Status:  "success",
			Details: map[string]interface{}{"email": user.Email},
		})
	}
	return pair, nil
}

// oidcVerifier adapts go-oidc's verifier to idTokenVerifier.
type oidcVerifier struct{ v *oidc.IDTokenVerifier }

func (o *oidcVerifier) Verify(ctx context.Context, raw string) (idTokenClaims, error) {
	idt, err := o.v.Verify(ctx, raw)
	if err != nil {
		return idTokenClaims{}, err
	}
	var c idTokenClaims
	if err := idt.Claims(&c); err != nil {
		return idTokenClaims{}, err
	}
	return c, nil
}

// mergeUserInfoClaims overlays UserInfo claims onto ID-token claims: fills a
// missing email, upgrades email_verified to true if UserInfo asserts it, and
// fills a missing name. The subject always stays the verified ID-token subject.
func mergeUserInfoClaims(primary, fallback idTokenClaims) idTokenClaims {
	out := primary
	if out.Email == "" {
		out.Email = fallback.Email
	}
	if !out.EmailVerified {
		out.EmailVerified = fallback.EmailVerified
	}
	if out.Name == "" {
		out.Name = fallback.Name
	}
	return out
}
