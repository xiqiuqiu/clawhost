package model

import (
	"sort"
	"time"

	"github.com/clawhost/clawhost/util"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminMembershipGrant struct {
	Role       string
	RoleKeys    []string
	ScopeMode  string
	AppScopeIDs []string
}

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

func ReplaceAdminMemberships(adminUserID string, memberships []*AdminMembership) error {
	return util.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("admin_user_id = ?", adminUserID).Delete(&AdminMembership{}).Error; err != nil {
			return err
		}

		for _, membership := range memberships {
			membership.AdminUserID = adminUserID
			if err := tx.Create(membership).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func ResolveAdminMembershipGrant(memberships []*AdminMembership) (*AdminMembershipGrant, error) {
	grant := &AdminMembershipGrant{
		RoleKeys:    []string{},
		ScopeMode:   AdminScopeModePlatform,
		AppScopeIDs: []string{},
	}
	if len(memberships) == 0 {
		return grant, nil
	}

	roleSet := make(map[string]struct{})
	appIDSet := make(map[string]struct{})
	hasPlatformScope := false

	for _, membership := range memberships {
		if membership == nil {
			continue
		}
		if membership.Role == "" {
			continue
		}
		if grant.Role == "" {
			grant.Role = membership.Role
		} else if grant.Role != membership.Role {
			return nil, ErrAdminMembershipMixedRoles
		}
		if _, ok := roleSet[membership.Role]; !ok {
			roleSet[membership.Role] = struct{}{}
			grant.RoleKeys = append(grant.RoleKeys, membership.Role)
		}
		if membership.ScopeType == AdminScopePlatform {
			hasPlatformScope = true
			continue
		}
		if membership.ScopeType == AdminScopeApp && membership.ScopeID != "" {
			if _, ok := appIDSet[membership.ScopeID]; ok {
				continue
			}
			appIDSet[membership.ScopeID] = struct{}{}
			grant.AppScopeIDs = append(grant.AppScopeIDs, membership.ScopeID)
		}
	}

	if !hasPlatformScope && len(grant.AppScopeIDs) > 0 {
		grant.ScopeMode = AdminScopeModeSelectedApps
	}

	sort.Strings(grant.RoleKeys)
	sort.Strings(grant.AppScopeIDs)
	return grant, nil
}

func AdminHasPermission(adminUserID, permission, appID string) (bool, error) {
	memberships, err := ListAdminMembershipsByUser(adminUserID)
	if err != nil {
		return false, err
	}

	grant, err := ResolveAdminMembershipGrant(memberships)
	if err != nil {
		return false, err
	}
	rolePermissionMap, err := ListAdminRolePermissionsByRoles(grant.RoleKeys)
	if err != nil {
		return false, err
	}

	for _, membership := range memberships {
		if !rolePermissionMap[membership.Role][permission] {
			continue
		}
		if membership.ScopeType == AdminScopePlatform {
			return true, nil
		}
		if membership.ScopeType == AdminScopeApp && appID == "" && supportsScopedPermissionWithoutTarget(permission) {
			return true, nil
		}
		if membership.ScopeType == AdminScopeApp && appID != "" && membership.ScopeID == appID {
			return true, nil
		}
	}

	return false, nil
}

type AdminAuditAccess struct {
	CanReadAll bool
	AppIDs     []string
}

func ResolveAdminAuditAccess(adminUserID string) (*AdminAuditAccess, error) {
	memberships, err := ListAdminMembershipsByUser(adminUserID)
	if err != nil {
		return nil, err
	}

	grant, err := ResolveAdminMembershipGrant(memberships)
	if err != nil {
		return nil, err
	}
	rolePermissionMap, err := ListAdminRolePermissionsByRoles(grant.RoleKeys)
	if err != nil {
		return nil, err
	}

	access := &AdminAuditAccess{
		AppIDs: []string{},
	}
	appIDSet := make(map[string]struct{})

	for _, membership := range memberships {
		if !rolePermissionMap[membership.Role][PermissionAuditRead] {
			continue
		}
		if membership.ScopeType == AdminScopePlatform {
			access.CanReadAll = true
			access.AppIDs = nil
			return access, nil
		}
		if membership.ScopeType == AdminScopeApp && membership.ScopeID != "" {
			if _, ok := appIDSet[membership.ScopeID]; ok {
				continue
			}
			appIDSet[membership.ScopeID] = struct{}{}
			access.AppIDs = append(access.AppIDs, membership.ScopeID)
		}
	}

	return access, nil
}

func supportsScopedPermissionWithoutTarget(permission string) bool {
	switch permission {
	case PermissionAppsRead, PermissionBotsRead, PermissionAuditRead, PermissionBotsUpgrade:
		return true
	default:
		return false
	}
}
