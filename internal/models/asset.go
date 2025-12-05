package models

import "gorm.io/gorm"

type AssetType string

const (
	AssetISPD   AssetType = "ispdn"
	AssetGIS    AssetType = "gis"
	AssetASUTP  AssetType = "asutp"
	AssetCorpIT AssetType = "corp_net"
)

type Asset struct {
	gorm.Model
	ClientID uint
	Client   Client

	Name        string    `gorm:"size:255;not null"`
	AssetType   AssetType `gorm:"type:varchar(50);not null"`
	Category    string    `gorm:"size:100"` // класс ИСПДн, УЗ ГИС и т.п.
	Description string    `gorm:"type:text"`
}
