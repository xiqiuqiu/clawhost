package middleware

import (
	auditservice "github.com/clawhost/clawhost/service/audit"
	"github.com/labstack/echo/v4"
)

func BuildAuditRequestContext(c echo.Context) auditservice.RequestContext {
	adminUser := GetAdminUserFromContext(c)

	ctx := auditservice.RequestContext{
		SourceIP:  c.RealIP(),
		UserAgent: c.Request().UserAgent(),
	}
	if adminUser != nil {
		ctx.ActorAdminID = adminUser.ID
		ctx.ActorEmail = adminUser.Email
	}
	return ctx
}
