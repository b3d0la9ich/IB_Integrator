package models

import "gorm.io/gorm"

type Client struct {
	gorm.Model
	Name         string `gorm:"size:255;not null;uniqueIndex"`   // Название организации (уникально)
	OrgType      string `gorm:"size:100"`                        // Тип: банк, госорган, КИИ и т.п.
	INN          string `gorm:"size:12;uniqueIndex"`             // ИНН (при наличии — уникален)
	OGRN         string `gorm:"size:15"`                         // ОГРН
	Industry     string `gorm:"size:100"`                        // Отрасль
	ContactName  string `gorm:"size:255"`                        // ФИО контактного лица
	ContactPost  string `gorm:"size:255"`                        // Должность контакта
	ContactEmail string `gorm:"size:255;uniqueIndex"`            // Email контактного лица (уникален)
	ContactPhone string `gorm:"size:50;uniqueIndex"`             // Телефон контактного лица (уникален)
	Notes        string `gorm:"type:text"`                       // Комментарии

	Assets   []Asset
	Projects []Project
}
