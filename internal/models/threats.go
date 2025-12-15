package models

import "gorm.io/gorm"

// Каталог угроз (по STRIDE / ФСТЭК / своей классификации)
type Threat struct {
	gorm.Model
	Code        string `gorm:"size:32;uniqueIndex"` // Например: T1, STRIDE-S, УБИ.001
	Name        string `gorm:"size:255;not null"`   // Краткое название угрозы
	Category    string `gorm:"size:64"`             // STRIDE, Техногенная, Несанкционированный доступ и т.п.
	Description string `gorm:"type:text"`           // Подробное описание
}

// Каталог мер / контролей / мероприятий по ИБ
type ControlMeasure struct {
	gorm.Model
	Code        string `gorm:"size:32;uniqueIndex"`
	Name        string `gorm:"size:255;not null"` // Например: Настройка МЭ, Внедрение СКЗИ
	Standard    string `gorm:"size:128"`          // Ссылка на ФСТЭК, ГОСТ, ISO и т.п.
	Description string `gorm:"type:text"`
}

// Связь "угроза для конкретного объекта защиты"
type AssetThreat struct {
	ID uint `gorm:"primaryKey"`

	AssetID  uint
	ThreatID uint

	RiskLevel string `gorm:"size:16"`   // low / medium / high
	Notes     string `gorm:"type:text"` // комментарии по риску / обоснование

	Asset  Asset
	Threat Threat
}
