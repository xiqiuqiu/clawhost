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

func setupAdminRoleTestDB(t *testing.T) {
	t.Helper()

	db := setupAdminAuthTestDB(t)
	if err := db.AutoMigrate(&model.AdminUser{}, &model.AdminMembership{}, &model.AdminRole{}, &model.AdminPermission{}, &model.AdminRolePermission{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate admin role handler models: %v", err)
	}
	if err := model.SeedSystemAdminPolicy(); err != nil {
		t.Fatalf("seed system admin policy: %v", err)
	}
}

func TestListAdminRoles(t *testing.T) {
	setupAdminRoleTestDB(t)

	admin := createPlatformAdminUser(t, "admin@example.com")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bot/api/v1/admin/roles", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(authmw.ContextKeyAdminUser, admin)

	if err := ListAdminRolesHandler(c); err != nil {
		t.Fatalf("ListAdminRolesHandler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp util.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode roles response: %v", err)
	}
	items, ok := resp.Data.([]interface{})
	if !ok || len(items) != 4 {
		t.Fatalf("expected 4 roles in response, got %#v", resp.Data)
	}
}

func TestUpdateAdminRolePermissions(t *testing.T) {
	setupAdminRoleTestDB(t)

	admin := createPlatformAdminUser(t, "admin@example.com")

	body, _ := json.Marshal(map[string][]string{
		"permissions": []string{
			model.PermissionAppsRead,
			model.PermissionBotsRead,
			model.PermissionBotsStart,
		},
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/bot/api/v1/admin/roles/operator/permissions", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("key")
	c.SetParamValues(model.AdminRoleOperator)
	c.Set(authmw.ContextKeyAdminUser, admin)

	if err := UpdateAdminRolePermissions(c); err != nil {
		t.Fatalf("UpdateAdminRolePermissions returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if model.RoleHasPermission(model.AdminRoleOperator, model.PermissionBotsStop) {
		t.Fatal("expected operator bots:stop permission to be removed after update")
	}
	if !model.RoleHasPermission(model.AdminRoleOperator, model.PermissionBotsStart) {
		t.Fatal("expected operator bots:start permission to remain after update")
	}
}

func TestUpdateAdminRolePermissionsProtectsPlatformAdminManageAccess(t *testing.T) {
	setupAdminRoleTestDB(t)

	admin := createPlatformAdminUser(t, "admin@example.com")

	body, _ := json.Marshal(map[string][]string{
		"permissions": []string{
			model.PermissionAppsRead,
		},
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/bot/api/v1/admin/roles/platform_admin/permissions", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("key")
	c.SetParamValues(model.AdminRolePlatformAdmin)
	c.Set(authmw.ContextKeyAdminUser, admin)

	if err := UpdateAdminRolePermissions(c); err != nil {
		t.Fatalf("UpdateAdminRolePermissions returned error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestListAdminPermissions(t *testing.T) {
	setupAdminRoleTestDB(t)

	admin := createPlatformAdminUser(t, "admin@example.com")

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bot/api/v1/admin/permissions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(authmw.ContextKeyAdminUser, admin)

	if err := ListAdminPermissionsHandler(c); err != nil {
		t.Fatalf("ListAdminPermissionsHandler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp util.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode permissions response: %v", err)
	}
	items, ok := resp.Data.([]interface{})
	if !ok || len(items) == 0 {
		t.Fatalf("expected permission catalog in response, got %#v", resp.Data)
	}
}
