package model

import (
	"errors"
	"testing"
)

func TestRolePermissions(t *testing.T) {
	db := setupModelTestDB(t)
	if err := db.AutoMigrate(&AdminRole{}, &AdminPermission{}, &AdminRolePermission{}); err != nil {
		t.Fatalf("migrate admin policy models: %v", err)
	}
	if err := SeedSystemAdminPolicy(); err != nil {
		t.Fatalf("seed system admin policy: %v", err)
	}

	if !RoleHasPermission(AdminRolePlatformAdmin, PermissionAppsCreate) {
		t.Fatal("expected platform admin to create apps")
	}
	if !RoleHasPermission(AdminRolePlatformAdmin, PermissionAdminsManage) {
		t.Fatal("expected platform admin to manage admin users")
	}
	if RoleHasPermission(AdminRoleViewer, PermissionAppsDelete) {
		t.Fatal("did not expect viewer to delete apps")
	}
	if RoleHasPermission(AdminRoleOperator, PermissionAdminsManage) {
		t.Fatal("did not expect operator to manage admin users")
	}
	if RoleHasPermission(AdminRoleViewer, PermissionAuditRead) {
		t.Fatal("did not expect viewer to read audit logs")
	}
	if !RoleHasPermission(AdminRoleOperator, PermissionBotsStop) {
		t.Fatal("expected operator to stop bots")
	}
}

func TestAdminMembershipScope(t *testing.T) {
	db := setupModelTestDB(t)
	if err := db.AutoMigrate(&AdminUser{}, &AdminMembership{}, &AdminRole{}, &AdminPermission{}, &AdminRolePermission{}); err != nil {
		t.Fatalf("migrate admin membership models: %v", err)
	}
	if err := SeedSystemAdminPolicy(); err != nil {
		t.Fatalf("seed system admin policy: %v", err)
	}

	admin := &AdminUser{
		Email:        "scoped@example.com",
		Name:         "Scoped",
		PasswordHash: "secret-123",
	}
	if err := CreateAdminUser(admin); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	if err := CreateAdminMembership(&AdminMembership{
		AdminUserID: admin.ID,
		Role:        AdminRoleAppAdmin,
		ScopeType:   AdminScopeApp,
		ScopeID:     "app-123",
	}); err != nil {
		t.Fatalf("create admin membership: %v", err)
	}

	allowed, err := AdminHasPermission(admin.ID, PermissionBotsCreate, "app-123")
	if err != nil {
		t.Fatalf("check scoped permission: %v", err)
	}
	if !allowed {
		t.Fatal("expected scoped app admin to have permission on owned app")
	}

	allowed, err = AdminHasPermission(admin.ID, PermissionBotsCreate, "app-999")
	if err != nil {
		t.Fatalf("check mismatched scoped permission: %v", err)
	}
	if allowed {
		t.Fatal("did not expect app-scoped admin to have permission on other app")
	}

	summary, err := ResolveAdminAccess(admin.ID)
	if err != nil {
		t.Fatalf("resolve scoped admin access: %v", err)
	}
	if summary.ScopeMode != AdminScopeModeSelectedApps {
		t.Fatalf("expected scope mode %q, got %q", AdminScopeModeSelectedApps, summary.ScopeMode)
	}
	if len(summary.AppScopeIDs) != 1 || summary.AppScopeIDs[0] != "app-123" {
		t.Fatalf("expected one scoped app id, got %#v", summary.AppScopeIDs)
	}
}

func TestResolveAdminAccess(t *testing.T) {
	db := setupModelTestDB(t)
	if err := db.AutoMigrate(&AdminUser{}, &AdminMembership{}, &AdminRole{}, &AdminPermission{}, &AdminRolePermission{}); err != nil {
		t.Fatalf("migrate admin membership models: %v", err)
	}
	if err := SeedSystemAdminPolicy(); err != nil {
		t.Fatalf("seed system admin policy: %v", err)
	}

	admin := &AdminUser{
		Email:        "access@example.com",
		Name:         "Access",
		PasswordHash: "secret-123",
	}
	if err := CreateAdminUser(admin); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	if err := CreateAdminMembership(&AdminMembership{
		AdminUserID: admin.ID,
		Role:        AdminRoleOperator,
		ScopeType:   AdminScopeApp,
		ScopeID:     "app-456",
	}); err != nil {
		t.Fatalf("create app-scoped operator membership: %v", err)
	}
	if err := CreateAdminMembership(&AdminMembership{
		AdminUserID: admin.ID,
		Role:        AdminRoleOperator,
		ScopeType:   AdminScopeApp,
		ScopeID:     "app-123",
	}); err != nil {
		t.Fatalf("create second app-scoped operator membership: %v", err)
	}

	summary, err := ResolveAdminAccess(admin.ID)
	if err != nil {
		t.Fatalf("resolve admin access: %v", err)
	}
	if len(summary.Roles) != 1 {
		t.Fatalf("expected 1 effective role, got %d", len(summary.Roles))
	}
	if len(summary.Memberships) != 2 {
		t.Fatalf("expected 2 memberships, got %d", len(summary.Memberships))
	}
	if !contains(summary.Permissions, PermissionBotsUpgrade) {
		t.Fatalf("expected bots upgrade permission in summary")
	}
	if summary.ScopeMode != AdminScopeModeSelectedApps {
		t.Fatalf("expected scope mode %q, got %q", AdminScopeModeSelectedApps, summary.ScopeMode)
	}
	if len(summary.AppScopeIDs) != 2 || summary.AppScopeIDs[0] != "app-123" || summary.AppScopeIDs[1] != "app-456" {
		t.Fatalf("unexpected app scope ids: %#v", summary.AppScopeIDs)
	}
}

func TestResolveAdminAccessRejectsMixedRoles(t *testing.T) {
	db := setupModelTestDB(t)
	if err := db.AutoMigrate(&AdminUser{}, &AdminMembership{}, &AdminRole{}, &AdminPermission{}, &AdminRolePermission{}); err != nil {
		t.Fatalf("migrate admin membership models: %v", err)
	}
	if err := SeedSystemAdminPolicy(); err != nil {
		t.Fatalf("seed system admin policy: %v", err)
	}

	admin := &AdminUser{
		Email:        "mixed-roles@example.com",
		Name:         "Mixed Roles",
		PasswordHash: "secret-123",
	}
	if err := CreateAdminUser(admin); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	memberships := []*AdminMembership{
		{
			AdminUserID: admin.ID,
			Role:        AdminRoleOperator,
			ScopeType:   AdminScopeApp,
			ScopeID:     "app-123",
		},
		{
			AdminUserID: admin.ID,
			Role:        AdminRoleViewer,
			ScopeType:   AdminScopeApp,
			ScopeID:     "app-456",
		},
	}
	if err := ReplaceAdminMemberships(admin.ID, memberships); err != nil {
		t.Fatalf("replace memberships: %v", err)
	}

	_, err := ResolveAdminAccess(admin.ID)
	if !errors.Is(err, ErrAdminMembershipMixedRoles) {
		t.Fatalf("expected mixed role error, got %v", err)
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
