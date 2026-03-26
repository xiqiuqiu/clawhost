package model

import (
	"strings"
	"time"

	authservice "github.com/clawhost/clawhost/service/auth"
	"github.com/clawhost/clawhost/util"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	AdminUserStatusActive   = "active"
	AdminUserStatusDisabled = "disabled"
)

type AdminUser struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Email        string    `json:"email" gorm:"type:varchar(255);uniqueIndex;not null"`
	Name         string    `json:"name" gorm:"type:varchar(255);not null"`
	PasswordHash string    `json:"-" gorm:"type:varchar(255);not null"`
	Status       string    `json:"status" gorm:"type:varchar(50);default:'active'"`
	IsBootstrap  bool      `json:"is_bootstrap" gorm:"default:false"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (AdminUser) TableName() string {
	return "admin_users"
}

func (u *AdminUser) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	u.Email = strings.TrimSpace(strings.ToLower(u.Email))
	if u.Status == "" {
		u.Status = AdminUserStatusActive
	}
	if u.PasswordHash != "" && !strings.HasPrefix(u.PasswordHash, "$2") {
		hashed, err := authservice.HashPassword(u.PasswordHash)
		if err != nil {
			return err
		}
		u.PasswordHash = hashed
	}
	return nil
}

func CreateAdminUser(user *AdminUser) error {
	return util.GetDB().Create(user).Error
}

func GetAdminUserByID(id string) (*AdminUser, error) {
	var user AdminUser
	if err := util.GetDB().Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func GetAdminUserByEmail(email string) (*AdminUser, error) {
	var user AdminUser
	email = strings.TrimSpace(strings.ToLower(email))
	if err := util.GetDB().Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func CountAdminUsers() (int64, error) {
	var count int64
	if err := util.GetDB().Model(&AdminUser{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func ListAdminUsers() ([]*AdminUser, error) {
	var users []*AdminUser
	if err := util.GetDB().Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func UpdateAdminUserStatus(id, status string) error {
	return util.GetDB().Model(&AdminUser{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}).Error
}

func SaveAdminUser(user *AdminUser) error {
	return util.GetDB().Save(user).Error
}

func HashAdminPassword(password string) (string, error) {
	return authservice.HashPassword(password)
}
