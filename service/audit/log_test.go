package audit

import (
	"fmt"
	"strings"
	"testing"

	"github.com/clawhost/clawhost/model"
	"github.com/clawhost/clawhost/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupAuditServiceTestDB(t *testing.T) *gorm.DB {
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

func TestWriteAuditLog(t *testing.T) {
	db := setupAuditServiceTestDB(t)
	if err := db.AutoMigrate(&model.AdminUser{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate audit models: %v", err)
	}

	admin := &model.AdminUser{
		Email:        "audit@example.com",
		Name:         "Audit Admin",
		PasswordHash: "secret-123",
	}
	if err := model.CreateAdminUser(admin); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	entry, err := Write(RequestContext{
		ActorAdminID: admin.ID,
		ActorEmail:   admin.Email,
		SourceIP:     "127.0.0.1",
		UserAgent:    "go-test",
	}, Entry{
		Action:      "app.create",
		AppID:       "app-123",
		TargetType:  "app",
		TargetID:    "app-123",
		TargetLabel: "Audit App",
		Result:      model.AuditResultSuccess,
		Metadata: map[string]interface{}{
			"owner_email": "owner@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if entry.ID == "" {
		t.Fatal("expected audit log ID to be set")
	}
	if entry.ActorAdminID != admin.ID {
		t.Fatalf("expected actor admin ID %q, got %q", admin.ID, entry.ActorAdminID)
	}
	if entry.SourceIP != "127.0.0.1" {
		t.Fatalf("expected source IP to be persisted, got %q", entry.SourceIP)
	}
	if entry.TargetID != "app-123" {
		t.Fatalf("expected target ID to be persisted, got %q", entry.TargetID)
	}
}

func TestWriteFailedAuditLog(t *testing.T) {
	db := setupAuditServiceTestDB(t)
	if err := db.AutoMigrate(&model.AdminUser{}, &model.AuditLog{}); err != nil {
		t.Fatalf("migrate audit models: %v", err)
	}

	entry, err := Write(RequestContext{
		ActorAdminID: "admin-123",
		ActorEmail:   "audit@example.com",
		SourceIP:     "10.0.0.9",
	}, Entry{
		Action:      "bot.stop",
		AppID:       "app-123",
		TargetType:  "bot",
		TargetID:    "bot-123",
		TargetLabel: "Failed Bot",
		Result:      model.AuditResultFailure,
		Metadata: map[string]interface{}{
			"reason": "bot is not running",
		},
	})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if entry.Result != model.AuditResultFailure {
		t.Fatalf("expected failed result, got %q", entry.Result)
	}
	if !strings.Contains(string(entry.Metadata), "bot is not running") {
		t.Fatalf("expected failure metadata to be persisted, got %s", string(entry.Metadata))
	}
}
