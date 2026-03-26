package v1

import (
	"errors"
	"net/http"
	"strings"
	"time"

	authmw "github.com/clawhost/clawhost/middleware"
	"github.com/clawhost/clawhost/model"
	auditservice "github.com/clawhost/clawhost/service/audit"
	"github.com/clawhost/clawhost/util"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type createAdminManagedUserRequest struct {
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	Password    string   `json:"password"`
	Role        string   `json:"role"`
	ScopeMode   string   `json:"scope_mode"`
	AppScopeIDs []string `json:"app_scope_ids"`
}

type updateAdminManagedUserRequest struct {
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Role        string   `json:"role"`
	Password    string   `json:"password"`
	ScopeMode   string   `json:"scope_mode"`
	AppScopeIDs []string `json:"app_scope_ids"`
}

func ListAdminUsers(c echo.Context) error {
	adminUsers, err := model.ListAdminUsers()
	if err != nil {
		return util.InternalError(c, "failed to list admin users")
	}

	data := make([]map[string]interface{}, 0, len(adminUsers))
	for _, adminUser := range adminUsers {
		payload, err := buildManagedAdminUserPayload(adminUser)
		if err != nil {
			return util.InternalError(c, "failed to resolve admin access")
		}
		data = append(data, payload)
	}

	return util.Success(c, data)
}

func CreateAdminManagedUser(c echo.Context) error {
	var req createAdminManagedUserRequest
	if err := c.Bind(&req); err != nil {
		return util.BadRequest(c, "invalid request body")
	}
	req.Role = strings.TrimSpace(req.Role)
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Password) == "" {
		return util.BadRequest(c, "email, name, and password are required")
	}
	role, err := getManagedAdminRole(req.Role)
	if err != nil {
		return util.BadRequest(c, err.Error())
	}
	memberships, normalizedScopeMode, normalizedAppScopeIDs, err := buildManagedAdminMemberships(role, req.ScopeMode, req.AppScopeIDs)
	if err != nil {
		return util.BadRequest(c, err.Error())
	}

	if _, err := model.GetAdminUserByEmail(req.Email); err == nil {
		return c.JSON(http.StatusConflict, util.Response{
			Code:    http.StatusConflict,
			Message: "admin user already exists",
		})
	} else if err != gorm.ErrRecordNotFound {
		return util.InternalError(c, "failed to check existing admin user")
	}

	adminUser := &model.AdminUser{
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: req.Password,
		Status:       model.AdminUserStatusActive,
	}
	if err := model.CreateAdminUser(adminUser); err != nil {
		return util.InternalError(c, "failed to create admin user")
	}
	if err := model.ReplaceAdminMemberships(adminUser.ID, memberships); err != nil {
		return util.InternalError(c, "failed to create admin membership")
	}

	writeAuditEntry(c, auditservice.Entry{
		Action:      "admin.create",
		TargetType:  "admin_user",
		TargetID:    adminUser.ID,
		TargetLabel: adminUser.Email,
		Metadata: map[string]interface{}{
			"role":          role.Key,
			"status":        adminUser.Status,
			"scope_mode":    normalizedScopeMode,
			"app_scope_ids": normalizedAppScopeIDs,
		},
	})

	payload, err := buildManagedAdminUserPayload(adminUser)
	if err != nil {
		return util.InternalError(c, "failed to resolve admin access")
	}

	return c.JSON(http.StatusCreated, util.Response{
		Code:    0,
		Message: "success",
		Data:    payload,
	})
}

