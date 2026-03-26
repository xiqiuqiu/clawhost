package v1

import (
	"context"

	authmw "github.com/clawhost/clawhost/middleware"
	"github.com/clawhost/clawhost/model"
	auditservice "github.com/clawhost/clawhost/service/audit"
	"github.com/clawhost/clawhost/service/k8s"
	"github.com/clawhost/clawhost/util"
	"github.com/labstack/echo/v4"
)

var getAdminDeploymentStatusInfoFn = func(botID string) (*k8s.DeploymentStatusInfo, error) {
	return k8s.GetDeploymentStatusInfo(context.Background(), botID)
}

func resolveAdminBotRuntimeStatus(bot *model.Bot) *model.Bot {
	cloned := *bot

	if bot.Status == model.BotStatusCreated || bot.Status == model.BotStatusStopped {
		return &cloned
	}

	statusInfo, err := getAdminDeploymentStatusInfoFn(bot.ID)
	if err != nil || statusInfo == nil {
		return &cloned
	}

	switch statusInfo.Status {
	case "ready":
		cloned.Status = model.BotStatusRunning
	case "starting", "updating", "not_ready":
		cloned.Status = model.BotStatusStarting
	case "not_found":
		if bot.Status == model.BotStatusRunning {
			cloned.Status = model.BotStatusStopped
			cloned.Endpoint = ""
		}
	}

	if cloned.Status != bot.Status || cloned.Endpoint != bot.Endpoint {
		if err := model.UpdateBot(&cloned); err != nil {
			return &cloned
		}
	}

	return &cloned
}

func filterBotsForAdminScope(adminUser *model.AdminUser, bots []*model.Bot) ([]*model.Bot, error) {
	if adminUser == nil {
		return bots, nil
	}

	access, err := model.ResolveAdminAccess(adminUser.ID)
	if err != nil {
		return nil, err
	}
	if access.ScopeMode != model.AdminScopeModeSelectedApps {
		return bots, nil
	}

	allowedAppIDs := make(map[string]struct{}, len(access.AppScopeIDs))
	for _, appID := range access.AppScopeIDs {
		allowedAppIDs[appID] = struct{}{}
	}

	filteredBots := make([]*model.Bot, 0, len(bots))
	for _, bot := range bots {
		if _, ok := allowedAppIDs[bot.AppID]; ok {
			filteredBots = append(filteredBots, bot)
		}
	}
	return filteredBots, nil
}

// AdminCreateBot creates a new bot (admin only)
func AdminCreateBot(c echo.Context) error {
	var req struct {
		AppID  string `json:"app_id"`
		UserID string `json:"user_id"`
		Name   string `json:"name"`
		Slug   string `json:"slug"`
	}
	if err := c.Bind(&req); err != nil {
		return util.BadRequest(c, "invalid request body")
	}
	if req.Name == "" {
		return util.BadRequest(c, "name is required")
	}
	if req.AppID == "" {
		return util.BadRequest(c, "app_id is required")
	}
	if req.UserID == "" {
		req.UserID = "admin"
	}

	adminUser := authmw.GetAdminUserFromContext(c)
	if adminUser != nil {
		allowed, err := model.AdminHasPermission(adminUser.ID, model.PermissionBotsCreate, req.AppID)
		if err != nil {
			return util.InternalError(c, "failed to resolve admin permissions")
		}
		if !allowed {
			return util.Forbidden(c, "insufficient permissions")
		}
	}

	bot := &model.Bot{
		AppID:  req.AppID,
		UserID: req.UserID,
		Name:   req.Name,
		Slug:   req.Slug,
		Status: model.BotStatusCreated,
	}

	resolvedConfig, err := applyDefaultBotConfig(nil, loadDefaultBotConfig())
	if err != nil {
		return util.BadRequest(c, err.Error())
	}
	if len(resolvedConfig) > 0 {
		if err := bot.SetConfigMap(resolvedConfig); err != nil {
			return util.InternalError(c, "failed to set config")
		}
	}

	if err := model.CreateBot(bot); err != nil {
		return util.InternalError(c, "failed to create bot")
	}
	writeAuditEntry(c, auditservice.Entry{
		AppID:       bot.AppID,
		Action:      "bot.create",
		TargetType:  "bot",
		TargetID:    bot.ID,
		TargetLabel: bot.Name,
		Result:      model.AuditResultSuccess,
		Metadata: map[string]interface{}{
			"slug": bot.Slug,
		},
	})

	return util.Success(c, bot)
}

// AdminListBots lists all bots across all apps (admin only)
func AdminListBots(c echo.Context) error {
	bots, err := model.ListAllBots()
	if err != nil {
		return util.InternalError(c, "failed to list bots")
	}
	adminUser := authmw.GetAdminUserFromContext(c)
	bots, err = filterBotsForAdminScope(adminUser, bots)
	if err != nil {
		return util.InternalError(c, "failed to resolve admin access")
	}

	items := make([]*model.Bot, 0, len(bots))
	for _, bot := range bots {
		items = append(items, resolveAdminBotRuntimeStatus(bot))
	}

	return util.Success(c, items)
}

