package model

import "sort"

const (
	AdminScopePlatform = "platform"
	AdminScopeApp      = "app"
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
	PermissionBotsRead       = "bots:read"
	PermissionBotsCreate     = "bots:create"
	PermissionBotsStart      = "bots:start"
	PermissionBotsStop       = "bots:stop"
	PermissionBotsDelete     = "bots:delete"
	PermissionBotsUpgrade    = "bots:upgrade"
	PermissionAuditRead      = "audit:read"
)

var rolePermissions = map[string]map[string]bool{
	AdminRolePlatformAdmin: {
		PermissionAppsRead:       true,
		PermissionAppsCreate:     true,
		PermissionAppsUpdate:     true,
		PermissionAppsDelete:     true,
		PermissionAppsTokenReset: true,
		PermissionBotsRead:       true,
		PermissionBotsCreate:     true,
		PermissionBotsStart:      true,
		PermissionBotsStop:       true,
		PermissionBotsDelete:     true,
		PermissionBotsUpgrade:    true,
		PermissionAuditRead:      true,
	},
	AdminRoleAppAdmin: {
		PermissionAppsRead:       true,
		PermissionAppsUpdate:     true,
		PermissionAppsTokenReset: true,
		PermissionBotsRead:       true,
		PermissionBotsCreate:     true,
		PermissionBotsStart:      true,
		PermissionBotsStop:       true,
		PermissionBotsDelete:     true,
		PermissionBotsUpgrade:    true,
		PermissionAuditRead:      true,
	},
	AdminRoleOperator: {
		PermissionAppsRead:    true,
		PermissionBotsRead:    true,
		PermissionBotsStart:   true,
		PermissionBotsStop:    true,
		PermissionBotsUpgrade: true,
		PermissionAuditRead:   true,
	},
	AdminRoleViewer: {
		PermissionAppsRead: true,
		PermissionBotsRead: true,
	},
}

var allPermissions = []string{
	PermissionAppsRead,
	PermissionAppsCreate,
	PermissionAppsUpdate,
	PermissionAppsDelete,
	PermissionAppsTokenReset,
	PermissionBotsRead,
	PermissionBotsCreate,
	PermissionBotsStart,
	PermissionBotsStop,
	PermissionBotsDelete,
	PermissionBotsUpgrade,
	PermissionAuditRead,
}

type AdminMembershipSummary struct {
	Role      string `json:"role"`
	ScopeType string `json:"scope_type"`
	ScopeID   string `json:"scope_id,omitempty"`
}

type AdminAccessSummary struct {
	Roles       []string                 `json:"roles"`
	Permissions []string                 `json:"permissions"`
	Memberships []AdminMembershipSummary `json:"memberships"`
}

func RoleHasPermission(role, permission string) bool {
	permissions, ok := rolePermissions[role]
	if !ok {
		return false
	}
	return permissions[permission]
}

func ResolveAdminAccess(adminUserID string) (*AdminAccessSummary, error) {
	memberships, err := ListAdminMembershipsByUser(adminUserID)
	if err != nil {
		return nil, err
	}

	roleSet := make(map[string]struct{})
	permissionSet := make(map[string]struct{})
	summary := &AdminAccessSummary{
		Roles:       []string{},
		Permissions: []string{},
		Memberships: make([]AdminMembershipSummary, 0, len(memberships)),
	}

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

		for _, permission := range allPermissions {
			if RoleHasPermission(membership.Role, permission) {
				permissionSet[permission] = struct{}{}
			}
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
