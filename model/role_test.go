package model

import "testing"

func TestRolePermissions(t *testing.T) {
	if !RoleHasPermission(AdminRolePlatformAdmin, PermissionAppsCreate) {
		t.Fatal("expected platform admin to create apps")
	}
	if RoleHasPermission(AdminRoleViewer, PermissionAppsDelete) {
		t.Fatal("did not expect viewer to delete apps")
	}
	if !RoleHasPermission(AdminRoleOperator, PermissionBotsStop) {
		t.Fatal("expected operator to stop bots")
	}
}

func TestAdminMembershipScope(t *testing.T) {
	db := setupModelTestDB(t)
	if err := db.AutoMigrate(&AdminUser{}, &AdminMembership{}); err != nil {
		t.Fatalf("migrate admin membership models: %v", err)
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
}

func TestResolveAdminAccess(t *testing.T) {
	db := setupModelTestDB(t)
	if err := db.AutoMigrate(&AdminUser{}, &AdminMembership{}); err != nil {
		t.Fatalf("migrate admin membership models: %v", err)
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
		ScopeType:   AdminScopePlatform,
	}); err != nil {
		t.Fatalf("create operator membership: %v", err)
	}
	if err := CreateAdminMembership(&AdminMembership{
		AdminUserID: admin.ID,
		Role:        AdminRoleAppAdmin,
		ScopeType:   AdminScopeApp,
		ScopeID:     "app-123",
	}); err != nil {
		t.Fatalf("create app admin membership: %v", err)
	}

	summary, err := ResolveAdminAccess(admin.ID)
	if err != nil {
		t.Fatalf("resolve admin access: %v", err)
	}
	if len(summary.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(summary.Roles))
	}
	if len(summary.Memberships) != 2 {
		t.Fatalf("expected 2 memberships, got %d", len(summary.Memberships))
	}
	if !contains(summary.Permissions, PermissionBotsUpgrade) {
		t.Fatalf("expected bots upgrade permission in summary")
	}
	if !contains(summary.Permissions, PermissionAppsUpdate) {
		t.Fatalf("expected apps update permission in summary")
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
