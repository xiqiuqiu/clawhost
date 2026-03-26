package model

import (
	"errors"
	"sort"

	"github.com/clawhost/clawhost/util"
	"gorm.io/gorm"
)

const (
	AdminScopePlatform = "platform"
	AdminScopeApp      = "app"
)

const (
	AdminScopeModePlatform     = "platform"
	AdminScopeModeSelectedApps = "selected_apps"
)

const (
	AdminRolePlatformAdmin = "platform_admin"
	AdminRoleAppAdmin      = "app_admin"
	AdminRoleOperator      = "operator"
	AdminRoleViewer        = "viewer"
)

const (
	PermissionAppsRead       = "apps:read"
	PermissionAppsCreate     = "apps:create"
	PermissionAppsUpdate     = "apps:update"
	PermissionAppsDelete     = "apps:delete"
	PermissionAppsTokenReset = "apps:token_reset"
	PermissionAdminsManage   = "admins:manage"
	PermissionBotsRead       = "bots:read"
	PermissionBotsCreate     = "bots:create"
	PermissionBotsStart      = "bots:start"
	PermissionBotsStop       = "bots:stop"
	PermissionBotsDelete     = "bots:delete"
	PermissionBotsUpgrade    = "bots:upgrade"
	PermissionAuditRead      = "audit:read"
)

type AdminMembershipSummary struct {
	Role      string `json:"role"`
	ScopeType string `json:"scope_type"`
	ScopeID   string `json:"scope_id,omitempty"`
}

type AdminAccessSummary struct {
	Roles       []string                 `json:"roles"`
	Permissions []string                 `json:"permissions"`
	Memberships []AdminMembershipSummary `json:"memberships"`
	ScopeMode   string                   `json:"scope_mode"`
	AppScopeIDs []string                 `json:"app_scope_ids,omitempty"`
}

var errMissingSystemAdminPolicy = errors.New("missing system admin policy")
var ErrAdminMembershipMixedRoles = errors.New("admin memberships must use a single role")

func SystemAdminRoleCatalog() []AdminRole {
	return []AdminRole{
		{
			Key:         AdminRolePlatformAdmin,
			Name:        "Platform Admin",
			Description: "Full platform access and policy management.",
			ScopeType:   AdminScopePlatform,
			IsSystem:    true,
		},
		{
			Key:         AdminRoleAppAdmin,
			Name:        "App Admin",
			Description: "Manage app-scoped app and bot resources.",
			ScopeType:   AdminScopeApp,
			IsSystem:    true,
		},
		{
			Key:         AdminRoleOperator,
			Name:        "Operator",
			Description: "Operate bot runtime actions without policy management.",
			ScopeType:   AdminScopePlatform,
			IsSystem:    true,
		},
		{
			Key:         AdminRoleViewer,
			Name:        "Viewer",
			Description: "Read-only platform access.",
			ScopeType:   AdminScopePlatform,
			IsSystem:    true,
		},
	}
}

func SystemAdminPermissionCatalog() []AdminPermission {
	return []AdminPermission{
		{Key: PermissionAppsRead, Name: "Read Apps", Description: "View applications.", ResourceType: "apps"},
		{Key: PermissionAppsCreate, Name: "Create Apps", Description: "Create applications.", ResourceType: "apps"},
		{Key: PermissionAppsUpdate, Name: "Update Apps", Description: "Edit applications.", ResourceType: "apps"},
		{Key: PermissionAppsDelete, Name: "Delete Apps", Description: "Delete applications.", ResourceType: "apps"},
		{Key: PermissionAppsTokenReset, Name: "Reset App Tokens", Description: "Reset application API tokens.", ResourceType: "apps"},
		{Key: PermissionAdminsManage, Name: "Manage Admins", Description: "Manage admins and role policy.", ResourceType: "admins"},
		{Key: PermissionBotsRead, Name: "Read Bots", Description: "View bots.", ResourceType: "bots"},
		{Key: PermissionBotsCreate, Name: "Create Bots", Description: "Create bots.", ResourceType: "bots"},
		{Key: PermissionBotsStart, Name: "Start Bots", Description: "Start bots.", ResourceType: "bots"},
		{Key: PermissionBotsStop, Name: "Stop Bots", Description: "Stop bots.", ResourceType: "bots"},
		{Key: PermissionBotsDelete, Name: "Delete Bots", Description: "Delete bots.", ResourceType: "bots"},
		{Key: PermissionBotsUpgrade, Name: "Upgrade Bots", Description: "Upgrade bots.", ResourceType: "bots"},
		{Key: PermissionAuditRead, Name: "Read Audit", Description: "View audit logs.", ResourceType: "audit"},
	}
}

