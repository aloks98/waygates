package service

import (
	"errors"
	"fmt"

	"go.uber.org/zap"

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
)

// ErrBaseDomainRequiredByMembers is re-exported from the repository so handlers
// map a single error to 409 without importing the repository package.
var ErrBaseDomainRequiredByMembers = repository.ErrBaseDomainRequiredByMembers

func (s *ProxyGroupService) ListGroups(params repository.ProxyGroupListParams) ([]models.ProxyGroup, int64, error) {
	return s.repo.List(params)
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
