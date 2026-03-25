package v1

import (
	"net/http"
	"strings"

	authmw "github.com/clawhost/clawhost/middleware"
	"github.com/clawhost/clawhost/model"
	"github.com/clawhost/clawhost/service/auth"
	"github.com/clawhost/clawhost/util"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type bootstrapAdminRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type adminLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func BootstrapAdmin(c echo.Context) error {
	if !hasValidBootstrapToken(c.Request().Header.Get("Authorization")) {
		return util.Unauthorized(c, "invalid admin bootstrap token")
	}

	var req bootstrapAdminRequest
	if err := c.Bind(&req); err != nil {
		return util.BadRequest(c, "invalid request body")
	}
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Password) == "" {
		return util.BadRequest(c, "email, name, and password are required")
	}

	count, err := model.CountAdminUsers()
	if err != nil {
		return util.InternalError(c, "failed to count admin users")
	}
	if count > 0 {
		return c.JSON(http.StatusConflict, util.Response{
			Code:    http.StatusConflict,
			Message: "admin bootstrap already completed",
		})
	}

	adminUser := &model.AdminUser{
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: req.Password,
		Status:       model.AdminUserStatusActive,
		IsBootstrap:  true,
	}
	if err := model.CreateAdminUser(adminUser); err != nil {
		return util.InternalError(c, "failed to create admin user")
	}

	return c.JSON(http.StatusCreated, util.Response{
		Code:    0,
		Message: "success",
		Data: map[string]interface{}{
			"admin": sanitizeAdminUser(adminUser),
		},
	})
}

func AdminLogin(c echo.Context) error {
	var req adminLoginRequest
	if err := c.Bind(&req); err != nil {
		return util.BadRequest(c, "invalid request body")
	}
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		return util.BadRequest(c, "email and password are required")
	}

	adminUser, err := model.GetAdminUserByEmail(req.Email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return util.Unauthorized(c, "invalid email or password")
		}
		return util.InternalError(c, "failed to get admin user")
	}
	if adminUser.Status != model.AdminUserStatusActive {
		return util.Forbidden(c, "admin user is disabled")
	}
	if !auth.VerifyPassword(adminUser.PasswordHash, req.Password) {
		return util.Unauthorized(c, "invalid email or password")
	}

	session, rawToken, err := model.CreateAdminSession(adminUser.ID, model.DefaultAdminSessionExpiry())
	if err != nil {
		return util.InternalError(c, "failed to create admin session")
	}

	return util.Success(c, map[string]interface{}{
		"admin":         sanitizeAdminUser(adminUser),
		"session_token": rawToken,
		"expires_at":    session.ExpiresAt,
	})
}

func AdminLogout(c echo.Context) error {
	session := authmw.GetAdminSessionFromContext(c)
	if session == nil {
		return util.Unauthorized(c, "admin session not found")
	}
	if err := model.RevokeAdminSession(session.ID); err != nil {
		return util.InternalError(c, "failed to revoke admin session")
	}
	return util.Success(c, map[string]bool{"revoked": true})
}

func GetCurrentAdmin(c echo.Context) error {
	adminUser := authmw.GetAdminUserFromContext(c)
	session := authmw.GetAdminSessionFromContext(c)
	if adminUser == nil || session == nil {
		return util.Unauthorized(c, "admin session not found")
	}

	return util.Success(c, map[string]interface{}{
		"admin":      sanitizeAdminUser(adminUser),
		"expires_at": session.ExpiresAt,
	})
}

func sanitizeAdminUser(adminUser *model.AdminUser) map[string]interface{} {
	return map[string]interface{}{
		"id":           adminUser.ID,
		"email":        adminUser.Email,
		"name":         adminUser.Name,
		"status":       adminUser.Status,
		"is_bootstrap": adminUser.IsBootstrap,
		"created_at":   adminUser.CreatedAt,
		"updated_at":   adminUser.UpdatedAt,
	}
}

func hasValidBootstrapToken(authHeader string) bool {
	adminToken := viper.GetString("api.admin_token")
	if adminToken == "" {
		return false
	}
	parts := strings.SplitN(authHeader, " ", 2)
	return len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" && parts[1] == adminToken
}
