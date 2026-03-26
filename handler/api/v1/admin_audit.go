package v1

import (
	"strconv"
	"time"

	authmw "github.com/clawhost/clawhost/middleware"
	"github.com/clawhost/clawhost/model"
	auditservice "github.com/clawhost/clawhost/service/audit"
	"github.com/clawhost/clawhost/util"
	"github.com/labstack/echo/v4"
)

func writeAuditEntry(c echo.Context, entry auditservice.Entry) {
	if _, err := auditservice.Write(authmw.BuildAuditRequestContext(c), entry); err != nil {
		c.Logger().Errorf("write audit log failed: %v", err)
	}
}

func ListAdminAuditLogs(c echo.Context) error {
	adminUser := authmw.GetAdminUserFromContext(c)
	if adminUser == nil {
		return util.Unauthorized(c, "admin session not found")
	}

	allowed, err := model.AdminHasPermission(adminUser.ID, model.PermissionAuditRead, "")
	if err != nil {
		return util.InternalError(c, "failed to resolve admin permissions")
	}
	if !allowed {
		return util.Forbidden(c, "insufficient permissions")
	}

	filter := model.AuditLogFilter{
		Actor:  c.QueryParam("actor"),
		AppID:  c.QueryParam("app_id"),
		Action: c.QueryParam("action"),
		Result: c.QueryParam("result"),
	}

	if limit := c.QueryParam("limit"); limit != "" {
		if parsedLimit, err := parsePositiveInt(limit); err == nil {
			filter.Limit = parsedLimit
		}
	}
	if dateFrom := c.QueryParam("date_from"); dateFrom != "" {
		parsed, err := time.Parse(time.RFC3339, dateFrom)
		if err != nil {
			return util.BadRequest(c, "invalid date_from")
		}
		filter.DateFrom = &parsed
	}
	if dateTo := c.QueryParam("date_to"); dateTo != "" {
		parsed, err := time.Parse(time.RFC3339, dateTo)
		if err != nil {
			return util.BadRequest(c, "invalid date_to")
		}
		filter.DateTo = &parsed
	}

	logs, err := model.ListAuditLogs(filter)
	if err != nil {
		return util.InternalError(c, "failed to list audit logs")
	}
	return util.Success(c, logs)
}

func parsePositiveInt(raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, strconv.ErrSyntax
	}
	return value, nil
}
