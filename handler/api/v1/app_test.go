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

func resolveAppIDFromParamForTest(c echo.Context) (string, error) {
	id := c.Param("id")
	if id == "" {
		return "", echo.NewHTTPError(http.StatusBadRequest, "app id is required")
	}
	return id, nil
}

func setupAppHandlerTestDB(t *testing.T) {
	t.Helper()

	db := setupAdminAuthTestDB(t)
	if err := db.AutoMigrate(&model.App{}); err != nil {
		t.Fatalf("migrate app model: %v", err)
	}
	migrateAdminPolicyModelsForTest(t, db)
}

func createScopedAdminUser(t *testing.T, email, role string, appIDs ...string) *model.AdminUser {
	t.Helper()

	admin := &model.AdminUser{
		Email:        email,
		Name:         email,
		PasswordHash: "secret-123",
	}
	if err := model.CreateAdminUser(admin); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	memberships := make([]*model.AdminMembership, 0, len(appIDs))
	if len(appIDs) == 0 {
		memberships = append(memberships, &model.AdminMembership{
			AdminUserID: admin.ID,
			Role:        role,
			ScopeType:   model.AdminScopePlatform,
		})
	} else {
		for _, appID := range appIDs {
			memberships = append(memberships, &model.AdminMembership{
				AdminUserID: admin.ID,
				Role:        role,
				ScopeType:   model.AdminScopeApp,
				ScopeID:     appID,
			})
		}
	}

	if err := model.ReplaceAdminMemberships(admin.ID, memberships); err != nil {
		t.Fatalf("replace admin memberships: %v", err)
	}
	return admin
}

func TestListAppsFiltersSelectedAppScope(t *testing.T) {
	setupAppHandlerTestDB(t)

	allowedApp := createTestAppRecord(t, "Allowed App")
	createTestAppRecord(t, "Other App")
	admin := createScopedAdminUser(t, "viewer@example.com", model.AdminRoleViewer, allowedApp.ID)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bot/api/v1/admin/apps", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(authmw.ContextKeyAdminUser, admin)

	handler := authmw.RequirePermission(model.PermissionAppsRead)(ListApps)
	if err := handler(c); err != nil {
		t.Fatalf("ListApps returned error: %v", err)
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
		t.Fatalf("expected apps array, got %#v", resp.Data)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 visible app, got %d", len(items))
	}
	appData, ok := items[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected app object, got %#v", items[0])
	}
	if appData["id"] != allowedApp.ID {
		t.Fatalf("expected allowed app %q, got %#v", allowedApp.ID, appData["id"])
	}
}

func TestGetAppRejectsOutOfScopeAdmin(t *testing.T) {
	setupAppHandlerTestDB(t)

	allowedApp := createTestAppRecord(t, "Allowed App")
	otherApp := createTestAppRecord(t, "Other App")
	admin := createScopedAdminUser(t, "app-admin@example.com", model.AdminRoleAppAdmin, allowedApp.ID)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bot/api/v1/admin/apps/"+otherApp.ID, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(otherApp.ID)
	c.Set(authmw.ContextKeyAdminUser, admin)

	handler := authmw.RequireAppPermission(model.PermissionAppsRead, resolveAppIDFromParamForTest)(GetApp)
	if err := handler(c); err != nil {
		t.Fatalf("GetApp returned error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestUpdateAppRejectsOutOfScopeAdmin(t *testing.T) {
	setupAppHandlerTestDB(t)

	allowedApp := createTestAppRecord(t, "Allowed App")
	otherApp := createTestAppRecord(t, "Other App")
	admin := createScopedAdminUser(t, "app-admin@example.com", model.AdminRoleAppAdmin, allowedApp.ID)

	body, _ := json.Marshal(UpdateAppRequest{Name: "Blocked Update"})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/bot/api/v1/admin/apps/"+otherApp.ID, bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(otherApp.ID)
	c.Set(authmw.ContextKeyAdminUser, admin)

	handler := authmw.RequireAppPermission(model.PermissionAppsUpdate, resolveAppIDFromParamForTest)(UpdateApp)
	if err := handler(c); err != nil {
		t.Fatalf("UpdateApp returned error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestDeleteAppRejectsOutOfScopeAdmin(t *testing.T) {
	setupAppHandlerTestDB(t)

	allowedApp := createTestAppRecord(t, "Allowed App")
	otherApp := createTestAppRecord(t, "Other App")
	admin := createScopedAdminUser(t, "app-admin@example.com", model.AdminRoleAppAdmin, allowedApp.ID)

	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/bot/api/v1/admin/apps/"+otherApp.ID, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(otherApp.ID)
	c.Set(authmw.ContextKeyAdminUser, admin)

	handler := authmw.RequireAppPermission(model.PermissionAppsDelete, resolveAppIDFromParamForTest)(DeleteApp)
	if err := handler(c); err != nil {
		t.Fatalf("DeleteApp returned error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestResetAppTokenRejectsOutOfScopeAdmin(t *testing.T) {
	setupAppHandlerTestDB(t)

	allowedApp := createTestAppRecord(t, "Allowed App")
	otherApp := createTestAppRecord(t, "Other App")
	admin := createScopedAdminUser(t, "app-admin@example.com", model.AdminRoleAppAdmin, allowedApp.ID)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bot/api/v1/admin/apps/"+otherApp.ID+"/reset-token", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(otherApp.ID)
	c.Set(authmw.ContextKeyAdminUser, admin)

	handler := authmw.RequireAppPermission(model.PermissionAppsTokenReset, resolveAppIDFromParamForTest)(ResetAppToken)
	if err := handler(c); err != nil {
		t.Fatalf("ResetAppToken returned error: %v", err)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}
