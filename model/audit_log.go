package model

import (
	"encoding/json"
	"time"

	"github.com/clawhost/clawhost/util"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	AuditResultSuccess = "success"
	AuditResultFailure = "failure"
)

type AuditLog struct {
	ID          string          `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ActorAdminID string         `json:"actor_admin_id,omitempty" gorm:"type:varchar(36);index"`
	ActorEmail  string          `json:"actor_email,omitempty" gorm:"type:varchar(255);index"`
	AppID       string          `json:"app_id,omitempty" gorm:"type:varchar(36);index"`
	Action      string          `json:"action" gorm:"type:varchar(100);index;not null"`
	TargetType  string          `json:"target_type" gorm:"type:varchar(50);index;not null"`
	TargetID    string          `json:"target_id,omitempty" gorm:"type:varchar(36);index"`
	TargetLabel string          `json:"target_label,omitempty" gorm:"type:varchar(255)"`
	Result      string          `json:"result" gorm:"type:varchar(50);index;not null"`
	SourceIP    string          `json:"source_ip,omitempty" gorm:"type:varchar(64)"`
	UserAgent   string          `json:"user_agent,omitempty" gorm:"type:varchar(255)"`
	Metadata    json.RawMessage `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type AuditLogFilter struct {
	Actor    string
	AppID    string
	Action   string
	Result   string
	DateFrom *time.Time
	DateTo   *time.Time
	Limit    int
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

func (l *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	if l.Result == "" {
		l.Result = AuditResultSuccess
	}
	return nil
}

func CreateAuditLog(entry *AuditLog) error {
	return util.GetDB().Create(entry).Error
}

func ListAuditLogs(filter AuditLogFilter) ([]*AuditLog, error) {
	var logs []*AuditLog

	query := util.GetDB().Model(&AuditLog{}).Order("created_at DESC")
	if filter.Actor != "" {
		query = query.Where("actor_admin_id = ? OR actor_email LIKE ?", filter.Actor, "%"+filter.Actor+"%")
	}
	if filter.AppID != "" {
		query = query.Where("app_id = ?", filter.AppID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.Result != "" {
		query = query.Where("result = ?", filter.Result)
	}
	if filter.DateFrom != nil {
		query = query.Where("created_at >= ?", *filter.DateFrom)
	}
	if filter.DateTo != nil {
		query = query.Where("created_at <= ?", *filter.DateTo)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	if err := query.Limit(limit).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
