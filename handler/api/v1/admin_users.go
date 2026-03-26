package v1

import (
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

var managedAdminRoles = map[string]bool{
	model.AdminRolePlatformAdmin: true,
	model.AdminRoleOperator:      true,
	model.AdminRoleViewer:        true,
}

type createAdminManagedUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type updateAdminManagedUserRequest struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Role     string `json:"role"`
	Password string `json:"password"`
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
	if !isManagedAdminRole(req.Role) {
		return util.BadRequest(c, "invalid role")
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
	if err := model.CreateAdminMembership(&model.AdminMembership{
		AdminUserID: adminUser.ID,
		Role:        req.Role,
		ScopeType:   model.AdminScopePlatform,
	}); err != nil {
		return util.InternalError(c, "failed to create admin membership")
	}

	writeAuditEntry(c, auditservice.Entry{
		Action:      "admin.create",
		TargetType:  "admin_user",
		TargetID:    adminUser.ID,
		TargetLabel: adminUser.Email,
		Metadata: map[string]interface{}{
			"role":   req.Role,
			"status": adminUser.Status,
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
	if actor != nil && actor.ID == adminUser.ID {
		if req.Status == model.AdminUserStatusDisabled {
			return util.Forbidden(c, "cannot disable the current admin user")
		}
		if req.Role != "" {
			access, err := model.ResolveAdminAccess(adminUser.ID)
			if err != nil {
				return util.InternalError(c, "failed to resolve current admin access")
			}
			currentRole := ""
			if len(access.Roles) > 0 {
				currentRole = access.Roles[0]
			}
			if currentRole == model.AdminRolePlatformAdmin && req.Role != model.AdminRolePlatformAdmin {
				return util.Forbidden(c, "cannot remove platform admin role from the current admin user")
			}
		}
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
		if !isManagedAdminRole(req.Role) {
			return util.BadRequest(c, "invalid role")
		}
		changedFields = append(changedFields, "role")
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

		if req.Role != "" {
			if err := tx.Where("admin_user_id = ?", adminUser.ID).Delete(&model.AdminMembership{}).Error; err != nil {
				return err
			}
			if err := tx.Create(&model.AdminMembership{
				AdminUserID: adminUser.ID,
				Role:        req.Role,
				ScopeType:   model.AdminScopePlatform,
			}).Error; err != nil {
				return err
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
			"changed_fields": changedFields,
			"status":         adminUser.Status,
			"role":           req.Role,
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
	return payload, nil
}

func isManagedAdminRole(role string) bool {
	return managedAdminRoles[strings.TrimSpace(role)]
}
