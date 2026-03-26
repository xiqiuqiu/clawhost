package model

import "testing"

func TestSeedSystemAdminPolicy(t *testing.T) {
	db := setupModelTestDB(t)
	if err := db.AutoMigrate(&AdminRole{}, &AdminPermission{}, &AdminRolePermission{}); err != nil {
		t.Fatalf("migrate admin policy models: %v", err)
	}

	if err := SeedSystemAdminPolicy(); err != nil {
		t.Fatalf("seed system admin policy: %v", err)
	}

	roles, err := ListAdminRoles()
	if err != nil {
		t.Fatalf("list admin roles: %v", err)
	}
	if len(roles) != 4 {
		t.Fatalf("expected 4 built-in roles, got %d", len(roles))
	}

	permissions, err := ListAdminPermissions()
	if err != nil {
		t.Fatalf("list admin permissions: %v", err)
	}
	if len(permissions) != len(SystemAdminPermissionCatalog()) {
		t.Fatalf("expected %d permissions, got %d", len(SystemAdminPermissionCatalog()), len(permissions))
	}

	if !RoleHasPermission(AdminRolePlatformAdmin, PermissionAdminsManage) {
		t.Fatal("expected persisted platform admin policy to include admins:manage")
	}
	if RoleHasPermission(AdminRoleViewer, PermissionAuditRead) {
		t.Fatal("did not expect persisted viewer policy to include audit:read")
	}
}
