package service

import (
	"errors"
	"fmt"
	"math"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/aloks98/waygates/backend/internal/models"
	"github.com/aloks98/waygates/backend/internal/repository"
)

// GroupSyncer rebuilds the entire Caddy config. A group mutation can change the
// effective config of every member at once, so there is no per-proxy sync path
// worth having: buildConfigBytes reconstructs the whole config anyway.
type GroupSyncer interface {
	RebuildAll() error
}

type ProxyGroupService struct {
	repo   repository.ProxyGroupRepositoryInterface
	syncer GroupSyncer
	logger *zap.Logger
}

type ProxyGroupServiceConfig struct {
	Repo   repository.ProxyGroupRepositoryInterface
	Syncer GroupSyncer
	Logger *zap.Logger
}

func NewProxyGroupService(cfg ProxyGroupServiceConfig) *ProxyGroupService {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	return &ProxyGroupService{
		repo:   cfg.Repo,
		syncer: cfg.Syncer,
		logger: cfg.Logger.Named("proxy-group-service"),
	}
}

var (
	ErrGroupNotFound     = errors.New("proxy group not found")
	ErrGroupNameConflict = errors.New("proxy group name already exists")
	ErrGroupHasMembers   = errors.New("proxy group has member proxies")

	// ErrGroupACLAssignmentExists / ErrGroupACLAssignmentNotFound mirror
	// ErrProxyACLExists / ErrProxyACLNotFound (acl_service.go) for the nested
	// proxy-group ACL assignment routes.
	ErrGroupACLAssignmentExists   = errors.New("this ACL group is already assigned to this proxy group")
	ErrGroupACLAssignmentNotFound = errors.New("proxy group ACL assignment not found")
)

// ErrBaseDomainRequiredByMembers is re-exported from the repository so handlers
// map a single error to 409 without importing the repository package.
var ErrBaseDomainRequiredByMembers = repository.ErrBaseDomainRequiredByMembers

const (
	proxyGroupsNameConstraint    = "uq_proxy_groups_name"
	groupACLAssignmentConstraint = "uq_pgaa_group_acl"

	// groupACLAssignmentACLGroupConstraint is the FK constraint name on
	// proxy_group_acl_assignments.acl_group_id
	// (migrations/000014_create_proxy_groups.up.sql). Matched against a create
	// failure to classify a bogus acl_group_id as ErrACLGroupNotFound instead
	// of a raw 500 — see AssignACLToGroup. Unlike ACLService.AssignToProxy,
	// ProxyGroupService has no ACL-repo dependency to pre-check the ACL group
	// with, so this FK check is the only guard, not defense-in-depth.
	groupACLAssignmentACLGroupConstraint = "fk_pgaa_acl_group"
)

// ListGroups returns a paginated page of groups. Page/Limit are clamped to
// sane defaults the same way ProxyService.ListProxies clamps them, so the
// zero-value request from an omitted page/limit query param doesn't turn
// into a GORM `LIMIT 0` (zero rows, always).
func (s *ProxyGroupService) ListGroups(params repository.ProxyGroupListParams) (*models.ProxyGroupListResponse, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 20
	}

	groups, total, err := s.repo.List(params)
	if err != nil {
		return nil, fmt.Errorf("failed to list proxy groups: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.Limit)))
	return &models.ProxyGroupListResponse{
		Items:      groups,
		Total:      total,
		Page:       params.Page,
		Limit:      params.Limit,
		TotalPages: totalPages,
	}, nil
}

// ListMembers returns the member proxies of a group. Used by the handler to
// determine which proxy IDs are affected by a base_domain rewrite, for audit
// logging.
func (s *ProxyGroupService) ListMembers(id int) ([]models.Proxy, error) {
	return s.repo.ListMembers(id)
}

func (s *ProxyGroupService) GetGroupByID(id int) (*models.ProxyGroup, error) {
	g, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrGroupNotFound
	}
	return g, nil
}

func (s *ProxyGroupService) CreateGroup(g *models.ProxyGroup, userID int) error {
	g.CreatedBy = userID
	if err := s.repo.Create(g); err != nil {
		if repository.IsUniqueViolation(err, proxyGroupsNameConstraint) {
			return ErrGroupNameConflict
		}
		return fmt.Errorf("failed to create proxy group: %w", err)
	}
	if err := s.syncer.RebuildAll(); err != nil {
		if delErr := s.repo.Delete(g.ID); delErr != nil {
			return fmt.Errorf("failed to sync and rollback failed: %w", errors.Join(err, delErr))
		}
		return fmt.Errorf("failed to sync proxy group: %w", err)
	}
	return nil
}

