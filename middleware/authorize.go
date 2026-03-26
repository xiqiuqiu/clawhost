package middleware

import (
	"fmt"

	"github.com/clawhost/clawhost/model"
	"github.com/clawhost/clawhost/util"
	"github.com/labstack/echo/v4"
)

func RequirePermission(permission string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			adminUser := GetAdminUserFromContext(c)
			if adminUser == nil {
				return util.Unauthorized(c, "admin user not found")
			}

			allowed, err := model.AdminHasPermission(adminUser.ID, permission, "")
			if err != nil {
				return util.InternalError(c, "failed to resolve admin permissions")
			}
			if !allowed {
				return util.Forbidden(c, "insufficient permissions")
			}
			return next(c)
		}
	}
}

func RequireAppPermission(permission string, appIDResolver func(echo.Context) (string, error)) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			adminUser := GetAdminUserFromContext(c)
			if adminUser == nil {
				return util.Unauthorized(c, "admin user not found")
			}

			appID, err := appIDResolver(c)
			if err != nil {
				if httpErr, ok := err.(*echo.HTTPError); ok {
					return c.JSON(httpErr.Code, util.Response{
						Code:    httpErr.Code,
						Message: fmt.Sprint(httpErr.Message),
					})
				}
				return util.BadRequest(c, err.Error())
			}

			allowed, err := model.AdminHasPermission(adminUser.ID, permission, appID)
			if err != nil {
				return util.InternalError(c, "failed to resolve admin permissions")
			}
			if !allowed {
				return util.Forbidden(c, "insufficient permissions")
			}
			return next(c)
		}
	}
}
