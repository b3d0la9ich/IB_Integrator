package models

import "gorm.io/gorm"

type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleSales    UserRole = "sales"
	RoleEngineer UserRole = "engineer"
	RoleViewer   UserRole = "viewer"
)

type User struct {
	gorm.Model
	Username     string   `gorm:"uniqueIndex;size:50;not null"`
	PasswordHash string   `gorm:"not null"`
	Role         UserRole `gorm:"type:varchar(20);not null"`
}
