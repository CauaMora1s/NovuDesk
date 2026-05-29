package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	slasvc "github.com/novudesk/novudesk/internal/application/sla"
	"github.com/novudesk/novudesk/internal/domain/sla"
	"github.com/novudesk/novudesk/internal/interfaces/http/handlers"
)

// --- fake SLA repository ---

type fakeSLARepo struct {
	policies []*sla.Policy
	stats    []*sla.CategorySLAStat
	err      error
}

func (f *fakeSLARepo) Create(_ context.Context, p *sla.Policy) error {
	if f.err != nil {
		return f.err
	}
	f.policies = append(f.policies, p)
	return nil
}

func (f *fakeSLARepo) FindByID(_ context.Context, id, orgID string) (*sla.Policy, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, p := range f.policies {
		if p.ID == id && p.OrgID == orgID {
			return p, nil
		}
	}
	return nil, nil
}

func (f *fakeSLARepo) FindByCategoryID(_ context.Context, categoryID, orgID string) (*sla.Policy, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, p := range f.policies {
		if p.CategoryID != nil && *p.CategoryID == categoryID && p.OrgID == orgID {
			return p, nil
		}
	}
	return nil, nil
}

func (f *fakeSLARepo) ListByOrg(_ context.Context, orgID string) ([]*sla.Policy, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []*sla.Policy
	for _, p := range f.policies {
		if p.OrgID == orgID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakeSLARepo) ListWithCategoryStats(_ context.Context, _ string) ([]*sla.CategorySLAStat, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.stats, nil
}

func (f *fakeSLARepo) Update(_ context.Context, id, orgID string, input sla.CreateInput) (*sla.Policy, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, p := range f.policies {
		if p.ID == id && p.OrgID == orgID {
			p.ResolutionValue = input.ResolutionValue
			p.ResolutionUnit = input.ResolutionUnit
			p.ResponseHours = input.ResponseHours
			return p, nil
		}
	}
	return nil, nil
}

func (f *fakeSLARepo) Delete(_ context.Context, id, orgID string) error {
	if f.err != nil {
		return f.err
	}
	for i, p := range f.policies {
		if p.ID == id && p.OrgID == orgID {
			f.policies = append(f.policies[:i], f.policies[i+1:]...)
			return nil
		}
	}
	return nil
}

// --- helpers ---

func newSLAHandler(repo sla.Repository) *handlers.SLAHandler {
	return handlers.NewSLAHandler(slasvc.NewService(repo))
}

func newSLAPolicy(id, orgID, categoryID string) *sla.Policy {
	catID := categoryID
	return &sla.Policy{
		ID:              id,
		OrgID:           orgID,
		Name:            "SLA",
		CategoryID:      &catID,
		ResolutionValue: 24,
		ResolutionUnit:  "hours",
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

// --- tests ---

func TestSLAHandler_List_ReturnsOk(t *testing.T) {
	slaID := "sla-1"
	resVal := 24
	resUnit := "hours"
	resHours := 24
	repo := &fakeSLARepo{
		stats: []*sla.CategorySLAStat{
			{
				CategoryID:      "cat-1",
				CategoryName:    "Infrastructure",
				SLAID:           &slaID,
				ResolutionValue: &resVal,
				ResolutionUnit:  &resUnit,
				ResolutionHours: &resHours,
			},
		},
	}
	h := newSLAHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sla-policies", nil)
	req = injectClaims(req, fakeClaims())
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}

	env := decodeEnvelope(t, rec.Body)
	data, ok := env["data"].([]any)
	if !ok {
		t.Fatalf("expected data array, got %T", env["data"])
	}
	if len(data) != 1 {
		t.Fatalf("expected 1 stat, got %d", len(data))
	}
}

func TestSLAHandler_UpsertForCategory_ValidInput_ReturnsOk(t *testing.T) {
	repo := &fakeSLARepo{}
	h := newSLAHandler(repo)

	body, _ := json.Marshal(map[string]any{
		"resolution_value": 48,
		"resolution_unit":  "hours",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/sla-policies/category/cat-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectClaims(req, fakeClaims())
	rec := httptest.NewRecorder()
	routeWith(http.MethodPut, "/api/v1/sla-policies/category/{categoryId}", h.UpsertForCategory).
		ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestSLAHandler_UpsertForCategory_MissingFields_Returns422(t *testing.T) {
	repo := &fakeSLARepo{}
	h := newSLAHandler(repo)

	body, _ := json.Marshal(map[string]any{
		"resolution_unit": "hours",
		// missing resolution_value
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/sla-policies/category/cat-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectClaims(req, fakeClaims())
	rec := httptest.NewRecorder()
	routeWith(http.MethodPut, "/api/v1/sla-policies/category/{categoryId}", h.UpsertForCategory).
		ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestSLAHandler_UpsertForCategory_InvalidUnit_Returns422(t *testing.T) {
	repo := &fakeSLARepo{}
	h := newSLAHandler(repo)

	body, _ := json.Marshal(map[string]any{
		"resolution_value": 5,
		"resolution_unit":  "months", // invalid
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/sla-policies/category/cat-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = injectClaims(req, fakeClaims())
	rec := httptest.NewRecorder()
	routeWith(http.MethodPut, "/api/v1/sla-policies/category/{categoryId}", h.UpsertForCategory).
		ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestSLAHandler_Delete_WhenNotFound_Returns404(t *testing.T) {
	repo := &fakeSLARepo{}
	h := newSLAHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sla-policies/nonexistent", nil)
	req = injectClaims(req, fakeClaims())
	rec := httptest.NewRecorder()
	routeWith(http.MethodDelete, "/api/v1/sla-policies/{id}", h.Delete).
		ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d — body: %s", rec.Code, rec.Body.String())
	}
}

func TestSLAHandler_Delete_WhenExists_Returns204(t *testing.T) {
	repo := &fakeSLARepo{
		policies: []*sla.Policy{newSLAPolicy("sla-1", testOrgID, "cat-1")},
	}
	h := newSLAHandler(repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sla-policies/sla-1", nil)
	req = injectClaims(req, fakeClaims())
	rec := httptest.NewRecorder()
	routeWith(http.MethodDelete, "/api/v1/sla-policies/{id}", h.Delete).
		ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d — body: %s", rec.Code, rec.Body.String())
	}
}
