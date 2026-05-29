package sla_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	slasvc "github.com/novudesk/novudesk/internal/application/sla"
	"github.com/novudesk/novudesk/internal/domain/sla"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
)

// --- fake repository ---

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

const testOrg = "org-abc"

func newPolicy(id, orgID, categoryID string) *sla.Policy {
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

func appErrStatus(t *testing.T, err error) int {
	t.Helper()
	var e *apperrors.AppError
	if !apperrors.As(err, &e) {
		t.Fatalf("expected *AppError, got %T: %v", err, err)
	}
	return e.HTTPStatus
}

// --- tests ---

func TestSLAService_Delete_WhenNotFound_ReturnsNotFound(t *testing.T) {
	repo := &fakeSLARepo{}
	svc := slasvc.NewService(repo)

	err := svc.Delete(context.Background(), "nonexistent", testOrg)
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}
	if !apperrors.IsNotFound(err) {
		t.Errorf("expected 404 NotFound, got status %d", appErrStatus(t, err))
	}
}

func TestSLAService_Delete_WhenExists_ReturnsNil(t *testing.T) {
	repo := &fakeSLARepo{
		policies: []*sla.Policy{newPolicy("sla-1", testOrg, "cat-1")},
	}
	svc := slasvc.NewService(repo)

	err := svc.Delete(context.Background(), "sla-1", testOrg)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(repo.policies) != 0 {
		t.Errorf("expected policy to be removed, got %d remaining", len(repo.policies))
	}
}

func TestSLAService_Delete_WhenRepoFails_ReturnsInternal(t *testing.T) {
	repo := &fakeSLARepo{
		policies: []*sla.Policy{newPolicy("sla-1", testOrg, "cat-1")},
		err:      errors.New("db error"),
	}
	svc := slasvc.NewService(repo)

	err := svc.Delete(context.Background(), "sla-1", testOrg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if appErrStatus(t, err) != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", appErrStatus(t, err))
	}
}

func TestSLAService_UpsertForCategory_WhenNew_CreatesPolicy(t *testing.T) {
	repo := &fakeSLARepo{}
	svc := slasvc.NewService(repo)

	p, err := svc.UpsertForCategory(context.Background(), testOrg, "cat-1", sla.CreateInput{
		ResolutionValue: 48,
		ResolutionUnit:  "hours",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected policy, got nil")
	}
	if p.ResolutionValue != 48 {
		t.Errorf("expected resolution_value=48, got %d", p.ResolutionValue)
	}
	if len(repo.policies) != 1 {
		t.Errorf("expected 1 policy stored, got %d", len(repo.policies))
	}
}

func TestSLAService_UpsertForCategory_WhenExists_UpdatesPolicy(t *testing.T) {
	existing := newPolicy("sla-1", testOrg, "cat-1")
	repo := &fakeSLARepo{policies: []*sla.Policy{existing}}
	svc := slasvc.NewService(repo)

	p, err := svc.UpsertForCategory(context.Background(), testOrg, "cat-1", sla.CreateInput{
		ResolutionValue: 7,
		ResolutionUnit:  "days",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected updated policy, got nil")
	}
	if p.ResolutionValue != 7 || p.ResolutionUnit != "days" {
		t.Errorf("expected resolution 7 days, got %d %s", p.ResolutionValue, p.ResolutionUnit)
	}
}

func TestSLAService_ListWithCategoryStats_ReturnsStats(t *testing.T) {
	slaID := "sla-1"
	resVal := 24
	resUnit := "hours"
	resHours := 24
	repo := &fakeSLARepo{
		stats: []*sla.CategorySLAStat{
			{
				CategoryID:   "cat-1",
				CategoryName: "Infrastructure",
				SLAID:        &slaID,
				ResolutionValue: &resVal,
				ResolutionUnit:  &resUnit,
				ResolutionHours: &resHours,
			},
		},
	}
	svc := slasvc.NewService(repo)

	stats, err := svc.ListWithCategoryStats(context.Background(), testOrg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stats) != 1 {
		t.Fatalf("expected 1 stat, got %d", len(stats))
	}
	if stats[0].CategoryName != "Infrastructure" {
		t.Errorf("unexpected category name: %s", stats[0].CategoryName)
	}
}

func TestSLAService_ListWithCategoryStats_WhenRepoFails_ReturnsInternal(t *testing.T) {
	repo := &fakeSLARepo{err: errors.New("db error")}
	svc := slasvc.NewService(repo)

	_, err := svc.ListWithCategoryStats(context.Background(), testOrg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if appErrStatus(t, err) != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", appErrStatus(t, err))
	}
}