func UpdateAdminManagedUser(c echo.Context) error {
	adminID := c.Param("id")
	if adminID == "" {
		return util.BadRequest(c, "admin user id is required")
	}

	var req updateAdminManagedUserRequest
	if err := c.Bind(&req); err != nil {
		return util.BadRequest(c, "invalid request body")
	}
	req.Role = strings.TrimSpace(req.Role)
	req.Status = strings.TrimSpace(req.Status)

	adminUser, err := model.GetAdminUserByID(adminID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return util.NotFound(c, "admin user not found")
		}
		return util.InternalError(c, "failed to load admin user")
	}
	actor := authmw.GetAdminUserFromContext(c)
	currentAccess, err := model.ResolveAdminAccess(adminUser.ID)
	if err != nil {
		return util.InternalError(c, "failed to resolve current admin access")
	}
	if actor != nil && actor.ID == adminUser.ID {
		if req.Status == model.AdminUserStatusDisabled {
			return util.Forbidden(c, "cannot disable the current admin user")
		}
		if req.Role != "" {
			currentRole := ""
			if len(currentAccess.Roles) > 0 {
				currentRole = currentAccess.Roles[0]
			}
			if currentRole == model.AdminRolePlatformAdmin && req.Role != model.AdminRolePlatformAdmin {
				return util.Forbidden(c, "cannot remove platform admin role from the current admin user")
			}
		}
	}
	effectiveRole := req.Role
	if effectiveRole == "" && len(currentAccess.Roles) > 0 {
		effectiveRole = currentAccess.Roles[0]
	}
	role, err := getManagedAdminRole(effectiveRole)
	if err != nil {
		return util.BadRequest(c, err.Error())
	}
	effectiveScopeMode := req.ScopeMode
	if effectiveScopeMode == "" {
		effectiveScopeMode = currentAccess.ScopeMode
	}
	effectiveAppScopeIDs := req.AppScopeIDs
	if req.ScopeMode == "" && len(req.AppScopeIDs) == 0 {
		effectiveAppScopeIDs = currentAccess.AppScopeIDs
	}
	memberships, normalizedScopeMode, normalizedAppScopeIDs, err := buildManagedAdminMemberships(role, effectiveScopeMode, effectiveAppScopeIDs)
	if err != nil {
		return util.BadRequest(c, err.Error())
	}

	changedFields := []string{}
	if trimmedName := strings.TrimSpace(req.Name); trimmedName != "" && trimmedName != adminUser.Name {
		adminUser.Name = trimmedName
		changedFields = append(changedFields, "name")
	}
	if req.Status != "" {
		if req.Status != model.AdminUserStatusActive && req.Status != model.AdminUserStatusDisabled {
			return util.BadRequest(c, "invalid status")
		}
		if req.Status != adminUser.Status {
			adminUser.Status = req.Status
			changedFields = append(changedFields, "status")
		}
	}
	if strings.TrimSpace(req.Password) != "" {
		hashed, err := model.HashAdminPassword(req.Password)
		if err != nil {
			return util.InternalError(c, "failed to hash admin password")
		}
		adminUser.PasswordHash = hashed
		changedFields = append(changedFields, "password")
	}

	if req.Role != "" {
		changedFields = append(changedFields, "role")
	}
	if req.ScopeMode != "" || len(req.AppScopeIDs) > 0 {
		changedFields = append(changedFields, "scope")
	}

	if err := util.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.AdminUser{}).
			Where("id = ?", adminUser.ID).
			Updates(map[string]interface{}{
				"name":          adminUser.Name,
				"status":        adminUser.Status,
				"password_hash": adminUser.PasswordHash,
				"updated_at":    time.Now(),
			}).Error; err != nil {
			return err
		}

		if req.Role != "" || req.ScopeMode != "" || len(req.AppScopeIDs) > 0 {
			if err := tx.Where("admin_user_id = ?", adminUser.ID).Delete(&model.AdminMembership{}).Error; err != nil {
				return err
			}
			for _, membership := range memberships {
				membership.AdminUserID = adminUser.ID
				if err := tx.Create(membership).Error; err != nil {
					return err
				}
			}
		}

		return nil
	}); err != nil {
		return util.InternalError(c, "failed to update admin user")
	}

	writeAuditEntry(c, auditservice.Entry{
		Action:      "admin.update",
		TargetType:  "admin_user",
		TargetID:    adminUser.ID,
		TargetLabel: adminUser.Email,
		Metadata: map[string]interface{}{
			"changed_fields":   changedFields,
			"status":           adminUser.Status,
			"role_before":      firstRole(currentAccess.Roles),
			"role_after":       role.Key,
			"scope_mode_before": currentAccess.ScopeMode,
			"scope_mode_after":  normalizedScopeMode,
			"app_scope_ids_before": currentAccess.AppScopeIDs,
			"app_scope_ids_after":  normalizedAppScopeIDs,
		},
	})

	payload, err := buildManagedAdminUserPayload(adminUser)
	if err != nil {
		return util.InternalError(c, "failed to resolve admin access")
	}

	return util.Success(c, payload)
}