// UpdateGroup writes the group's settings and, when base_domain changes,
// re-homes its label-addressed members' materialized hostnames — both in one
// transaction (see ProxyGroupRepository.UpdateGroupTx). The config is rebuilt
// only after a successful write — a failed update must leave the served
// config, and the database, exactly as it was.
func (s *ProxyGroupService) UpdateGroup(g *models.ProxyGroup) error {
	existing, err := s.repo.GetByID(g.ID)
	if err != nil {
		return ErrGroupNotFound
	}

	changed := baseDomainChanged(existing.BaseDomain, g.BaseDomain)
	if err := s.repo.UpdateGroupTx(g, changed); err != nil {
		if repository.IsUniqueViolation(err, proxyGroupsNameConstraint) {
			return ErrGroupNameConflict
		}
		var hostErr *repository.HostnameConflictError
		if errors.As(err, &hostErr) {
			return fmt.Errorf("%w: %s", ErrHostnameConflict, hostErr.Hostname)
		}
		return fmt.Errorf("failed to update proxy group: %w", err)
	}

	if changed {
		s.logger.Info("proxy group base domain changed",
			zap.Int("group_id", g.ID),
			zap.Stringp("old", existing.BaseDomain),
			zap.Stringp("new", g.BaseDomain))
	}

	return s.syncer.RebuildAll()
}

func (s *ProxyGroupService) DeleteGroup(id int) error {
	n, err := s.repo.MemberCount(id)
	if err != nil {
		return fmt.Errorf("failed to count members: %w", err)
	}
	if n > 0 {
		return fmt.Errorf("%w: %d member proxies; reassign or remove them first", ErrGroupHasMembers, n)
	}
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete proxy group: %w", err)
	}
	return s.syncer.RebuildAll()
}

func baseDomainChanged(old, next *string) bool {
	switch {
	case old == nil && next == nil:
		return false
	case old == nil || next == nil:
		return true
	default:
		return *old != *next
	}
}

// =============================================================================
// ACL Assignment (nested under a proxy group)
//
// These mirror ACLService's proxy-assignment methods (AssignToProxy /
// UpdateProxyAssignment / RemoveFromProxy / GetProxyACL in acl_service.go),
// reusing its validatePathPattern/ErrInvalidPathPattern rather than
// duplicating them — both packages live in `service`. Every mutation calls
// RebuildAll(): a group's ACL change alters what Caddy enforces for every
// member proxy at once, so there is no per-proxy sync path worth having.
// =============================================================================

// ListACLAssignments returns the ACL assignments for a proxy group.
func (s *ProxyGroupService) ListACLAssignments(groupID int) ([]models.ProxyGroupACLAssignment, error) {
	if _, err := s.repo.GetByID(groupID); err != nil {
		return nil, ErrGroupNotFound
	}
	assignments, err := s.repo.ListACLAssignments(groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to list proxy group ACL assignments: %w", err)
	}
	return assignments, nil
}

// AssignACLToGroup assigns an ACL group to a proxy group. A create can be
// rolled back by deleting the row it just inserted, so — like CreateGroup —
// a sync failure undoes the write rather than leaving a DB row that isn't
// reflected in the served config.
func (s *ProxyGroupService) AssignACLToGroup(groupID, aclGroupID int, pathPattern string, priority int, enabled bool) error {
	if _, err := s.repo.GetByID(groupID); err != nil {
		return ErrGroupNotFound
	}
	if err := validatePathPattern(pathPattern); err != nil {
		return err
	}
	if pathPattern == "" {
		pathPattern = "/*"
	}

	a := &models.ProxyGroupACLAssignment{
		ProxyGroupID: groupID,
		ACLGroupID:   aclGroupID,
		PathPattern:  pathPattern,
		Priority:     priority,
		Enabled:      enabled,
	}
	if err := s.repo.CreateACLAssignment(a); err != nil {
		if repository.IsUniqueViolation(err, groupACLAssignmentConstraint) {
			return ErrGroupACLAssignmentExists
		}
		if repository.IsForeignKeyViolation(err, groupACLAssignmentACLGroupConstraint) {
			return ErrACLGroupNotFound
		}
		return fmt.Errorf("failed to assign ACL to proxy group: %w", err)
	}

	// Create rolls back on sync failure because it can: the row it just
	// inserted can be cleanly deleted, leaving the database exactly as it was
	// before the request (mirrors ProxyService.CreateProxy's convention).
	// UpdateGroupACLAssignment and RemoveACLFromGroup below do NOT roll back —
	// see the comments there for why that asymmetry is intentional.
	if err := s.syncer.RebuildAll(); err != nil {
		if delErr := s.repo.DeleteACLAssignment(groupID, aclGroupID); delErr != nil {
			return fmt.Errorf("failed to sync and rollback failed: %w", errors.Join(err, delErr))
		}
		return fmt.Errorf("failed to sync proxy group ACL assignment: %w", err)
	}
	return nil
}

