package user_test

import (
	"testing"

	usersvc "github.com/novudesk/novudesk/internal/application/user"
	"github.com/novudesk/novudesk/internal/domain/user"
)

func TestApplyOverrides_GrantAddsPermission(t *testing.T) {
	result := usersvc.ApplyOverrides(
		[]string{"tickets:read"},
		[]user.PermissionOverride{{PermissionKey: "tickets:create", IsGranted: true}},
	)
	if !contains(result, "tickets:create") {
		t.Errorf("expected tickets:create to be granted, got %v", result)
	}
	if !contains(result, "tickets:read") {
		t.Errorf("expected tickets:read to be preserved, got %v", result)
	}
}

func TestApplyOverrides_DenyRemovesPermission(t *testing.T) {
	result := usersvc.ApplyOverrides(
		[]string{"tickets:read", "tickets:delete"},
		[]user.PermissionOverride{{PermissionKey: "tickets:delete", IsGranted: false}},
	)
	if contains(result, "tickets:delete") {
		t.Errorf("expected tickets:delete to be removed, got %v", result)
	}
	if !contains(result, "tickets:read") {
		t.Errorf("expected tickets:read to be preserved, got %v", result)
	}
}

func TestApplyOverrides_NoOverrides_ReturnsRoleKeys(t *testing.T) {
	keys := []string{"tickets:read", "comments:create"}
	result := usersvc.ApplyOverrides(keys, nil)
	if len(result) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(result))
	}
}

func TestApplyOverrides_GrantAlreadyPresent_NoDuplicate(t *testing.T) {
	result := usersvc.ApplyOverrides(
		[]string{"tickets:read"},
		[]user.PermissionOverride{{PermissionKey: "tickets:read", IsGranted: true}},
	)
	count := 0
	for _, k := range result {
		if k == "tickets:read" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 occurrence of tickets:read, got %d", count)
	}
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
