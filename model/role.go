package model

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
	},
	AdminRoleOperator: {
		PermissionAppsRead:    true,
		PermissionBotsRead:    true,
		PermissionBotsStart:   true,
		PermissionBotsStop:    true,
		PermissionBotsUpgrade: true,
	},
	AdminRoleViewer: {
		PermissionAppsRead: true,
		PermissionBotsRead: true,
	},
}

func RoleHasPermission(role, permission string) bool {
	permissions, ok := rolePermissions[role]
	if !ok {
		return false
	}
	return permissions[permission]
}
