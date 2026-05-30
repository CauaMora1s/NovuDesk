package quota_test

import (
	"context"
	"net/http"
	"testing"

	quotasvc "github.com/novudesk/novudesk/internal/application/quota"
	"github.com/novudesk/novudesk/internal/domain/organization"
	apperrors "github.com/novudesk/novudesk/pkg/errors"
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
func (f *fakeOrgRepo) Update(context.Context, string, organization.UpdateInput) (*organization.Organization, error) {
	return f.org, nil
}
func (f *fakeOrgRepo) UpdatePlan(context.Context, string, organization.PlanUpdate) (*organization.Organization, error) {
	return f.org, nil
}
func (f *fakeOrgRepo) SlugExists(context.Context, string) (bool, error) { return false, nil }

type fakeUsageRepo struct {
	usage *organization.Usage
}

func (f *fakeUsageRepo) Snapshot(context.Context, string) (*organization.Usage, error) {
	return f.usage, nil
}

const testOrg = "org-1"

func appErrStatus(t *testing.T, err error) int {
	t.Helper()
	var e *apperrors.AppError
	if !apperrors.As(err, &e) {
		t.Fatalf("expected *AppError, got %T: %v", err, err)
	}
	return e.HTTPStatus
}

// --- tests ---

func TestQuota_WithinLimit_Passes(t *testing.T) {
	// Free plan allows 3 seats; 2 used + 1 = 3, within limit.
	orgs := &fakeOrgRepo{org: &organization.Organization{ID: testOrg, PlanTier: "free"}}
	usage := &fakeUsageRepo{usage: &organization.Usage{Members: 2}}
	svc := quotasvc.NewService(orgs, usage)

	if err := svc.EnsureWithinLimit(context.Background(), testOrg, quotasvc.ResourceSeats, 1); err != nil {
		t.Fatalf("expected within limit, got error: %v", err)
	}
}

func TestQuota_ExceedingLimit_ReturnsConflict(t *testing.T) {
	// Free plan allows 3 seats; 3 used + 1 = 4, exceeds.
	orgs := &fakeOrgRepo{org: &organization.Organization{ID: testOrg, PlanTier: "free"}}
	usage := &fakeUsageRepo{usage: &organization.Usage{Members: 3}}
	svc := quotasvc.NewService(orgs, usage)

	err := svc.EnsureWithinLimit(context.Background(), testOrg, quotasvc.ResourceSeats, 1)
	if err == nil {
		t.Fatal("expected quota exceeded error, got nil")
	}
	if appErrStatus(t, err) != http.StatusConflict {
		t.Errorf("expected 409 Conflict, got %d", appErrStatus(t, err))
	}
}

func TestQuota_Unlimited_AlwaysPasses(t *testing.T) {
	// Enterprise plan has unlimited seats.
	orgs := &fakeOrgRepo{org: &organization.Organization{ID: testOrg, PlanTier: "enterprise"}}
	usage := &fakeUsageRepo{usage: &organization.Usage{Members: 100000}}
	svc := quotasvc.NewService(orgs, usage)

	if err := svc.EnsureWithinLimit(context.Background(), testOrg, quotasvc.ResourceSeats, 1); err != nil {
		t.Fatalf("expected unlimited to pass, got error: %v", err)
	}
}

func TestQuota_Storage_ExceedingBytes_ReturnsConflict(t *testing.T) {
	// Free plan allows 500 MB; adding a 1 MB file when already at the cap exceeds.
	const mb = int64(1024 * 1024)
	orgs := &fakeOrgRepo{org: &organization.Organization{ID: testOrg, PlanTier: "free"}}
	usage := &fakeUsageRepo{usage: &organization.Usage{StorageBytes: 500 * mb}}
	svc := quotasvc.NewService(orgs, usage)

	err := svc.EnsureWithinLimit(context.Background(), testOrg, quotasvc.ResourceStorage, mb)
	if err == nil {
		t.Fatal("expected storage quota exceeded, got nil")
	}
	if appErrStatus(t, err) != http.StatusConflict {
		t.Errorf("expected 409 Conflict, got %d", appErrStatus(t, err))
	}
}

func TestQuota_UnknownTier_NoEnforcement(t *testing.T) {
	orgs := &fakeOrgRepo{org: &organization.Organization{ID: testOrg, PlanTier: "mystery"}}
	usage := &fakeUsageRepo{usage: &organization.Usage{Members: 999}}
	svc := quotasvc.NewService(orgs, usage)

	if err := svc.EnsureWithinLimit(context.Background(), testOrg, quotasvc.ResourceSeats, 1); err != nil {
		t.Fatalf("expected unknown tier to skip enforcement, got error: %v", err)
	}
}

func TestQuota_OrgNotFound_ReturnsNotFound(t *testing.T) {
	orgs := &fakeOrgRepo{org: nil}
	usage := &fakeUsageRepo{usage: &organization.Usage{}}
	svc := quotasvc.NewService(orgs, usage)

	err := svc.EnsureWithinLimit(context.Background(), testOrg, quotasvc.ResourceSeats, 1)
	if err == nil {
		t.Fatal("expected not found, got nil")
	}
	if !apperrors.IsNotFound(err) {
		t.Errorf("expected 404, got %d", appErrStatus(t, err))
	}
}
