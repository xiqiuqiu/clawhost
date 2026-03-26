package model

import (
	"time"

	"github.com/clawhost/clawhost/util"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminMembership struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	AdminUserID string    `json:"admin_user_id" gorm:"type:varchar(36);index;not null"`
	Role        string    `json:"role" gorm:"type:varchar(100);not null"`
	ScopeType   string    `json:"scope_type" gorm:"type:varchar(50);index;not null"`
	ScopeID     string    `json:"scope_id,omitempty" gorm:"type:varchar(36);index"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (AdminMembership) TableName() string {
	return "admin_memberships"
}

func (m *AdminMembership) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

func CreateAdminMembership(membership *AdminMembership) error {
	return util.GetDB().Create(membership).Error
}

func ListAdminMembershipsByUser(adminUserID string) ([]*AdminMembership, error) {
	var memberships []*AdminMembership
	if err := util.GetDB().Where("admin_user_id = ?", adminUserID).Find(&memberships).Error; err != nil {
		return nil, err
	}
	return memberships, nil
}

func AdminHasPermission(adminUserID, permission, appID string) (bool, error) {
	memberships, err := ListAdminMembershipsByUser(adminUserID)
	if err != nil {
		return false, err
	}

	for _, membership := range memberships {
		if !RoleHasPermission(membership.Role, permission) {
			continue
		}
		if membership.ScopeType == AdminScopePlatform {
			return true, nil
		}
		if membership.ScopeType == AdminScopeApp && appID != "" && membership.ScopeID == appID {
			return true, nil
		}
	}

	return false, nil
}
