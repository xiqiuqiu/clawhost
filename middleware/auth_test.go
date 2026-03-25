package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clawhost/clawhost/model"
	"github.com/clawhost/clawhost/util"
	"github.com/labstack/echo/v4"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupMiddlewareTestDB(t *testing.T) *gorm.DB {
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
	})

	return db
}

func TestAdminSessionAuth(t *testing.T) {
	db := setupMiddlewareTestDB(t)
	if err := db.AutoMigrate(&model.AdminUser{}, &model.AdminSession{}); err != nil {
		t.Fatalf("migrate admin auth models: %v", err)
	}

	admin := &model.AdminUser{
		Email:        "auth@example.com",
		Name:         "Auth",
		PasswordHash: "secret-123",
	}
	if err := model.CreateAdminUser(admin); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	_, rawToken, err := model.CreateAdminSession(admin.ID, model.DefaultAdminSessionExpiry())
	if err != nil {
		t.Fatalf("create admin session: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/admin/me", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+rawToken)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	handler := AdminSessionAuth()(func(c echo.Context) error {
		called = true
		if GetAdminUserFromContext(c) == nil {
			t.Fatal("expected admin user in context")
		}
		if GetAdminSessionFromContext(c) == nil {
			t.Fatal("expected admin session in context")
		}
		return c.NoContent(http.StatusNoContent)
	})

	if err := handler(c); err != nil {
		t.Fatalf("admin session auth returned error: %v", err)
	}
	if !called {
		t.Fatal("expected downstream handler to be called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}
}
