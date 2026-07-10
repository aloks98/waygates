package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aloks98/goauth/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// --- mocks ---

type ssoMockUserRepo struct {
	byEmail   map[string]*models.User
	created   *models.User
	lastLogin int
}

func (m *ssoMockUserRepo) GetByEmail(email string) (*models.User, error) {
	if u, ok := m.byEmail[email]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *ssoMockUserRepo) GetByUsernameOrEmail(id string) (*models.User, error) {
	if u, ok := m.byEmail[id]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *ssoMockUserRepo) Create(u *models.User) error {
	u.ID = 99
	m.created = u
	return nil
}
func (m *ssoMockUserRepo) UpdateLastLogin(id int, _ time.Time) error {
	m.lastLogin = id
	return nil
}

type ssoMockIssuer struct {
	assignedRole string
	assignErr    error
}

func (m *ssoMockIssuer) AssignRole(_ context.Context, _, role string) error {
	m.assignedRole = role
	return m.assignErr
}
func (m *ssoMockIssuer) GenerateTokenPair(_ context.Context, _ string, _ map[string]any) (*token.Pair, error) {
	return &token.Pair{AccessToken: "a", RefreshToken: "r"}, nil
}

func newTestSSO(repo userRepoForSSO, issuer ssoTokenIssuer) *SSOService {
	return &SSOService{users: repo, issuer: issuer, audit: nil, codes: NewOneTimeCodeStore(time.Minute)}
}

func TestMatchOrProvision_FoundActive(t *testing.T) {
	repo := &ssoMockUserRepo{byEmail: map[string]*models.User{
		"a@x.com": {ID: 1, Email: "a@x.com", Username: "a", Active: true, PasswordHash: "h"},
	}}
	svc := newTestSSO(repo, &ssoMockIssuer{})
	u, err := svc.matchOrProvision(context.Background(),
		idTokenClaims{Email: "a@x.com", EmailVerified: true}, SSOConfig{})
	require.NoError(t, err)
	assert.Equal(t, 1, u.ID)
}

func TestMatchOrProvision_FoundInactive(t *testing.T) {
	repo := &ssoMockUserRepo{byEmail: map[string]*models.User{
		"a@x.com": {ID: 1, Email: "a@x.com", Active: false},
	}}
	svc := newTestSSO(repo, &ssoMockIssuer{})
	_, err := svc.matchOrProvision(context.Background(),
		idTokenClaims{Email: "a@x.com", EmailVerified: true}, SSOConfig{})
	assert.ErrorIs(t, err, ErrAccountDisabled)
}

func TestMatchOrProvision_NoAccount_NoAutoProvision(t *testing.T) {
	repo := &ssoMockUserRepo{byEmail: map[string]*models.User{}}
	svc := newTestSSO(repo, &ssoMockIssuer{})
	_, err := svc.matchOrProvision(context.Background(),
		idTokenClaims{Email: "new@x.com", EmailVerified: true}, SSOConfig{AutoProvision: false})
	assert.ErrorIs(t, err, ErrNoAccount)
}

func TestMatchOrProvision_AutoProvision(t *testing.T) {
	repo := &ssoMockUserRepo{byEmail: map[string]*models.User{}}
	issuer := &ssoMockIssuer{}
	svc := newTestSSO(repo, issuer)
	u, err := svc.matchOrProvision(context.Background(),
		idTokenClaims{Email: "new@x.com", EmailVerified: true, Name: "New"},
		SSOConfig{AutoProvision: true, DefaultRole: "operator"})
	require.NoError(t, err)
	assert.Equal(t, "new@x.com", u.Email)
	assert.Equal(t, "new", u.Username)
	assert.Empty(t, repo.created.PasswordHash, "auto-provisioned users have no password")
	assert.Equal(t, "operator", issuer.assignedRole)
}

func TestMatchOrProvision_EmailUnverified(t *testing.T) {
	svc := newTestSSO(&ssoMockUserRepo{byEmail: map[string]*models.User{}}, &ssoMockIssuer{})
	_, err := svc.matchOrProvision(context.Background(),
		idTokenClaims{Email: "a@x.com", EmailVerified: false}, SSOConfig{})
	assert.ErrorIs(t, err, ErrEmailUnverified)
}

func TestMatchOrProvision_MissingEmail(t *testing.T) {
	svc := newTestSSO(&ssoMockUserRepo{byEmail: map[string]*models.User{}}, &ssoMockIssuer{})
	_, err := svc.matchOrProvision(context.Background(),
		idTokenClaims{Email: "", EmailVerified: true}, SSOConfig{})
	assert.ErrorIs(t, err, ErrEmailUnverified)
}

