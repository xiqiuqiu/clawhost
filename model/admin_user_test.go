package model

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/clawhost/clawhost/service/auth"
	"github.com/clawhost/clawhost/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupModelTestDB(t *testing.T) *gorm.DB {
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

func TestAdminUserBeforeCreateGeneratesIDAndDefaults(t *testing.T) {
	db := setupModelTestDB(t)
	if err := db.AutoMigrate(&AdminUser{}); err != nil {
		t.Fatalf("migrate admin users: %v", err)
	}

	user := &AdminUser{
		Email:        "admin@example.com",
		Name:         "Admin",
		PasswordHash: "hashed-password",
	}
	if err := CreateAdminUser(user); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	if user.ID == "" {
		t.Fatal("expected generated admin user ID")
	}
	if user.Status != AdminUserStatusActive {
		t.Fatalf("expected default status %q, got %q", AdminUserStatusActive, user.Status)
	}
}

func TestAdminUserBeforeCreateHashesPassword(t *testing.T) {
	db := setupModelTestDB(t)
	if err := db.AutoMigrate(&AdminUser{}); err != nil {
		t.Fatalf("migrate admin users: %v", err)
	}

	user := &AdminUser{
		Email:        "hash@example.com",
		Name:         "Hash",
		PasswordHash: "plain-password",
	}
	if err := CreateAdminUser(user); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	if user.PasswordHash == "plain-password" {
		t.Fatal("expected password to be hashed before persistence")
	}
	if !strings.HasPrefix(user.PasswordHash, "$2") {
		t.Fatalf("expected bcrypt-style hash, got %q", user.PasswordHash)
	}
	if !auth.VerifyPassword(user.PasswordHash, "plain-password") {
		t.Fatal("expected hashed password to verify against original plain password")
	}
}

func TestAdminSessionCreateGeneratesToken(t *testing.T) {
	db := setupModelTestDB(t)
	if err := db.AutoMigrate(&AdminUser{}, &AdminSession{}); err != nil {
		t.Fatalf("migrate admin models: %v", err)
	}

	user := &AdminUser{
		Email:        "session@example.com",
		Name:         "Session",
		PasswordHash: "plain-password",
	}
	if err := CreateAdminUser(user); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	session, rawToken, err := CreateAdminSession(user.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("create admin session: %v", err)
	}

	if rawToken == "" {
		t.Fatal("expected raw session token to be returned")
	}
	if session.SessionToken == "" {
		t.Fatal("expected stored session token value")
	}
	if session.SessionToken == rawToken {
		t.Fatal("expected stored session token to differ from raw token")
	}
}

func TestGetActiveAdminSessionRejectsExpiredSessions(t *testing.T) {
	db := setupModelTestDB(t)
	if err := db.AutoMigrate(&AdminUser{}, &AdminSession{}); err != nil {
		t.Fatalf("migrate admin models: %v", err)
	}

	user := &AdminUser{
		Email:        "expired@example.com",
		Name:         "Expired",
		PasswordHash: "plain-password",
	}
	if err := CreateAdminUser(user); err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	session, rawToken, err := CreateAdminSession(user.ID, time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatalf("create expired admin session: %v", err)
	}

	got, err := GetActiveAdminSessionByToken(rawToken)
	if err == nil {
		t.Fatalf("expected expired session lookup to fail, got session %+v", got)
	}
	if got != nil {
		t.Fatalf("expected no expired session, got %+v", got)
	}

	if session.ID == "" {
		t.Fatal("expected expired session to still be created for lookup test")
	}
}
