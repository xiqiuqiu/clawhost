package v1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authmw "github.com/clawhost/clawhost/middleware"
	"github.com/clawhost/clawhost/model"
	"github.com/clawhost/clawhost/util"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupAdminAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	util.SetDBForTesting(db)
	t.Cleanup(func() {
		util.SetDBForTesting(nil)
		viper.Reset()
	})

	return db
}

func TestBootstrapAdmin(t *testing.T) {
	db := setupAdminAuthTestDB(t)
	viper.Set("api.admin_token", "bootstrap-token")
	if err := db.AutoMigrate(&model.AdminUser{}, &model.AdminSession{}, &model.AdminMembership{}); err != nil {
		t.Fatalf("migrate admin auth models: %v", err)
	}

	reqBody := map[string]string{
		"email":    "admin@example.com",
		"name":     "Admin",
		"password": "secret-123",
	}
	body, _ := json.Marshal(reqBody)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bot/api/v1/admin/bootstrap", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAuthorization, "Bearer bootstrap-token")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := BootstrapAdmin(c); err != nil {
		t.Fatalf("BootstrapAdmin returned error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var count int64
	if err := db.Table("admin_users").Count(&count).Error; err != nil {
		t.Fatalf("count admin users: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 admin user after bootstrap, got %d", count)
	}

	req = httptest.NewRequest(http.MethodPost, "/bot/api/v1/admin/bootstrap", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAuthorization, "Bearer bootstrap-token")
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	if err := BootstrapAdmin(c); err != nil {
		t.Fatalf("second BootstrapAdmin returned error: %v", err)
	}
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status %d for second bootstrap, got %d", http.StatusConflict, rec.Code)
	}
}

func TestAdminLogin(t *testing.T) {
	db := setupAdminAuthTestDB(t)
	if err := db.AutoMigrate(&model.AdminUser{}, &model.AdminSession{}, &model.AdminMembership{}); err != nil {
		t.Fatalf("migrate admin auth models: %v", err)
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

	reqBody := map[string]string{
		"email":    "admin@example.com",
		"password": "secret-123",
	}
	body, _ := json.Marshal(reqBody)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bot/api/v1/admin/login", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := AdminLogin(c); err != nil {
		t.Fatalf("AdminLogin returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp util.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected login response data map, got %#v", resp.Data)
	}
	if data["session_token"] == "" {
		t.Fatalf("expected session token in login response, got %#v", data)
	}
	permissions, ok := data["permissions"].([]interface{})
	if !ok || len(permissions) == 0 {
		t.Fatalf("expected permissions in login response, got %#v", data["permissions"])
	}
}

func TestAdminLogout(t *testing.T) {
	db := setupAdminAuthTestDB(t)
	if err := db.AutoMigrate(&model.AdminUser{}, &model.AdminSession{}, &model.AdminMembership{}); err != nil {
		t.Fatalf("migrate admin auth models: %v", err)
	}

	admin := &model.AdminUser{
		Email:        "logout@example.com",
		Name:         "Logout",
		PasswordHash: "secret-123",
	}
	if err := model.CreateAdminUser(admin); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	session, rawToken, err := model.CreateAdminSession(admin.ID, model.DefaultAdminSessionExpiry())
	if err != nil {
		t.Fatalf("create admin session: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bot/api/v1/admin/logout", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+rawToken)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(authmw.ContextKeyAdminUser, admin)
	c.Set(authmw.ContextKeyAdminSession, session)

	if err := AdminLogout(c); err != nil {
		t.Fatalf("AdminLogout returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	got, err := model.GetActiveAdminSessionByToken(rawToken)
	if err == nil || got != nil {
		t.Fatalf("expected revoked session to be unavailable, got session=%+v err=%v", got, err)
	}
}

func TestVerifyAdminSession(t *testing.T) {
	db := setupAdminAuthTestDB(t)
	if err := db.AutoMigrate(&model.AdminUser{}, &model.AdminSession{}, &model.AdminMembership{}); err != nil {
		t.Fatalf("migrate admin auth models: %v", err)
	}

	admin := &model.AdminUser{
		Email:        "me@example.com",
		Name:         "Me",
		PasswordHash: "secret-123",
	}
	if err := model.CreateAdminUser(admin); err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	if err := model.CreateAdminMembership(&model.AdminMembership{
		AdminUserID: admin.ID,
		Role:        model.AdminRoleOperator,
		ScopeType:   model.AdminScopePlatform,
	}); err != nil {
		t.Fatalf("create admin membership: %v", err)
	}

	session, _, err := model.CreateAdminSession(admin.ID, model.DefaultAdminSessionExpiry())
	if err != nil {
		t.Fatalf("create admin session: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bot/api/v1/admin/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(authmw.ContextKeyAdminUser, admin)
	c.Set(authmw.ContextKeyAdminSession, session)

	if err := GetCurrentAdmin(c); err != nil {
		t.Fatalf("GetCurrentAdmin returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp util.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected me response data map, got %#v", resp.Data)
	}
	roles, ok := data["roles"].([]interface{})
	if !ok || len(roles) != 1 {
		t.Fatalf("expected role summary in me response, got %#v", data["roles"])
	}
}