func TestLookupMethod(t *testing.T) {
	repo := &ssoMockUserRepo{byEmail: map[string]*models.User{
		"sso@x.com":  {ID: 1, Email: "sso@x.com", PasswordHash: ""},
		"pass@x.com": {ID: 2, Email: "pass@x.com", PasswordHash: "h"},
	}}
	svc := newTestSSO(repo, &ssoMockIssuer{})
	svc.enabledFn = func() bool { return true }
	assert.Equal(t, "sso", svc.LookupMethod("sso@x.com"))
	assert.Equal(t, "password", svc.LookupMethod("pass@x.com"))
	assert.Equal(t, "password", svc.LookupMethod("missing@x.com"))
	svc.enabledFn = func() bool { return false }
	assert.Equal(t, "password", svc.LookupMethod("sso@x.com"))
}

// ssoCapturingAudit is a minimal AuditServiceInterface that captures LogEvent calls.
// All other methods are deliberate no-ops.
type ssoCapturingAudit struct {
	events []models.AuditEvent
}

func (m *ssoCapturingAudit) LogEvent(_ context.Context, e models.AuditEvent) error {
	m.events = append(m.events, e)
	return nil
}
func (m *ssoCapturingAudit) GetConfig() (*models.AuditConfig, error) { return nil, nil }
func (m *ssoCapturingAudit) SetConfig(_ *models.AuditConfig) error   { return nil }
func (m *ssoCapturingAudit) InvalidateConfigCache()                  {}
func (m *ssoCapturingAudit) ListAuditLogs(_ repository.AuditLogListParams) (*models.AuditLogListResponse, error) {
	return &models.AuditLogListResponse{}, nil
}
func (m *ssoCapturingAudit) GetAuditLogByID(_ int) (*models.AuditLog, error) { return nil, nil }
func (m *ssoCapturingAudit) GetStats() (*models.AuditLogStats, error) {
	return &models.AuditLogStats{}, nil
}
func (m *ssoCapturingAudit) LogProxyCreate(_ context.Context, _ int, _ *models.Proxy, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogProxyUpdate(_ context.Context, _ int, _ *models.Proxy, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogProxyDelete(_ context.Context, _ int, _ int, _, _, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogProxyEnable(_ context.Context, _ int, _ *models.Proxy, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogProxyDisable(_ context.Context, _ int, _ *models.Proxy, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogProxyGroupCreate(_ context.Context, _ int, _ *models.ProxyGroup, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogProxyGroupUpdate(_ context.Context, _ int, _, _ *models.ProxyGroup, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogProxyGroupDelete(_ context.Context, _, _ int, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogProxyGroupRehome(_ context.Context, _, _ int, _, _ string, _ []int, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogLogin(_ context.Context, _ int, _, _, _ string) error    { return nil }
func (m *ssoCapturingAudit) LogLoginFailed(_ context.Context, _, _, _, _ string) error  { return nil }
func (m *ssoCapturingAudit) LogLogout(_ context.Context, _ int, _, _, _ string) error   { return nil }
func (m *ssoCapturingAudit) LogRegister(_ context.Context, _ int, _, _, _ string) error { return nil }
func (m *ssoCapturingAudit) LogPasswordChange(_ context.Context, _ int, _, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogSettingsUpdate(_ context.Context, _ int, _, _, _, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogSyncStarted(_ context.Context) error                   { return nil }
func (m *ssoCapturingAudit) LogSyncCompleted(_ context.Context, _ int) error          { return nil }
func (m *ssoCapturingAudit) LogSyncFailed(_ context.Context, _ string) error          { return nil }
func (m *ssoCapturingAudit) LogSystemStartup(_ context.Context) error                 { return nil }
func (m *ssoCapturingAudit) LogCaddyReload(_ context.Context, _ bool, _ string) error { return nil }
func (m *ssoCapturingAudit) LogACLGroupCreate(_ context.Context, _ int, _ *models.ACLGroup, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogACLGroupUpdate(_ context.Context, _ int, _ *models.ACLGroup, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogACLGroupDelete(_ context.Context, _ int, _ int, _, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogACLIPRuleAdd(_ context.Context, _ int, _ int, _ string, _ *models.ACLIPRule, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogACLIPRuleUpdate(_ context.Context, _ int, _ *models.ACLIPRule, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogACLIPRuleDelete(_ context.Context, _ int, _ int, _, _, _, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogACLBasicAuthAdd(_ context.Context, _ int, _ int, _, _, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogACLBasicAuthUpdate(_ context.Context, _ int, _ int, _, _, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogACLBasicAuthDelete(_ context.Context, _ int, _ int, _, _, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogACLWaygatesAuthUpdate(_ context.Context, _ int, _ int, _ string, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogACLAssignmentCreate(_ context.Context, _ int, _ int, _ string, _ int, _, _, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogACLAssignmentUpdate(_ context.Context, _ int, _ *models.ProxyACLAssignment, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogACLAssignmentDelete(_ context.Context, _ int, _ int, _ string, _ int, _, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogACLBrandingUpdate(_ context.Context, _ int, _ map[string]interface{}, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogACLSessionRevoke(_ context.Context, _ int, _ int, _, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogACLOAuthRestrictionSet(_ context.Context, _ int, _ int, _, _ string, _ *models.ACLOAuthProviderRestriction, _ bool, _ []string, _ []string, _, _ string) error {
	return nil
}
func (m *ssoCapturingAudit) LogACLOAuthRestrictionDelete(_ context.Context, _ int, _ int, _, _, _, _ string) error {
	return nil
}

var _ AuditServiceInterface = (*ssoCapturingAudit)(nil)

func TestAuditLoginFailure(t *testing.T) {
	audit := &ssoCapturingAudit{}
	svc := &SSOService{
		users:  &ssoMockUserRepo{byEmail: map[string]*models.User{}},
		issuer: &ssoMockIssuer{},
		audit:  audit,
		codes:  NewOneTimeCodeStore(time.Minute),
	}

	svc.AuditLoginFailure(context.Background(), "no_account")

	require.Len(t, audit.events, 1, "exactly one audit event must be emitted")
	evt := audit.events[0]
	assert.Equal(t, models.AuditActionAuthSSOLogin, evt.Action)
	assert.Equal(t, "failure", evt.Status)
	assert.Nil(t, evt.UserID, "failure events have no user ID")
	require.NotNil(t, evt.Details)
	assert.Equal(t, "no_account", evt.Details["reason"])
}

func TestAuditLoginFailure_NilAudit(_ *testing.T) {
	svc := &SSOService{
		users:  &ssoMockUserRepo{byEmail: map[string]*models.User{}},
		issuer: &ssoMockIssuer{},
		audit:  nil,
		codes:  NewOneTimeCodeStore(time.Minute),
	}
	// Must not panic when no audit service is configured.
	svc.AuditLoginFailure(context.Background(), "sso_disabled")
}

// TestFlexBool_UnmarshalJSON covers all JSON representations of email_verified.
func TestFlexBool_UnmarshalJSON(t *testing.T) {
	type wrapper struct {
		V flexBool `json:"v"`
	}
	cases := []struct {
		input string
		want  bool
	}{
		{`{"v":true}`, true},
		{`{"v":false}`, false},
		{`{"v":"true"}`, true},
		{`{"v":"false"}`, false},
		{`{"v":null}`, false},
	}
	for _, tc := range cases {
		var w wrapper
		require.NoError(t, json.Unmarshal([]byte(tc.input), &w), "input: %s", tc.input)
		assert.Equal(t, flexBool(tc.want), w.V, "input: %s", tc.input)
	}
}

// TestMergeUserInfoClaims covers the Zitadel-style case (email absent from ID
// token) and the "already complete" case where primary values are preserved.
func TestMergeUserInfoClaims(t *testing.T) {
	t.Run("zitadel case — email from UserInfo", func(t *testing.T) {
		primary := idTokenClaims{Email: "", EmailVerified: false, Sub: "s1", Name: ""}
		fallback := idTokenClaims{Email: "a@x.com", EmailVerified: true, Sub: "ignored", Name: "Alice"}
		got := mergeUserInfoClaims(primary, fallback)
		assert.Equal(t, "a@x.com", got.Email)
		assert.True(t, bool(got.EmailVerified))
		assert.Equal(t, "s1", got.Sub, "sub must stay from primary")
		assert.Equal(t, "Alice", got.Name)
	})

	t.Run("primary already complete — unchanged", func(t *testing.T) {
		primary := idTokenClaims{Email: "b@x.com", EmailVerified: true, Sub: "s2", Name: "Bob"}
		fallback := idTokenClaims{Email: "other@x.com", EmailVerified: false, Sub: "s3", Name: "Other"}
		got := mergeUserInfoClaims(primary, fallback)
		assert.Equal(t, "b@x.com", got.Email)
		assert.True(t, bool(got.EmailVerified))
		assert.Equal(t, "s2", got.Sub)
		assert.Equal(t, "Bob", got.Name)
	})
}

var _ = errors.Is