func systemAdminRolePermissionCatalog() map[string][]string {
	return map[string][]string{
		AdminRolePlatformAdmin: {
			PermissionAppsRead,
			PermissionAppsCreate,
			PermissionAppsUpdate,
			PermissionAppsDelete,
			PermissionAppsTokenReset,
			PermissionAdminsManage,
			PermissionBotsRead,
			PermissionBotsCreate,
			PermissionBotsStart,
			PermissionBotsStop,
			PermissionBotsDelete,
			PermissionBotsUpgrade,
			PermissionAuditRead,
		},
		AdminRoleAppAdmin: {
			PermissionAppsRead,
			PermissionAppsUpdate,
			PermissionAppsTokenReset,
			PermissionBotsRead,
			PermissionBotsCreate,
			PermissionBotsStart,
			PermissionBotsStop,
			PermissionBotsDelete,
			PermissionBotsUpgrade,
			PermissionAuditRead,
		},
		AdminRoleOperator: {
			PermissionAppsRead,
			PermissionBotsRead,
			PermissionBotsStart,
			PermissionBotsStop,
			PermissionBotsUpgrade,
			PermissionAuditRead,
		},
		AdminRoleViewer: {
			PermissionAppsRead,
			PermissionBotsRead,
		},
	}
}

func SeedSystemAdminPolicy() error {
	db := util.GetDB()
	if db == nil {
		return errors.New("db is not initialized")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, role := range SystemAdminRoleCatalog() {
			if err := tx.Where("key = ?", role.Key).Assign(map[string]interface{}{
				"name":        role.Name,
				"description": role.Description,
				"scope_type":  role.ScopeType,
				"is_system":   role.IsSystem,
			}).FirstOrCreate(&AdminRole{Key: role.Key}).Error; err != nil {
				return err
			}
		}

		for _, permission := range SystemAdminPermissionCatalog() {
			if err := tx.Where("key = ?", permission.Key).Assign(map[string]interface{}{
				"name":          permission.Name,
				"description":   permission.Description,
				"resource_type": permission.ResourceType,
			}).FirstOrCreate(&AdminPermission{Key: permission.Key}).Error; err != nil {
				return err
			}
		}

		for roleKey, permissionKeys := range systemAdminRolePermissionCatalog() {
			for _, permissionKey := range permissionKeys {
				if err := tx.Where("role_key = ? AND permission_key = ?", roleKey, permissionKey).
					FirstOrCreate(&AdminRolePermission{
						RoleKey:       roleKey,
						PermissionKey: permissionKey,
					}).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func EnsureSystemAdminPolicy() error {
	if err := SeedSystemAdminPolicy(); err != nil {
		return err
	}

	roles, err := ListAdminRoles()
	if err != nil {
		return err
	}
	if len(roles) < len(SystemAdminRoleCatalog()) {
		return errMissingSystemAdminPolicy
	}

	permissions, err := ListAdminPermissions()
	if err != nil {
		return err
	}
	if len(permissions) < len(SystemAdminPermissionCatalog()) {
		return errMissingSystemAdminPolicy
	}

	for _, role := range SystemAdminRoleCatalog() {
		if _, err := GetAdminRoleByKey(role.Key); err != nil {
			return errMissingSystemAdminPolicy
		}
	}
	for _, permission := range SystemAdminPermissionCatalog() {
		if _, err := GetAdminPermissionByKey(permission.Key); err != nil {
			return errMissingSystemAdminPolicy
		}
	}
	return nil
}

func RoleHasPermission(role, permission string) bool {
	db := util.GetDB()
	if db == nil {
		return false
	}

	var count int64
	if err := db.Model(&AdminRolePermission{}).
		Where("role_key = ? AND permission_key = ?", role, permission).
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func ResolveAdminAccess(adminUserID string) (*AdminAccessSummary, error) {
	memberships, err := ListAdminMembershipsByUser(adminUserID)
	if err != nil {
		return nil, err
	}

	grant, err := ResolveAdminMembershipGrant(memberships)
	if err != nil {
		return nil, err
	}

	rolePermissionMap, err := ListAdminRolePermissionsByRoles(grant.RoleKeys)
	if err != nil {
		return nil, err
	}

	summary := &AdminAccessSummary{
		Roles:       []string{},
		Permissions: []string{},
		Memberships: make([]AdminMembershipSummary, 0, len(memberships)),
		ScopeMode:   grant.ScopeMode,
		AppScopeIDs: grant.AppScopeIDs,
	}
	permissionSet := make(map[string]struct{})
	roleSet := make(map[string]struct{})

	for _, membership := range memberships {
		if _, ok := roleSet[membership.Role]; !ok {
			roleSet[membership.Role] = struct{}{}
			summary.Roles = append(summary.Roles, membership.Role)
		}
		summary.Memberships = append(summary.Memberships, AdminMembershipSummary{
			Role:      membership.Role,
			ScopeType: membership.ScopeType,
			ScopeID:   membership.ScopeID,
		})

		for permission := range rolePermissionMap[membership.Role] {
			permissionSet[permission] = struct{}{}
		}
	}

	for permission := range permissionSet {
		summary.Permissions = append(summary.Permissions, permission)
	}

	sort.Strings(summary.Roles)
	sort.Strings(summary.Permissions)
	sort.Slice(summary.Memberships, func(i, j int) bool {
		if summary.Memberships[i].ScopeType == summary.Memberships[j].ScopeType {
			if summary.Memberships[i].ScopeID == summary.Memberships[j].ScopeID {
				return summary.Memberships[i].Role < summary.Memberships[j].Role
			}
			return summary.Memberships[i].ScopeID < summary.Memberships[j].ScopeID
		}
		return summary.Memberships[i].ScopeType < summary.Memberships[j].ScopeType
	})

	return summary, nil
}
