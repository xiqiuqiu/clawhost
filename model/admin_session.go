package model

import (
	"time"

	"github.com/clawhost/clawhost/service/auth"
	"github.com/clawhost/clawhost/util"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminSession struct {
	ID           string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	AdminUserID  string     `json:"admin_user_id" gorm:"type:varchar(36);index;not null"`
	SessionToken string     `json:"-" gorm:"type:varchar(255);uniqueIndex;not null"`
	ExpiresAt    time.Time  `json:"expires_at" gorm:"index;not null"`
	LastSeenAt   *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (AdminSession) TableName() string {
	return "admin_sessions"
}

func (s *AdminSession) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

func DefaultAdminSessionExpiry() time.Time {
	return time.Now().Add(24 * time.Hour)
}

func CreateAdminSession(adminUserID string, expiresAt time.Time) (*AdminSession, string, error) {
	rawToken, hashedToken, err := auth.GenerateSessionToken()
	if err != nil {
		return nil, "", err
	}

	session := &AdminSession{
		AdminUserID:  adminUserID,
		SessionToken: hashedToken,
		ExpiresAt:    expiresAt,
	}
	if err := util.GetDB().Create(session).Error; err != nil {
		return nil, "", err
	}
	return session, rawToken, nil
}

func GetActiveAdminSessionByToken(rawToken string) (*AdminSession, error) {
	var session AdminSession
	hashedToken := auth.HashToken(rawToken)
	if err := util.GetDB().Where("session_token = ? AND expires_at > ?", hashedToken, time.Now()).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func TouchAdminSession(id string) error {
	now := time.Now()
	return util.GetDB().Model(&AdminSession{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_seen_at": now,
		"updated_at":   now,
	}).Error
}

func RevokeAdminSession(id string) error {
	return util.GetDB().Where("id = ?", id).Delete(&AdminSession{}).Error
}

func RevokeAllAdminSessionsForUser(adminUserID string) error {
	return util.GetDB().Where("admin_user_id = ?", adminUserID).Delete(&AdminSession{}).Error
}

func PruneExpiredAdminSessions() error {
	return util.GetDB().Where("expires_at <= ?", time.Now()).Delete(&AdminSession{}).Error
}
