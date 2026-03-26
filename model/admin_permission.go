package model

import (
	"time"

	"github.com/clawhost/clawhost/util"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminPermission struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Key          string    `json:"key" gorm:"type:varchar(100);uniqueIndex;not null"`
	Name         string    `json:"name" gorm:"type:varchar(255);not null"`
	Description  string    `json:"description" gorm:"type:text"`
	ResourceType string    `json:"resource_type" gorm:"type:varchar(100);not null"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (AdminPermission) TableName() string {
	return "admin_permissions"
}

func (p *AdminPermission) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

func ListAdminPermissions() ([]*AdminPermission, error) {
	var permissions []*AdminPermission
	if err := util.GetDB().Order("resource_type ASC, key ASC").Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

func GetAdminPermissionByKey(key string) (*AdminPermission, error) {
	var permission AdminPermission
	if err := util.GetDB().Where("key = ?", key).First(&permission).Error; err != nil {
		return nil, err
	}
	return &permission, nil
}