// AdminStartBot starts a bot by ID (admin only)
func AdminStartBot(c echo.Context) error {
	botID := c.Param("id")
	bot, err := model.GetBotByID(botID)
	if err != nil {
		return util.NotFound(c, "bot not found")
	}

	if bot.Status == model.BotStatusRunning {
		writeAuditEntry(c, auditservice.Entry{
			AppID:       bot.AppID,
			Action:      "bot.start",
			TargetType:  "bot",
			TargetID:    bot.ID,
			TargetLabel: bot.Name,
			Result:      model.AuditResultFailure,
			Metadata: map[string]interface{}{
				"reason": "bot is already running",
			},
		})
		return util.BadRequest(c, "bot is already running")
	}

	ctx := context.Background()
	openclawConfig, _ := bot.GetOpenClawConfig()
	k8sConfig := convertToK8sConfig(bot, openclawConfig)

	if err := k8s.CreateDeployment(ctx, bot.ID, bot.UserID, bot.AccessToken, k8sConfig); err != nil {
		return util.InternalError(c, "failed to create deployment: "+err.Error())
	}

	endpoint, err := k8s.CreateService(ctx, bot.ID, bot.UserID)
	if err != nil {
		k8s.DeleteDeployment(ctx, bot.ID)
		return util.InternalError(c, "failed to create service: "+err.Error())
	}

	if err := model.UpdateBotStatus(bot.ID, model.BotStatusStarting, endpoint); err != nil {
		return util.InternalError(c, "failed to update bot status")
	}

	// Wait for pod ready in background, then update status
	go func() {
		bgCtx := context.Background()
		_, err := k8s.WaitForPodReady(bgCtx, bot.ID, 120)
		if err != nil {
			model.UpdateBotStatus(bot.ID, model.BotStatusError, endpoint)
			return
		}
		model.UpdateBotStatus(bot.ID, model.BotStatusRunning, endpoint)
		if k8sConfig.AccessToken != "" {
			k8s.WriteConfigToBot(bgCtx, bot.ID, k8sConfig, false)
		}
	}()

	bot.Status = model.BotStatusStarting
	bot.Endpoint = endpoint
	writeAuditEntry(c, auditservice.Entry{
		AppID:       bot.AppID,
		Action:      "bot.start",
		TargetType:  "bot",
		TargetID:    bot.ID,
		TargetLabel: bot.Name,
		Result:      model.AuditResultSuccess,
	})
	return util.Success(c, bot)
}

// AdminStopBot stops a bot by ID (admin only)
func AdminStopBot(c echo.Context) error {
	botID := c.Param("id")
	bot, err := model.GetBotByID(botID)
	if err != nil {
		return util.NotFound(c, "bot not found")
	}

	if bot.Status != model.BotStatusRunning {
		writeAuditEntry(c, auditservice.Entry{
			AppID:       bot.AppID,
			Action:      "bot.stop",
			TargetType:  "bot",
			TargetID:    bot.ID,
			TargetLabel: bot.Name,
			Result:      model.AuditResultFailure,
			Metadata: map[string]interface{}{
				"reason": "bot is not running",
			},
		})
		return util.BadRequest(c, "bot is not running")
	}

	ctx := context.Background()
	if err := k8s.DeleteDeployment(ctx, bot.ID); err != nil {
		return util.InternalError(c, "failed to delete deployment: "+err.Error())
	}
	if err := k8s.DeleteService(ctx, bot.ID); err != nil {
		return util.InternalError(c, "failed to delete service: "+err.Error())
	}

	if err := model.UpdateBotStatus(bot.ID, model.BotStatusStopped, ""); err != nil {
		return util.InternalError(c, "failed to update bot status")
	}

	bot.Status = model.BotStatusStopped
	bot.Endpoint = ""
	writeAuditEntry(c, auditservice.Entry{
		AppID:       bot.AppID,
		Action:      "bot.stop",
		TargetType:  "bot",
		TargetID:    bot.ID,
		TargetLabel: bot.Name,
		Result:      model.AuditResultSuccess,
	})
	return util.Success(c, bot)
}

// AdminDeleteBot deletes a bot by ID (admin only)
func AdminDeleteBot(c echo.Context) error {
	botID := c.Param("id")
	bot, err := model.GetBotByID(botID)
	if err != nil {
		return util.NotFound(c, "bot not found")
	}

	ctx := context.Background()
	if bot.Status == model.BotStatusRunning {
		k8s.DeleteDeployment(ctx, bot.ID)
		k8s.DeleteService(ctx, bot.ID)
	}

	if err := model.DeleteBot(bot.ID); err != nil {
		return util.InternalError(c, "failed to delete bot")
	}
	writeAuditEntry(c, auditservice.Entry{
		AppID:       bot.AppID,
		Action:      "bot.delete",
		TargetType:  "bot",
		TargetID:    bot.ID,
		TargetLabel: bot.Name,
		Result:      model.AuditResultSuccess,
	})

	return util.Success(c, map[string]string{"message": "bot deleted"})
}
