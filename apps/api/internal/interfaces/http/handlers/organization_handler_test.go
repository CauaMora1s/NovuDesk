package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	billingsvc "github.com/novudesk/novudesk/internal/application/billing"
	orgsvc "github.com/novudesk/novudesk/internal/application/organization"
	domainbilling "github.com/novudesk/novudesk/internal/domain/billing"
	"github.com/novudesk/novudesk/internal/domain/organization"
	"github.com/novudesk/novudesk/internal/interfaces/http/handlers"
)

// --- fakes ---

type fakeOrgRepo struct {
	org *organization.Organization
}

func (f *fakeOrgRepo) Create(context.Context, *organization.Organization) error { return nil }
func (f *fakeOrgRepo) FindByID(context.Context, string) (*organization.Organization, error) {
	return f.org, nil
}
func (f *fakeOrgRepo) FindBySlug(context.Context, string) (*organization.Organization, error) {
	return nil, nil
}
func (f *fakeOrgRepo) Update(_ context.Context, _ string, input organization.UpdateInput) (*organization.Organization, error) {
	if input.Name != nil {
		f.org.Name = *input.Name
	}
	return f.org, nil
}
func (f *fakeOrgRepo) UpdatePlan(_ context.Context, _ string, input organization.PlanUpdate) (*organization.Organization, error) {
	f.org.PlanTier = input.PlanTier
	f.org.PlanRenewsAt = input.RenewsAt
	return f.org, nil
}
func (f *fakeOrgRepo) SlugExists(context.Context, string) (bool, error) { return false, nil }

type fakeUsageRepo struct {
	usage *organization.Usage
}

func (f *fakeUsageRepo) Snapshot(context.Context, string) (*organization.Usage, error) {
	return f.usage, nil
}

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

// --- helpers ---

func newOrgFixture() *organization.Organization {
	return &organization.Organization{
		ID:        testOrgID,
		Name:      "Acme",
		Slug:      "acme",
		PlanTier:  "free",
		CreatedAt: time.Now(),
	}
}

func newOrgHandler(orgRepo *fakeOrgRepo, usage *fakeUsageRepo, sessions *fakeSessionRepo) *handlers.OrganizationHandler {
	orgService := orgsvc.NewService(orgRepo, nil, nil, usage)
	billingService := billingsvc.NewService(sessions, orgRepo, billingsvc.NewManualProvider())
	return handlers.NewOrganizationHandler(orgService, billingService)
}

// --- tests ---

func TestOrganizationHandler_Get_ReturnsOverview(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: newOrgFixture()}
	usage := &fakeUsageRepo{usage: &organization.Usage{Members: 2, TicketsThisMonth: 10}}
	h := newOrgHandler(orgRepo, usage, newFakeSessionRepo())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organization", nil)
	req = injectClaims(req, fakeClaims())
	rec := httptest.NewRecorder()
	h.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	env := decodeEnvelope(t, rec.Body)
	data, ok := env["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %T", env["data"])
	}
	if data["organization"] == nil || data["plan"] == nil || data["usage"] == nil {
		t.Errorf("overview missing fields: %v", data)
	}
}

