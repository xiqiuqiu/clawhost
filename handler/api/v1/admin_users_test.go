package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	authmw "github.com/clawhost/clawhost/middleware"
	"github.com/clawhost/clawhost/model"
	"github.com/clawhost/clawhost/service/auth"
	"github.com/clawhost/clawhost/util"
	"github.com/labstack/echo/v4"
)

func setupAdminUserManagementTestDB(t *testing.T) {
	t.Helper()

	db := setupAdminAuthTestDB(t)
	migrateAdminPolicyModelsForTest(t, db)
}

func createPlatformAdminUser(t *testing.T, email string) *model.AdminUser {
	t.Helper()

	admin := &model.AdminUser{
		Email:        email,
		Name:         "Platform Admin",
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
		t.Fatalf("create platform admin membership: %v", err)
	}
	return admin
}

func TestListAdminUsersReturnsAccessSummary(t *testing.T) {
	setupAdminUserManagementTestDB(t)

	actor := createPlatformAdminUser(t, "admin@example.com")
	viewer := &model.AdminUser{
		Email:        "viewer@example.com",
		Name:         "Viewer",
		PasswordHash: "secret-123",
	}
	if err := model.CreateAdminUser(viewer); err != nil {
		t.Fatalf("create viewer admin: %v", err)
	}
	if err := model.CreateAdminMembership(&model.AdminMembership{
		AdminUserID: viewer.ID,
		Role:        model.AdminRoleViewer,
		ScopeType:   model.AdminScopePlatform,
	}); err != nil {
		t.Fatalf("create viewer membership: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/bot/api/v1/admin/admin-users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(authmw.ContextKeyAdminUser, actor)

	if err := ListAdminUsers(c); err != nil {
		t.Fatalf("ListAdminUsers returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp util.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	items, ok := resp.Data.([]interface{})
	if !ok || len(items) != 2 {
		t.Fatalf("expected 2 admin users, got %#v", resp.Data)
	}

	foundViewer := false
	for _, item := range items {
		record, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("expected admin user object, got %#v", item)
		}
		if record["email"] == viewer.Email {
			foundViewer = true
			roles, ok := record["roles"].([]interface{})
			if !ok || len(roles) != 1 || roles[0] != model.AdminRoleViewer {
				t.Fatalf("expected viewer role summary, got %#v", record["roles"])
			}
		}
	}
	if !foundViewer {
		t.Fatal("expected viewer admin user in list response")
	}
}

func TestCreateAdminUserCreatesMembershipAndAuditLog(t *testing.T) {
	setupAdminUserManagementTestDB(t)

	actor := createPlatformAdminUser(t, "admin@example.com")

	body, _ := json.Marshal(map[string]string{
		"email":    "operator@example.com",
		"name":     "Operator",
		"password": "secret-123",
		"role":     model.AdminRoleOperator,
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bot/api/v1/admin/admin-users", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(authmw.ContextKeyAdminUser, actor)

	if err := CreateAdminManagedUser(c); err != nil {
		t.Fatalf("CreateAdminManagedUser returned error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	created, err := model.GetAdminUserByEmail("operator@example.com")
	if err != nil {
		t.Fatalf("load created admin: %v", err)
	}
	if created.Name != "Operator" {
		t.Fatalf("expected name to be saved, got %q", created.Name)
	}

	access, err := model.ResolveAdminAccess(created.ID)
	if err != nil {
		t.Fatalf("resolve created admin access: %v", err)
	}
	if len(access.Roles) != 1 || access.Roles[0] != model.AdminRoleOperator {
		t.Fatalf("expected operator role, got %#v", access.Roles)
	}

	logs, err := model.ListAuditLogs(model.AuditLogFilter{Action: "admin.create"})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(logs))
	}
	if logs[0].TargetID != created.ID {
		t.Fatalf("expected created admin to be audit target, got %q", logs[0].TargetID)
	}
}

func TestUpdateAdminUserUpdatesRoleStatusPasswordAndAuditLog(t *testing.T) {
	setupAdminUserManagementTestDB(t)

	actor := createPlatformAdminUser(t, "admin@example.com")
	target := &model.AdminUser{
		Email:        "viewer@example.com",
		Name:         "Viewer",
		PasswordHash: "secret-123",
	}
	if err := model.CreateAdminUser(target); err != nil {
		t.Fatalf("create target admin: %v", err)
	}
	if err := model.CreateAdminMembership(&model.AdminMembership{
		AdminUserID: target.ID,
		Role:        model.AdminRoleViewer,
		ScopeType:   model.AdminScopePlatform,
	}); err != nil {
		t.Fatalf("create target membership: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"name":     "Viewer Updated",
		"role":     model.AdminRoleOperator,
		"status":   model.AdminUserStatusDisabled,
		"password": "secret-456",
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPut, "/bot/api/v1/admin/admin-users/"+target.ID, bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(target.ID)
	c.Set(authmw.ContextKeyAdminUser, actor)

	if err := UpdateAdminManagedUser(c); err != nil {
		t.Fatalf("UpdateAdminManagedUser returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	updated, err := model.GetAdminUserByID(target.ID)
	if err != nil {
		t.Fatalf("reload updated admin: %v", err)
	}
	if updated.Name != "Viewer Updated" {
		t.Fatalf("expected updated name, got %q", updated.Name)
	}
	if updated.Status != model.AdminUserStatusDisabled {
		t.Fatalf("expected disabled status, got %q", updated.Status)
	}
	if !auth.VerifyPassword(updated.PasswordHash, "secret-456") {
		t.Fatal("expected updated password to verify")
	}

	access, err := model.ResolveAdminAccess(target.ID)
	if err != nil {
		t.Fatalf("resolve updated admin access: %v", err)
	}
	if len(access.Roles) != 1 || access.Roles[0] != model.AdminRoleOperator {
		t.Fatalf("expected operator role after update, got %#v", access.Roles)
	}

	logs, err := model.ListAuditLogs(model.AuditLogFilter{Action: "admin.update"})
	if err != nil {
		t.Fatalf("list update audit logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 admin.update audit log, got %d", len(logs))
	}
	if logs[0].TargetID != target.ID {
		t.Fatalf("expected updated admin to be audit target, got %q", logs[0].TargetID)
	}
}

func TestCreateAdminUserRejectsAppScopedRole(t *testing.T) {
	setupAdminUserManagementTestDB(t)

	actor := createPlatformAdminUser(t, "admin@example.com")

	body, _ := json.Marshal(map[string]string{
		"email":    "appadmin@example.com",
		"name":     "App Admin",
		"password": "secret-123",
		"role":     model.AdminRoleAppAdmin,
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/bot/api/v1/admin/admin-users", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(authmw.ContextKeyAdminUser, actor)

	if err := CreateAdminManagedUser(c); err != nil {
		t.Fatalf("CreateAdminManagedUser returned error: %v", err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
