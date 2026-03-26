package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clawhost/clawhost/model"
	"github.com/labstack/echo/v4"
)

func TestRequirePermission(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	if err := db.AutoMigrate(&model.AdminUser{}, &model.AdminMembership{}, &model.AdminRole{}, &model.AdminPermission{}, &model.AdminRolePermission{}); err != nil {
		t.Fatalf("migrate admin permission models: %v", err)
	}
	if err := model.SeedSystemAdminPolicy(); err != nil {
		t.Fatalf("seed system admin policy: %v", err)
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
		Role:        model.AdminRoleViewer,
		ScopeType:   model.AdminScopePlatform,
	}); err != nil {
		t.Fatalf("create admin membership: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/admin/apps", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextKeyAdminUser, admin)

	called := false
	handler := RequirePermission(model.PermissionAppsCreate)(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusNoContent)
	})

	if err := handler(c); err != nil {
		t.Fatalf("RequirePermission returned error: %v", err)
	}
	if called {
		t.Fatal("did not expect viewer to reach protected handler")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestRequireAppScope(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	if err := db.AutoMigrate(&model.AdminUser{}, &model.AdminMembership{}, &model.AdminRole{}, &model.AdminPermission{}, &model.AdminRolePermission{}); err != nil {
		t.Fatalf("migrate admin permission models: %v", err)
	}
	if err := model.SeedSystemAdminPolicy(); err != nil {
		t.Fatalf("seed system admin policy: %v", err)
	}

	admin := &model.AdminUser{
		Email:        "appadmin@example.com",
		Name:         "App Admin",
		PasswordHash: "secret-123",
	}
	if err := model.CreateAdminUser(admin); err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	if err := model.CreateAdminMembership(&model.AdminMembership{
		AdminUserID: admin.ID,
		Role:        model.AdminRoleAppAdmin,
		ScopeType:   model.AdminScopeApp,
		ScopeID:     "app-123",
	}); err != nil {
		t.Fatalf("create admin membership: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/admin/bots", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(ContextKeyAdminUser, admin)

	called := false
	handler := RequireAppPermission(model.PermissionBotsCreate, func(echo.Context) (string, error) {
		return "app-123", nil
	})(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusNoContent)
	})

	if err := handler(c); err != nil {
		t.Fatalf("RequireAppPermission returned error: %v", err)
	}
	if !called {
		t.Fatal("expected scoped app admin to reach protected handler")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(ContextKeyAdminUser, admin)
	called = false
	handler = RequireAppPermission(model.PermissionBotsCreate, func(echo.Context) (string, error) {
		return "app-999", nil
	})(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusNoContent)
	})

	if err := handler(c); err != nil {
		t.Fatalf("RequireAppPermission mismatch returned error: %v", err)
	}
	if called {
		t.Fatal("did not expect scoped app admin to reach mismatched app handler")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}

	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)
	c.Set(ContextKeyAdminUser, admin)
	handler = RequireAppPermission(model.PermissionBotsCreate, func(echo.Context) (string, error) {
		return "", errors.New("missing app id")
	})(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	if err := handler(c); err != nil {
		t.Fatalf("RequireAppPermission resolver error returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
