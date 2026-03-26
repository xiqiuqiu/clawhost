package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	authmw "github.com/clawhost/clawhost/middleware"
	"github.com/clawhost/clawhost/model"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

func TestUpgradeBotWritesFailureAuditWhenBotNotRunning(t *testing.T) {
	db := setupAdminAuthTestDB(t)
	if err := db.AutoMigrate(&model.AdminUser{}, &model.AuditLog{}, &model.Bot{}); err != nil {
		t.Fatalf("migrate models: %v", err)
	}
	migrateAdminPolicyModelsForTest(t, db)

	admin := mustCreateAdminUser(t, "operator@example.com")
	bot := &model.Bot{
		AppID:  "app-123",
		UserID: "admin",
		Name:   "Stopped Bot",
		Status: model.BotStatusStopped,
	}
	if err := model.CreateBot(bot); err != nil {
		t.Fatalf("create bot: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bot/api/v1/admin/bots/"+bot.ID+"/upgrade", bytes.NewBufferString(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(bot.ID)
	c.Set(authmw.ContextKeyAdminUser, admin)

	if err := UpgradeBot(c); err != nil {
		t.Fatalf("UpgradeBot returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	logs, err := model.ListAuditLogs(model.AuditLogFilter{Action: "bot.upgrade"})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Result != model.AuditResultFailure {
		t.Fatalf("expected failure audit result, got %s", logs[0].Result)
	}
	if logs[0].TargetID != bot.ID {
		t.Fatalf("expected target id %s, got %s", bot.ID, logs[0].TargetID)
	}

	metadata := decodeAuditMetadata(t, logs[0].Metadata)
	if metadata["reason"] != "bot is not running" {
		t.Fatalf("expected failure reason, got %#v", metadata["reason"])
	}
}

func TestUpgradeAllBotsWritesSummaryAudit(t *testing.T) {
	db := setupAdminAuthTestDB(t)
	if err := db.AutoMigrate(&model.AdminUser{}, &model.AuditLog{}, &model.Bot{}); err != nil {
		t.Fatalf("migrate models: %v", err)
	}
	migrateAdminPolicyModelsForTest(t, db)

	admin := mustCreateAdminUser(t, "platform@example.com")
	botA := &model.Bot{
		AppID:  "app-123",
		UserID: "admin",
		Name:   "Bot A",
		Status: model.BotStatusRunning,
	}
	botB := &model.Bot{
		AppID:  "app-456",
		UserID: "admin",
		Name:   "Bot B",
		Status: model.BotStatusRunning,
	}
	for _, bot := range []*model.Bot{botA, botB} {
		if err := model.CreateBot(bot); err != nil {
			t.Fatalf("create bot: %v", err)
		}
	}

	restoreGetImage := getDeploymentImageFn
	restoreUpdateImage := updateDeploymentImageFn
	restoreSyncConfig := syncConfigToPodFn
	defer func() {
		getDeploymentImageFn = restoreGetImage
		updateDeploymentImageFn = restoreUpdateImage
		syncConfigToPodFn = restoreSyncConfig
	}()

	viper.Set("openclaw.image", "target:image")
	getDeploymentImageFn = func(_ context.Context, botID string) (string, error) {
		if botID == botA.ID {
			return "old:image", nil
		}
		return "target:image", nil
	}
	updateDeploymentImageFn = func(_ context.Context, botID, image string) error {
		return nil
	}
	syncConfigToPodFn = func(_ context.Context, _ string) error {
		return nil
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bot/api/v1/admin/bots/upgrade", bytes.NewBufferString(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(authmw.ContextKeyAdminUser, admin)

	if err := UpgradeAllBots(c); err != nil {
		t.Fatalf("UpgradeAllBots returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	logs, err := model.ListAuditLogs(model.AuditLogFilter{Action: "bot.upgrade_all"})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Result != model.AuditResultSuccess {
		t.Fatalf("expected success audit result, got %s", logs[0].Result)
	}

	metadata := decodeAuditMetadata(t, logs[0].Metadata)
	assertJSONNumber(t, metadata["total"], 2)
	assertJSONNumber(t, metadata["upgraded"], 1)
	assertJSONNumber(t, metadata["failed"], 0)
	assertJSONNumber(t, metadata["skipped"], 1)
	if metadata["image"] != "target:image" {
		t.Fatalf("expected target image metadata, got %#v", metadata["image"])
	}
}

func TestRestartAllBotsWritesInitiationAudit(t *testing.T) {
	db := setupAdminAuthTestDB(t)
	if err := db.AutoMigrate(&model.AdminUser{}, &model.AuditLog{}, &model.Bot{}); err != nil {
		t.Fatalf("migrate models: %v", err)
	}
	migrateAdminPolicyModelsForTest(t, db)

	admin := mustCreateAdminUser(t, "restart@example.com")
	bot := &model.Bot{
		AppID:  "app-789",
		UserID: "admin",
		Name:   "Restart Bot",
		Status: model.BotStatusRunning,
	}
	if err := model.CreateBot(bot); err != nil {
		t.Fatalf("create bot: %v", err)
	}

	restoreRestartAsync := restartBotsAsyncFn
	defer func() {
		restartBotsAsyncFn = restoreRestartAsync
	}()
	restartBotsAsyncFn = func(_ []*model.Bot) {}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bot/api/v1/admin/bots/restart", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(authmw.ContextKeyAdminUser, admin)

	if err := RestartAllBots(c); err != nil {
		t.Fatalf("RestartAllBots returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	logs, err := model.ListAuditLogs(model.AuditLogFilter{Action: "bot.restart_all"})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Result != model.AuditResultSuccess {
		t.Fatalf("expected success audit result, got %s", logs[0].Result)
	}

	metadata := decodeAuditMetadata(t, logs[0].Metadata)
	assertJSONNumber(t, metadata["total"], 1)
	if metadata["status"] != "initiated" {
		t.Fatalf("expected initiated status, got %#v", metadata["status"])
	}
}

func mustCreateAdminUser(t *testing.T, email string) *model.AdminUser {
	t.Helper()

	admin := &model.AdminUser{
		Email:        email,
		Name:         "Admin",
		PasswordHash: "secret-123",
		Status:       model.AdminUserStatusActive,
	}
	if err := model.CreateAdminUser(admin); err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	if err := model.CreateAdminMembership(&model.AdminMembership{
		AdminUserID: admin.ID,
		Role:        model.AdminRolePlatformAdmin,
		ScopeType:   model.AdminScopePlatform,
	}); err != nil {
		t.Fatalf("create admin membership: %v", err)
	}
	return admin
}

func decodeAuditMetadata(t *testing.T, raw json.RawMessage) map[string]interface{} {
	t.Helper()

	if len(raw) == 0 {
		return map[string]interface{}{}
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(raw, &metadata); err != nil {
		t.Fatalf("decode audit metadata: %v", err)
	}
	return metadata
}

func assertJSONNumber(t *testing.T, value interface{}, expected float64) {
	t.Helper()

	number, ok := value.(float64)
	if !ok {
		t.Fatalf("expected json number, got %#v", value)
	}
	if number != expected {
		t.Fatalf("expected %.0f, got %.0f", expected, number)
	}
}
