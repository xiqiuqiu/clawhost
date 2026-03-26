package model

import (
	"sort"
	"time"

	"github.com/clawhost/clawhost/util"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminRolePermission struct {
	ID            string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	RoleKey       string    `json:"role_key" gorm:"type:varchar(100);index:idx_role_permission,unique;not null"`
	PermissionKey string    `json:"permission_key" gorm:"type:varchar(100);index:idx_role_permission,unique;not null"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (AdminRolePermission) TableName() string {
	return "admin_role_permissions"
}

func (rp *AdminRolePermission) BeforeCreate(tx *gorm.DB) error {
	if rp.ID == "" {
		rp.ID = uuid.New().String()
	}
	return nil
}

func ListAdminRolePermissions(roleKey string) ([]string, error) {
	var items []*AdminRolePermission
	if err := util.GetDB().Where("role_key = ?", roleKey).Find(&items).Error; err != nil {
		return nil, err
	}
	permissions := make([]string, 0, len(items))
	for _, item := range items {
		permissions = append(permissions, item.PermissionKey)
	}
	sort.Strings(permissions)
	return permissions, nil
}

func ListAdminRolePermissionsByRoles(roleKeys []string) (map[string]map[string]bool, error) {
	permissionMap := make(map[string]map[string]bool, len(roleKeys))
	if len(roleKeys) == 0 {
		return permissionMap, nil
	}

	var items []*AdminRolePermission
	if err := util.GetDB().Where("role_key IN ?", roleKeys).Find(&items).Error; err != nil {
		return nil, err
	}

	for _, roleKey := range roleKeys {
		permissionMap[roleKey] = map[string]bool{}
	}
	for _, item := range items {
		if _, ok := permissionMap[item.RoleKey]; !ok {
			permissionMap[item.RoleKey] = map[string]bool{}
		}
		permissionMap[item.RoleKey][item.PermissionKey] = true
	}
	return permissionMap, nil
}

func ReplaceAdminRolePermissions(roleKey string, permissionKeys []string) error {
	return util.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_key = ?", roleKey).Delete(&AdminRolePermission{}).Error; err != nil {
			return err
		}
		for _, permissionKey := range permissionKeys {
			if err := tx.Create(&AdminRolePermission{
				RoleKey:       roleKey,
				PermissionKey: permissionKey,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
