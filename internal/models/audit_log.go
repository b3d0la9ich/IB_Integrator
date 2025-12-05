package models

import "time"

type AuditLog struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time

	UserID uint
	User   User

	Entity   string `gorm:"size:50;not null"` // "client", "project", "asset"
	EntityID uint
	Action   string `gorm:"size:50;not null"` // "create", "status_change" и т.п.
	Details  string `gorm:"type:text"`
}
