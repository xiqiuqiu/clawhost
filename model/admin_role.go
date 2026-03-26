package model

import (
	"time"

	"github.com/clawhost/clawhost/util"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminRole struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Key         string    `json:"key" gorm:"type:varchar(100);uniqueIndex;not null"`
	Name        string    `json:"name" gorm:"type:varchar(255);not null"`
	Description string    `json:"description" gorm:"type:text"`
	ScopeType   string    `json:"scope_type" gorm:"type:varchar(50);not null"`
	IsSystem    bool      `json:"is_system" gorm:"default:false"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (AdminRole) TableName() string {
	return "admin_roles"
}

func (r *AdminRole) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return nil
}

func ListAdminRoles() ([]*AdminRole, error) {
	var roles []*AdminRole
	if err := util.GetDB().Order("scope_type ASC, key ASC").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func ListAdminRolesByScope(scopeType string) ([]*AdminRole, error) {
	var roles []*AdminRole
	if err := util.GetDB().Where("scope_type = ?", scopeType).Order("key ASC").Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func GetAdminRoleByKey(key string) (*AdminRole, error) {
	var role AdminRole
	if err := util.GetDB().Where("key = ?", key).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}
