package v1

import (
	"sort"
	"strings"

	"github.com/clawhost/clawhost/model"
	auditservice "github.com/clawhost/clawhost/service/audit"
	"github.com/clawhost/clawhost/util"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type updateAdminRolePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

func ListAdminRolesHandler(c echo.Context) error {
	roles, err := model.ListAdminRoles()
	if err != nil {
		return util.InternalError(c, "failed to list admin roles")
	}

	data := make([]map[string]interface{}, 0, len(roles))
	for _, role := range roles {
		permissions, err := model.ListAdminRolePermissions(role.Key)
		if err != nil {
			return util.InternalError(c, "failed to list role permissions")
		}
		data = append(data, map[string]interface{}{
			"id":          role.ID,
			"key":         role.Key,
			"name":        role.Name,
			"description": role.Description,
			"scope_type":  role.ScopeType,
			"is_system":   role.IsSystem,
			"permissions": permissions,
			"created_at":  role.CreatedAt,
			"updated_at":  role.UpdatedAt,
		})
	}

	return util.Success(c, data)
}

func ListAdminPermissionsHandler(c echo.Context) error {
	permissions, err := model.ListAdminPermissions()
	if err != nil {
		return util.InternalError(c, "failed to list admin permissions")
	}
	return util.Success(c, permissions)
}

func UpdateAdminRolePermissions(c echo.Context) error {
	roleKey := strings.TrimSpace(c.Param("key"))
	if roleKey == "" {
		return util.BadRequest(c, "role key is required")
	}

	role, err := model.GetAdminRoleByKey(roleKey)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return util.NotFound(c, "admin role not found")
		}
		return util.InternalError(c, "failed to load admin role")
	}
	if !role.IsSystem {
		return util.Forbidden(c, "only system roles can be updated")
	}

	var req updateAdminRolePermissionsRequest
	if err := c.Bind(&req); err != nil {
		return util.BadRequest(c, "invalid request body")
	}

	permissionKeys := make([]string, 0, len(req.Permissions))
	seen := make(map[string]struct{}, len(req.Permissions))
	for _, permissionKey := range req.Permissions {
		permissionKey = strings.TrimSpace(permissionKey)
		if permissionKey == "" {
			continue
		}
		if _, ok := seen[permissionKey]; ok {
			continue
		}
		seen[permissionKey] = struct{}{}
		permissionKeys = append(permissionKeys, permissionKey)
	}
	sort.Strings(permissionKeys)

	validPermissions, err := model.ListAdminPermissions()
	if err != nil {
		return util.InternalError(c, "failed to load permission catalog")
	}
	validPermissionKeys := make(map[string]struct{}, len(validPermissions))
	for _, permission := range validPermissions {
		validPermissionKeys[permission.Key] = struct{}{}
	}
	for _, permissionKey := range permissionKeys {
		if _, ok := validPermissionKeys[permissionKey]; !ok {
			return util.BadRequest(c, "invalid permission key")
		}
	}

	if roleKey == model.AdminRolePlatformAdmin && !containsString(permissionKeys, model.PermissionAdminsManage) {
		return util.Forbidden(c, "cannot remove platform admin manage access")
	}

	beforePermissions, err := model.ListAdminRolePermissions(roleKey)
	if err != nil {
		return util.InternalError(c, "failed to load existing role permissions")
	}
	if err := model.ReplaceAdminRolePermissions(roleKey, permissionKeys); err != nil {
		return util.InternalError(c, "failed to update role permissions")
	}

	writeAuditEntry(c, auditservice.Entry{
		Action:      "role.update_permissions",
		TargetType:  "admin_role",
		TargetID:    role.Key,
		TargetLabel: role.Name,
		Metadata: map[string]interface{}{
			"role_key":           role.Key,
			"before_permissions": beforePermissions,
			"after_permissions":  permissionKeys,
		},
	})

	return util.Success(c, map[string]interface{}{
		"key":         role.Key,
		"name":        role.Name,
		"description": role.Description,
		"scope_type":  role.ScopeType,
		"is_system":   role.IsSystem,
		"permissions": permissionKeys,
	})
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