func buildManagedAdminUserPayload(adminUser *model.AdminUser) (map[string]interface{}, error) {
	access, err := model.ResolveAdminAccess(adminUser.ID)
	if err != nil {
		return nil, err
	}

	payload := sanitizeAdminUser(adminUser)
	payload["roles"] = access.Roles
	payload["permissions"] = access.Permissions
	payload["memberships"] = access.Memberships
	payload["scope_mode"] = access.ScopeMode
	payload["app_scope_ids"] = access.AppScopeIDs
	return payload, nil
}

func getManagedAdminRole(roleKey string) (*model.AdminRole, error) {
	roleKey = strings.TrimSpace(roleKey)
	if roleKey == "" {
		return nil, errors.New("role is required")
	}
	role, err := model.GetAdminRoleByKey(roleKey)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("invalid role")
		}
		return nil, err
	}
	if !role.IsSystem {
		return nil, errors.New("invalid role")
	}
	return role, nil
}

func buildManagedAdminMemberships(role *model.AdminRole, requestedScopeMode string, appScopeIDs []string) ([]*model.AdminMembership, string, []string, error) {
	scopeMode := strings.TrimSpace(requestedScopeMode)
	if scopeMode == "" {
		scopeMode = model.AdminScopeModePlatform
	}

	normalizedAppScopeIDs := make([]string, 0, len(appScopeIDs))
	appScopeIDSet := make(map[string]struct{})
	for _, appID := range appScopeIDs {
		appID = strings.TrimSpace(appID)
		if appID == "" {
			continue
		}
		if _, ok := appScopeIDSet[appID]; ok {
			continue
		}
		appScopeIDSet[appID] = struct{}{}
		if _, err := model.GetAppByID(appID); err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, "", nil, errors.New("invalid app scope")
			}
			return nil, "", nil, err
		}
		normalizedAppScopeIDs = append(normalizedAppScopeIDs, appID)
	}

	if role.Key == model.AdminRolePlatformAdmin {
		if scopeMode != model.AdminScopeModePlatform || len(normalizedAppScopeIDs) > 0 {
			return nil, "", nil, errors.New("platform admin must use platform scope")
		}
		return []*model.AdminMembership{
			{
				Role:      role.Key,
				ScopeType: model.AdminScopePlatform,
			},
		}, model.AdminScopeModePlatform, nil, nil
	}

	switch scopeMode {
	case model.AdminScopeModePlatform:
		return []*model.AdminMembership{
			{
				Role:      role.Key,
				ScopeType: model.AdminScopePlatform,
			},
		}, model.AdminScopeModePlatform, nil, nil
	case model.AdminScopeModeSelectedApps:
		if len(normalizedAppScopeIDs) == 0 {
			return nil, "", nil, errors.New("selected app scope requires at least one app")
		}
		memberships := make([]*model.AdminMembership, 0, len(normalizedAppScopeIDs))
		for _, appID := range normalizedAppScopeIDs {
			memberships = append(memberships, &model.AdminMembership{
				Role:      role.Key,
				ScopeType: model.AdminScopeApp,
				ScopeID:   appID,
			})
		}
		return memberships, model.AdminScopeModeSelectedApps, normalizedAppScopeIDs, nil
	default:
		return nil, "", nil, errors.New("invalid scope mode")
	}
}

func firstRole(roles []string) string {
	if len(roles) == 0 {
		return ""
	}
	return roles[0]
}
