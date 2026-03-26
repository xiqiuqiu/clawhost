package v1

import (
	"context"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	authmw "github.com/clawhost/clawhost/middleware"
	"github.com/clawhost/clawhost/model"
	"github.com/clawhost/clawhost/util"
	"github.com/labstack/echo/v4"
)

func setupAdminBotHandlerTestDB(t *testing.T) {
	t.Helper()

	db := setupAdminAuthTestDB(t)
	if err := db.AutoMigrate(&model.App{}, &model.Bot{}); err != nil {
		t.Fatalf("migrate app and bot models: %v", err)
	}
	migrateAdminPolicyModelsForTest(t, db)
}

func createTestBotRecord(t *testing.T, appID, name string, status model.BotStatus) *model.Bot {
	t.Helper()

	bot := &model.Bot{
		AppID:  appID,
		UserID: "admin",
		Name:   name,
		Status: status,
	}
	if err := model.CreateBot(bot); err != nil {
		t.Fatalf("create test bot: %v", err)
	}
	return bot
}

func resolveBotAppIDFromParamForTest(c echo.Context) (string, error) {
	botID := c.Param("id")
	if botID == "" {
		return "", echo.NewHTTPError(http.StatusBadRequest, "bot id is required")
	}
	bot, err := model.GetBotByID(botID)
	if err != nil {
		return "", echo.NewHTTPError(http.StatusNotFound, "bot not found")
	}
	return bot.AppID, nil
}

