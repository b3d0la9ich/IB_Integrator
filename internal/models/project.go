package models

import (
	"time"

	"gorm.io/gorm"
)

type ProjectType string
type ProjectStatus string

const (
	ProjectAudit       ProjectType = "audit"
	ProjectSZIDeploy   ProjectType = "szi_deploy"
	ProjectPentest     ProjectType = "pentest"
	ProjectMaintenance ProjectType = "maintenance"
	ProjectCompliance  ProjectType = "compliance"

	StatusPlanned    ProjectStatus = "planned"
	StatusInProgress ProjectStatus = "in_progress"
	StatusOnApproval ProjectStatus = "on_approval"
	StatusFinished   ProjectStatus = "finished"
	StatusCancelled  ProjectStatus = "cancelled"
)

type Project struct {
	gorm.Model
	ClientID uint
	Client   Client

	AssetID uint
	Asset   Asset

	Title       string        `gorm:"size:255;not null"`
	Type        ProjectType   `gorm:"type:varchar(50);not null"`
	Status      ProjectStatus `gorm:"type:varchar(50);not null"`
	Description string        `gorm:"type:text"`

	PlannedStart *time.Time
	PlannedEnd   *time.Time
	ActualEnd    *time.Time

	SalesID    uint // User.ID роли sales/admin
	EngineerID uint // User.ID роли engineer
}