// UpdateGroupACLAssignment updates an existing assignment. assignmentID must
// belong to groupID — this ownership check mirrors
// ProxyACLHandler.UpdateProxyACLAssignment's: without it, a caller could
// modify another group's assignment through this group's route.
//
// Update/Delete have no rollback-on-sync-failure step, unlike Create: there is
// nothing to symmetrically undo (re-applying the previous values, or
// recreating a deleted row, would be inventing a new pattern UpdateGroup and
// DeleteGroup don't use either) — cleanly "un-updating" is fragile and worse
// than relying on the 60-second periodic sync (SyncService) to reconcile.
// The sync error is still logged and propagated (not swallowed): the caller
// gets a 500 telling it the write may not be live yet, and the periodic sync
// fixes the served config regardless.
func (s *ProxyGroupService) UpdateGroupACLAssignment(groupID, assignmentID int, pathPattern string, priority int, enabled bool) error {
	// Empty pathPattern is intentionally passed through unvalidated and
	// unchanged, mirroring ACLService.UpdateProxyAssignment's exact behavior.
	if pathPattern != "" {
		if err := validatePathPattern(pathPattern); err != nil {
			return err
		}
	}

	assignments, err := s.repo.ListACLAssignments(groupID)
	if err != nil {
		return fmt.Errorf("failed to list proxy group ACL assignments: %w", err)
	}
	owned := false
	for i := range assignments {
		if assignments[i].ID == assignmentID {
			owned = true
			break
		}
	}
	if !owned {
		return ErrGroupACLAssignmentNotFound
	}

	a := &models.ProxyGroupACLAssignment{ID: assignmentID, PathPattern: pathPattern, Priority: priority, Enabled: enabled}
	if err := s.repo.UpdateACLAssignment(a); err != nil {
		return fmt.Errorf("failed to update proxy group ACL assignment: %w", err)
	}
	if err := s.syncer.RebuildAll(); err != nil {
		s.logger.Error("failed to rebuild caddy config after updating proxy group ACL assignment",
			zap.Int("group_id", groupID), zap.Int("assignment_id", assignmentID), zap.Error(err))
		return fmt.Errorf("failed to sync proxy group ACL assignment update: %w", err)
	}
	return nil
}

// RemoveACLFromGroup removes an ACL group assignment from a proxy group. If
// no assignment matches (groupID, aclGroupID) — a bogus aclGroupID, or one
// already removed — it returns ErrGroupACLAssignmentNotFound without
// rebuilding the config or letting the caller log an audit event for a
// deletion that never happened.
//
// Like UpdateGroupACLAssignment above, a successful delete has no
// rollback-on-sync-failure step: the row is already gone, so there is nothing
// to symmetrically undo. The sync error is logged and propagated; the
// periodic sync (SyncService) reconciles the served config regardless.
func (s *ProxyGroupService) RemoveACLFromGroup(groupID, aclGroupID int) error {
	if err := s.repo.DeleteACLAssignment(groupID, aclGroupID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrGroupACLAssignmentNotFound
		}
		return fmt.Errorf("failed to remove ACL from proxy group: %w", err)
	}
	if err := s.syncer.RebuildAll(); err != nil {
		s.logger.Error("failed to rebuild caddy config after removing ACL from proxy group",
			zap.Int("group_id", groupID), zap.Int("acl_group_id", aclGroupID), zap.Error(err))
		return fmt.Errorf("failed to sync proxy group ACL removal: %w", err)
	}
	return nil
}
