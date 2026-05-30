package billing_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	billingsvc "github.com/novudesk/novudesk/internal/application/billing"
	domainbilling "github.com/novudesk/novudesk/internal/domain/billing"
	"github.com/novudesk/novudesk/internal/domain/organization"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
)

// --- fakes ---

type fakeOrgRepo struct {
	org      *organization.Organization
	lastPlan *organization.PlanUpdate
}

func (f *fakeOrgRepo) Create(context.Context, *organization.Organization) error { return nil }
func (f *fakeOrgRepo) FindByID(context.Context, string) (*organization.Organization, error) {
	return f.org, nil
}
func (f *fakeOrgRepo) FindBySlug(context.Context, string) (*organization.Organization, error) {
	return nil, nil
}
func (f *fakeOrgRepo) Update(context.Context, string, organization.UpdateInput) (*organization.Organization, error) {
	return f.org, nil
}
func (f *fakeOrgRepo) UpdatePlan(_ context.Context, _ string, input organization.PlanUpdate) (*organization.Organization, error) {
	f.lastPlan = &input
	f.org.PlanTier = input.PlanTier
	f.org.PlanRenewsAt = input.RenewsAt
	return f.org, nil
}
func (f *fakeOrgRepo) SlugExists(context.Context, string) (bool, error) { return false, nil }

type fakeSessionRepo struct {
	sessions map[string]*domainbilling.PaymentSession
	pending  *domainbilling.PaymentSession
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{sessions: map[string]*domainbilling.PaymentSession{}}
}

func (f *fakeSessionRepo) Create(_ context.Context, s *domainbilling.PaymentSession) error {
	f.sessions[s.ID] = s
	if s.Status == domainbilling.StatusPending {
		f.pending = s
	}
	return nil
}
func (f *fakeSessionRepo) FindByID(_ context.Context, id, _ string) (*domainbilling.PaymentSession, error) {
	return f.sessions[id], nil
}
func (f *fakeSessionRepo) UpdateStatus(_ context.Context, id, _ string, status domainbilling.SessionStatus, completedAt *time.Time) error {
	if s, ok := f.sessions[id]; ok {
		s.Status = status
		s.CompletedAt = completedAt
		if f.pending != nil && f.pending.ID == id {
			f.pending = nil
		}
	}
	return nil
}
func (f *fakeSessionRepo) ListByOrg(context.Context, string) ([]*domainbilling.PaymentSession, error) {
	out := make([]*domainbilling.PaymentSession, 0, len(f.sessions))
	for _, s := range f.sessions {
		out = append(out, s)
	}
	return out, nil
}
func (f *fakeSessionRepo) FindActivePending(context.Context, string) (*domainbilling.PaymentSession, error) {
	return f.pending, nil
}

const testOrg = "org-1"
const testUser = "user-1"

func appErrStatus(t *testing.T, err error) int {
	t.Helper()
	var e *apperrors.AppError
	if !apperrors.As(err, &e) {
		t.Fatalf("expected *AppError, got %T: %v", err, err)
	}
	return e.HTTPStatus
}

func newSvc(orgs *fakeOrgRepo, sessions *fakeSessionRepo) *billingsvc.Service {
	return billingsvc.NewService(sessions, orgs, billingsvc.NewManualProvider())
}

// --- tests ---

func TestBilling_InitiateChange_InvalidTier_ReturnsBadRequest(t *testing.T) {
	orgs := &fakeOrgRepo{org: &organization.Organization{ID: testOrg, PlanTier: "free"}}
	svc := newSvc(orgs, newFakeSessionRepo())

	_, err := svc.InitiateChange(context.Background(), testOrg, testUser, "platinum")
	if err == nil {
		t.Fatal("expected error for invalid tier, got nil")
	}
	if appErrStatus(t, err) != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", appErrStatus(t, err))
	}
}

func TestBilling_InitiateChange_DoesNotChangePlan(t *testing.T) {
	orgs := &fakeOrgRepo{org: &organization.Organization{ID: testOrg, PlanTier: "free"}}
	sessions := newFakeSessionRepo()
	svc := newSvc(orgs, sessions)

	session, err := svc.InitiateChange(context.Background(), testOrg, testUser, "pro")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.Status != domainbilling.StatusPending {
		t.Errorf("expected pending session, got %s", session.Status)
	}
	if orgs.org.PlanTier != "free" {
		t.Errorf("plan must stay 'free' until confirmation, got %s", orgs.org.PlanTier)
	}
	if orgs.lastPlan != nil {
		t.Error("UpdatePlan must not be called on initiate")
	}
	if session.ProrationCents <= 0 {
		t.Errorf("expected positive proration for upgrade, got %d", session.ProrationCents)
	}
}