func TestAdminListBotsFiltersSelectedAppScope(t *testing.T) {
	setupAdminBotHandlerTestDB(t)

	allowedApp := createTestAppRecord(t, "Allowed App")
	otherApp := createTestAppRecord(t, "Other App")
	createTestBotRecord(t, allowedApp.ID, "Allowed Bot", model.BotStatusCreated)
	createTestBotRecord(t, otherApp.ID, "Other Bot", model.BotStatusCreated)
	admin := createScopedAdminUser(t, "viewer@example.com", model.AdminRoleViewer, allowedApp.ID)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bot/api/v1/admin/bots", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(authmw.ContextKeyAdminUser, admin)

	handler := authmw.RequirePermission(model.PermissionBotsRead)(AdminListBots)
	if err := handler(c); err != nil {
		t.Fatalf("AdminListBots returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp util.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	items, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatalf("expected bot array, got %#v", resp.Data)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 visible bot, got %d", len(items))
	}
	record, ok := items[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected bot object, got %#v", items[0])
	}
	if record["app_id"] != allowedApp.ID {
		t.Fatalf("expected allowed app id %q, got %#v", allowedApp.ID, record["app_id"])
	}
}

func TestAdminStartBotRejectsOutOfScopeAdmin(t *testing.T) {
	setupAdminBotHandlerTestDB(t)

	allowedApp := createTestAppRecord(t, "Allowed App")
	otherApp := createTestAppRecord(t, "Other App")
	bot := createTestBotRecord(t, otherApp.ID, "Other Bot", model.BotStatusCreated)
	admin := createScopedAdminUser(t, "operator@example.com", model.AdminRoleOperator, allowedApp.ID)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bot/api/v1/admin/bots/"+bot.ID+"/start", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(bot.ID)
	c.Set(authmw.ContextKeyAdminUser, admin)

	handler := authmw.RequireAppPermission(model.PermissionBotsStart, resolveBotAppIDFromParamForTest)(AdminStartBot)
	if err := handler(c); err != nil {
		t.Fatalf("AdminStartBot returned error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestAdminStopBotRejectsOutOfScopeAdmin(t *testing.T) {
	setupAdminBotHandlerTestDB(t)

	allowedApp := createTestAppRecord(t, "Allowed App")
	otherApp := createTestAppRecord(t, "Other App")
	bot := createTestBotRecord(t, otherApp.ID, "Other Bot", model.BotStatusRunning)
	admin := createScopedAdminUser(t, "operator@example.com", model.AdminRoleOperator, allowedApp.ID)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bot/api/v1/admin/bots/"+bot.ID+"/stop", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(bot.ID)
	c.Set(authmw.ContextKeyAdminUser, admin)

	handler := authmw.RequireAppPermission(model.PermissionBotsStop, resolveBotAppIDFromParamForTest)(AdminStopBot)
	if err := handler(c); err != nil {
		t.Fatalf("AdminStopBot returned error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestAdminDeleteBotRejectsOutOfScopeAdmin(t *testing.T) {
	setupAdminBotHandlerTestDB(t)

	allowedApp := createTestAppRecord(t, "Allowed App")
	otherApp := createTestAppRecord(t, "Other App")
	bot := createTestBotRecord(t, otherApp.ID, "Other Bot", model.BotStatusCreated)
	admin := createScopedAdminUser(t, "operator@example.com", model.AdminRoleOperator, allowedApp.ID)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/bot/api/v1/admin/bots/"+bot.ID, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(bot.ID)
	c.Set(authmw.ContextKeyAdminUser, admin)

	handler := authmw.RequireAppPermission(model.PermissionBotsDelete, resolveBotAppIDFromParamForTest)(AdminDeleteBot)
	if err := handler(c); err != nil {
		t.Fatalf("AdminDeleteBot returned error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestUpgradeBotRejectsOutOfScopeAdmin(t *testing.T) {
	setupAdminBotHandlerTestDB(t)

	allowedApp := createTestAppRecord(t, "Allowed App")
	otherApp := createTestAppRecord(t, "Other App")
	bot := createTestBotRecord(t, otherApp.ID, "Other Bot", model.BotStatusRunning)
	admin := createScopedAdminUser(t, "operator@example.com", model.AdminRoleOperator, allowedApp.ID)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bot/api/v1/admin/bots/"+bot.ID+"/upgrade", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(bot.ID)
	c.Set(authmw.ContextKeyAdminUser, admin)

	handler := authmw.RequireAppPermission(model.PermissionBotsUpgrade, resolveBotAppIDFromParamForTest)(UpgradeBot)
	if err := handler(c); err != nil {
		t.Fatalf("UpgradeBot returned error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestUpgradeAllBotsOnlyProcessesScopedApps(t *testing.T) {
	setupAdminBotHandlerTestDB(t)

	allowedApp := createTestAppRecord(t, "Allowed App")
	otherApp := createTestAppRecord(t, "Other App")
	createTestBotRecord(t, allowedApp.ID, "Allowed Running Bot", model.BotStatusRunning)
	createTestBotRecord(t, otherApp.ID, "Other Running Bot", model.BotStatusRunning)
	admin := createScopedAdminUser(t, "operator@example.com", model.AdminRoleOperator, allowedApp.ID)

	originalGetImage := getDeploymentImageFn
	originalUpdateImage := updateDeploymentImageFn
	originalSyncConfig := syncConfigToPodFn
	defer func() {
		getDeploymentImageFn = originalGetImage
		updateDeploymentImageFn = originalUpdateImage
		syncConfigToPodFn = originalSyncConfig
	}()

	getDeploymentImageFn = func(ctx context.Context, botID string) (string, error) {
		return "old-image", nil
	}
	updateDeploymentImageFn = func(ctx context.Context, botID, image string) error {
		return nil
	}
	syncConfigToPodFn = func(ctx context.Context, botID string) error {
		return nil
	}

	body, _ := json.Marshal(UpgradeBotsRequest{Image: "test-image:latest"})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bot/api/v1/admin/bots/upgrade", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(authmw.ContextKeyAdminUser, admin)

	handler := authmw.RequirePermission(model.PermissionBotsUpgrade)(UpgradeAllBots)
	if err := handler(c); err != nil {
		t.Fatalf("UpgradeAllBots returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp util.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	payload, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object response, got %#v", resp.Data)
	}
	if payload["total"] != float64(1) {
		t.Fatalf("expected only 1 in-scope bot to be processed, got %#v", payload["total"])
	}
}
