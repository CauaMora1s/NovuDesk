package billing

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainbilling "github.com/novudesk/novudesk/internal/domain/billing"
	"github.com/novudesk/novudesk/internal/domain/organization"
	"github.com/novudesk/novudesk/internal/domain/plan"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
)

const (
	billingCycleDays  = 30
	sessionTTLMinutes = 60
)

// Service orchestrates the plan-change payment session lifecycle. The
// organization's active plan only changes once a session is confirmed.
type Service struct {
	sessions domainbilling.SessionRepository
	orgs     organization.Repository
	provider Provider
}

func NewService(sessions domainbilling.SessionRepository, orgs organization.Repository, provider Provider) *Service {
	return &Service{sessions: sessions, orgs: orgs, provider: provider}
}

// InitiateChange creates a pending payment session for switching to toTier. It
// does NOT change the active plan. Returns an error if a pending session
// already exists or the target tier is invalid.
func (s *Service) InitiateChange(ctx context.Context, orgID, userID, toTier string) (*domainbilling.PaymentSession, error) {
	if !plan.IsValidTier(toTier) {
		return nil, apperrors.New(apperrors.CodeInvalidPlan, "unknown plan tier", 400)
	}

	org, err := s.orgs.FindByID(ctx, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if org == nil {
		return nil, apperrors.NotFound(apperrors.CodeOrgNotFound, "organization not found")
	}
	if org.PlanTier == toTier {
		return nil, apperrors.BadRequest("organization is already on this plan")
	}

	pending, err := s.sessions.FindActivePending(ctx, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if pending != nil {
		return nil, apperrors.Conflict(apperrors.CodeConflict, "a pending plan change already exists")
	}

	fromPlan, _ := plan.ByTier(plan.Tier(org.PlanTier))
	toPlan, _ := plan.ByTier(plan.Tier(toTier))

	daysRemaining := billingCycleDays
	if org.PlanRenewsAt != nil {
		daysRemaining = int(time.Until(*org.PlanRenewsAt).Hours() / 24)
	}
	proration := plan.Proration(fromPlan, toPlan, daysRemaining, billingCycleDays)

	fromTier := org.PlanTier
	expiresAt := time.Now().Add(sessionTTLMinutes * time.Minute)
	var createdBy *string
	if userID != "" {
		createdBy = &userID
	}

	session := &domainbilling.PaymentSession{
		ID:             uuid.NewString(),
		OrgID:          orgID,
		FromTier:       &fromTier,
		ToTier:         toTier,
		Status:         domainbilling.StatusPending,
		AmountCents:    toPlan.PriceCents,
		ProrationCents: proration,
		Currency:       toPlan.Currency,
		Provider:       s.provider.Name(),
		CreatedBy:      createdBy,
		ExpiresAt:      &expiresAt,
	}

	checkout, err := s.provider.CreateCheckout(ctx, session)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if checkout.ProviderRef != "" {
		session.ProviderRef = &checkout.ProviderRef
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, apperrors.Internal(err)
	}
	return session, nil
}

// Confirm marks a pending session completed and applies the plan change to the
// organization. With a real gateway this would be driven by a webhook.
func (s *Service) Confirm(ctx context.Context, orgID, sessionID string) (*domainbilling.PaymentSession, error) {
	session, err := s.getPending(ctx, orgID, sessionID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	renewsAt := now.AddDate(0, 1, 0)
	if _, err := s.orgs.UpdatePlan(ctx, orgID, organization.PlanUpdate{PlanTier: session.ToTier, RenewsAt: &renewsAt}); err != nil {
		return nil, apperrors.Internal(err)
	}

	if err := s.sessions.UpdateStatus(ctx, sessionID, orgID, domainbilling.StatusCompleted, &now); err != nil {
		return nil, apperrors.Internal(err)
	}

	session.Status = domainbilling.StatusCompleted
	session.CompletedAt = &now
	return session, nil
}

// Cancel marks a pending session cancelled without changing the plan.
func (s *Service) Cancel(ctx context.Context, orgID, sessionID string) error {
	if _, err := s.getPending(ctx, orgID, sessionID); err != nil {
		return err
	}
	if err := s.sessions.UpdateStatus(ctx, sessionID, orgID, domainbilling.StatusCancelled, nil); err != nil {
		return apperrors.Internal(err)
	}
	return nil
}

// History returns all payment sessions for the organization, newest first.
func (s *Service) History(ctx context.Context, orgID string) ([]*domainbilling.PaymentSession, error) {
	list, err := s.sessions.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return list, nil
}

// ActivePending returns the org's pending session, or nil.
func (s *Service) ActivePending(ctx context.Context, orgID string) (*domainbilling.PaymentSession, error) {
	p, err := s.sessions.FindActivePending(ctx, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	return p, nil
}

func (s *Service) getPending(ctx context.Context, orgID, sessionID string) (*domainbilling.PaymentSession, error) {
	session, err := s.sessions.FindByID(ctx, sessionID, orgID)
	if err != nil {
		return nil, apperrors.Internal(err)
	}
	if session == nil {
		return nil, apperrors.NotFound(apperrors.CodeSessionNotFound, "payment session not found")
	}
	if session.Status != domainbilling.StatusPending {
		return nil, apperrors.Conflict(apperrors.CodeConflict, "payment session is not pending")
	}
	return session, nil
}
