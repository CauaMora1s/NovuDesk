package quota

import (
	"context"
	"fmt"

	"github.com/novudesk/novudesk/internal/domain/organization"
	"github.com/novudesk/novudesk/internal/domain/plan"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
)

// Resource identifies a plan-limited resource.
type Resource string

const (
	ResourceSeats      Resource = "seats"
	ResourceTickets    Resource = "tickets"
	ResourceStorage    Resource = "storage"
	ResourceTeams      Resource = "teams"
	ResourceCategories Resource = "categories"
	ResourceAPIKeys    Resource = "api_keys"
)

// Service enforces plan usage limits.
type Service struct {
	orgs  organization.Repository
	usage organization.UsageRepository
}

func NewService(orgs organization.Repository, usage organization.UsageRepository) *Service {
	return &Service{orgs: orgs, usage: usage}
}

// EnsureWithinLimit checks that adding `additional` units of the given resource
// keeps the organization within its plan limit. For count resources pass 1; for
// storage pass the byte size being added. Unlimited limits always pass.
func (s *Service) EnsureWithinLimit(ctx context.Context, orgID string, resource Resource, additional int64) error {
	org, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return apperrors.Internal(err)
	}
	if org == nil {
		return apperrors.NotFound(apperrors.CodeOrgNotFound, "organization not found")
	}

	p, ok := plan.ByTier(plan.Tier(org.PlanTier))
	if !ok {
		// Unknown tier: treat as no enforcement rather than blocking the org.
		return nil
	}

	limit, current, err := s.resolve(ctx, orgID, resource, p.Limits)
	if err != nil {
		return err
	}

	if plan.IsUnlimited(limit) {
		return nil
	}
	if current+additional > limit {
		return apperrors.Conflict(apperrors.CodeQuotaExceeded,
			fmt.Sprintf("plan limit reached for %s (%d/%d)", resource, current, limit))
	}
	return nil
}

func (s *Service) resolve(ctx context.Context, orgID string, resource Resource, limits plan.Limits) (limit, current int64, err error) {
	u, err := s.usage.Snapshot(ctx, orgID)
	if err != nil {
		return 0, 0, apperrors.Internal(err)
	}
	switch resource {
	case ResourceSeats:
		return limits.Seats, u.Members, nil
	case ResourceTickets:
		return limits.TicketsPerMonth, u.TicketsThisMonth, nil
	case ResourceStorage:
		return limits.StorageBytes, u.StorageBytes, nil
	case ResourceTeams:
		return limits.Teams, u.Teams, nil
	case ResourceCategories:
		return limits.Categories, u.Categories, nil
	case ResourceAPIKeys:
		return limits.APIKeys, u.APIKeys, nil
	default:
		return plan.Unlimited, 0, nil
	}
}