func TestBilling_InitiateChange_ExistingPending_ReturnsConflict(t *testing.T) {
	orgs := &fakeOrgRepo{org: &organization.Organization{ID: testOrg, PlanTier: "free"}}
	sessions := newFakeSessionRepo()
	sessions.pending = &domainbilling.PaymentSession{ID: "existing", OrgID: testOrg, Status: domainbilling.StatusPending}
	svc := newSvc(orgs, sessions)

	_, err := svc.InitiateChange(context.Background(), testOrg, testUser, "pro")
	if err == nil {
		t.Fatal("expected conflict, got nil")
	}
	if appErrStatus(t, err) != http.StatusConflict {
		t.Errorf("expected 409, got %d", appErrStatus(t, err))
	}
}

func TestBilling_InitiateChange_SameTier_ReturnsBadRequest(t *testing.T) {
	orgs := &fakeOrgRepo{org: &organization.Organization{ID: testOrg, PlanTier: "pro"}}
	svc := newSvc(orgs, newFakeSessionRepo())

	_, err := svc.InitiateChange(context.Background(), testOrg, testUser, "pro")
	if err == nil {
		t.Fatal("expected bad request, got nil")
	}
	if appErrStatus(t, err) != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", appErrStatus(t, err))
	}
}

func TestBilling_Confirm_AppliesPlan(t *testing.T) {
	orgs := &fakeOrgRepo{org: &organization.Organization{ID: testOrg, PlanTier: "free"}}
	sessions := newFakeSessionRepo()
	svc := newSvc(orgs, sessions)

	session, err := svc.InitiateChange(context.Background(), testOrg, testUser, "pro")
	if err != nil {
		t.Fatalf("initiate failed: %v", err)
	}

	confirmed, err := svc.Confirm(context.Background(), testOrg, session.ID)
	if err != nil {
		t.Fatalf("confirm failed: %v", err)
	}
	if confirmed.Status != domainbilling.StatusCompleted {
		t.Errorf("expected completed, got %s", confirmed.Status)
	}
	if orgs.org.PlanTier != "pro" {
		t.Errorf("expected plan applied as 'pro', got %s", orgs.org.PlanTier)
	}
	if orgs.lastPlan == nil || orgs.lastPlan.RenewsAt == nil {
		t.Error("expected UpdatePlan with a renewal date on confirm")
	}
}

func TestBilling_Cancel_KeepsPlan(t *testing.T) {
	orgs := &fakeOrgRepo{org: &organization.Organization{ID: testOrg, PlanTier: "free"}}
	sessions := newFakeSessionRepo()
	svc := newSvc(orgs, sessions)

	session, err := svc.InitiateChange(context.Background(), testOrg, testUser, "pro")
	if err != nil {
		t.Fatalf("initiate failed: %v", err)
	}

	if err := svc.Cancel(context.Background(), testOrg, session.ID); err != nil {
		t.Fatalf("cancel failed: %v", err)
	}
	if orgs.org.PlanTier != "free" {
		t.Errorf("plan must stay 'free' after cancel, got %s", orgs.org.PlanTier)
	}
	if sessions.sessions[session.ID].Status != domainbilling.StatusCancelled {
		t.Errorf("expected cancelled session, got %s", sessions.sessions[session.ID].Status)
	}
}

func TestBilling_Confirm_NonexistentSession_ReturnsNotFound(t *testing.T) {
	orgs := &fakeOrgRepo{org: &organization.Organization{ID: testOrg, PlanTier: "free"}}
	svc := newSvc(orgs, newFakeSessionRepo())

	_, err := svc.Confirm(context.Background(), testOrg, "missing")
	if err == nil {
		t.Fatal("expected not found, got nil")
	}
	if !apperrors.IsNotFound(err) {
		t.Errorf("expected 404, got %d", appErrStatus(t, err))
	}
}
