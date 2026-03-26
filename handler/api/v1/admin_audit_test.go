package v1

import (
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

func TestAuditCreateAppWritesEntry(t *testing.T) {
	db := setupAdminAuthTestDB(t)
	if err := db.AutoMigrate(&model.App{}, &model.AdminUser{}, &model.AdminMembership{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate audit handler models: %v", err)
	}

	admin := &model.AdminUser{
		Email:        "admin@example.com",
		Name:         "Admin",
		PasswordHash: "secret-123",
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

	body, _ := json.Marshal(CreateAppRequest{
		Name:              "Audit App",
		OwnerEmail:        "owner@example.com",
		BotDomainTemplate: "https://{bot_id}.clawhost.ai",
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bot/api/v1/admin/apps", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(authmw.ContextKeyAdminUser, admin)

	if err := CreateApp(c); err != nil {
		t.Fatalf("CreateApp returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	logs, err := model.ListAuditLogs(model.AuditLogFilter{})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].Action != "app.create" {
		t.Fatalf("expected app.create audit action, got %q", logs[0].Action)
	}
}

func TestAuditListEndpoint(t *testing.T) {
	db := setupAdminAuthTestDB(t)
	if err := db.AutoMigrate(&model.AdminUser{}, &model.AdminMembership{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate audit list models: %v", err)
	}

	admin := &model.AdminUser{
		Email:        "viewer@example.com",
		Name:         "Viewer",
		PasswordHash: "secret-123",
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
	if err := model.CreateAuditLog(&model.AuditLog{
		ActorAdminID: admin.ID,
		ActorEmail:   admin.Email,
		AppID:        "app-123",
		Action:       "bot.create",
		TargetType:   "bot",
		TargetID:     "bot-123",
		TargetLabel:  "Audit Bot",
		Result:       model.AuditResultSuccess,
		SourceIP:     "127.0.0.1",
	}); err != nil {
		t.Fatalf("create audit log: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bot/api/v1/admin/audit?action=bot.create", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(authmw.ContextKeyAdminUser, admin)

	if err := ListAdminAuditLogs(c); err != nil {
		t.Fatalf("ListAdminAuditLogs returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp util.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode audit response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected success code, got %d", resp.Code)
	}
}