func TestOrganizationHandler_Update_RenamesOrg(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: newOrgFixture()}
	h := newOrgHandler(orgRepo, &fakeUsageRepo{usage: &organization.Usage{}}, newFakeSessionRepo())

	body, _ := json.Marshal(map[string]any{"name": "New Name"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/organization", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectClaims(req, fakeClaims())
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if orgRepo.org.Name != "New Name" {
		t.Errorf("expected name updated, got %q", orgRepo.org.Name)
	}
}

func TestOrganizationHandler_Update_InvalidName_Returns422(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: newOrgFixture()}
	h := newOrgHandler(orgRepo, &fakeUsageRepo{usage: &organization.Usage{}}, newFakeSessionRepo())

	body, _ := json.Marshal(map[string]any{"name": "x"}) // too short
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/organization", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectClaims(req, fakeClaims())
	rec := httptest.NewRecorder()
	h.Update(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestOrganizationHandler_ListPlans_ReturnsCatalog(t *testing.T) {
	h := newOrgHandler(&fakeOrgRepo{org: newOrgFixture()}, &fakeUsageRepo{usage: &organization.Usage{}}, newFakeSessionRepo())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organization/plans", nil)
	req = injectClaims(req, fakeClaims())
	rec := httptest.NewRecorder()
	h.ListPlans(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	env := decodeEnvelope(t, rec.Body)
	data, ok := env["data"].([]any)
	if !ok || len(data) != 4 {
		t.Fatalf("expected 4 plans, got %v", env["data"])
	}
}

func TestOrganizationHandler_InitiatePlanChange_CreatesPendingSession(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: newOrgFixture()}
	h := newOrgHandler(orgRepo, &fakeUsageRepo{usage: &organization.Usage{}}, newFakeSessionRepo())

	body, _ := json.Marshal(map[string]any{"to_tier": "pro"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organization/plan/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectClaims(req, fakeClaims())
	rec := httptest.NewRecorder()
	h.InitiatePlanChange(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if orgRepo.org.PlanTier != "free" {
		t.Errorf("plan must stay 'free' until confirmation, got %s", orgRepo.org.PlanTier)
	}
}

func TestOrganizationHandler_InitiatePlanChange_InvalidTier_Returns400(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: newOrgFixture()}
	h := newOrgHandler(orgRepo, &fakeUsageRepo{usage: &organization.Usage{}}, newFakeSessionRepo())

	body, _ := json.Marshal(map[string]any{"to_tier": "platinum"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organization/plan/sessions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectClaims(req, fakeClaims())
	rec := httptest.NewRecorder()
	h.InitiatePlanChange(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestOrganizationHandler_ConfirmPlanChange_AppliesPlan(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: newOrgFixture()}
	sessions := newFakeSessionRepo()
	billingService := billingsvc.NewService(sessions, orgRepo, billingsvc.NewManualProvider())
	session, err := billingService.InitiateChange(context.Background(), testOrgID, testUserID, "pro")
	if err != nil {
		t.Fatalf("setup initiate failed: %v", err)
	}
	h := handlers.NewOrganizationHandler(orgsvc.NewService(orgRepo, nil, nil, &fakeUsageRepo{usage: &organization.Usage{}}), billingService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organization/plan/sessions/"+session.ID+"/confirm", nil)
	req = injectClaims(req, fakeClaims())
	rec := httptest.NewRecorder()
	routeWith(http.MethodPost, "/api/v1/organization/plan/sessions/{id}/confirm", h.ConfirmPlanChange).
		ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if orgRepo.org.PlanTier != "pro" {
		t.Errorf("expected plan applied as 'pro', got %s", orgRepo.org.PlanTier)
	}
}

func TestOrganizationHandler_CancelPlanChange_KeepsPlan(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: newOrgFixture()}
	sessions := newFakeSessionRepo()
	billingService := billingsvc.NewService(sessions, orgRepo, billingsvc.NewManualProvider())
	session, err := billingService.InitiateChange(context.Background(), testOrgID, testUserID, "pro")
	if err != nil {
		t.Fatalf("setup initiate failed: %v", err)
	}
	h := handlers.NewOrganizationHandler(orgsvc.NewService(orgRepo, nil, nil, &fakeUsageRepo{usage: &organization.Usage{}}), billingService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/organization/plan/sessions/"+session.ID+"/cancel", nil)
	req = injectClaims(req, fakeClaims())
	rec := httptest.NewRecorder()
	routeWith(http.MethodPost, "/api/v1/organization/plan/sessions/{id}/cancel", h.CancelPlanChange).
		ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d — body: %s", rec.Code, rec.Body.String())
	}
	if orgRepo.org.PlanTier != "free" {
		t.Errorf("plan must stay 'free' after cancel, got %s", orgRepo.org.PlanTier)
	}
}

func TestOrganizationHandler_ListSessions_ReturnsHistory(t *testing.T) {
	orgRepo := &fakeOrgRepo{org: newOrgFixture()}
	sessions := newFakeSessionRepo()
	sessions.sessions["s1"] = &domainbilling.PaymentSession{ID: "s1", OrgID: testOrgID, ToTier: "pro", Status: domainbilling.StatusCompleted}
	h := newOrgHandler(orgRepo, &fakeUsageRepo{usage: &organization.Usage{}}, sessions)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/organization/plan/sessions", nil)
	req = injectClaims(req, fakeClaims())
	rec := httptest.NewRecorder()
	h.ListSessions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	env := decodeEnvelope(t, rec.Body)
	data, ok := env["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("expected 1 session, got %v", env["data"])
	}
}
