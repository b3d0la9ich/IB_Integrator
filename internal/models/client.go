package models

import "gorm.io/gorm"

type Client struct {
	gorm.Model
	Name         string `gorm:"size:255;not null"`  // Название организации
	OrgType      string `gorm:"size:100"`          // Тип: банк, госорган, КИИ и т.п.
	INN          string `gorm:"size:12"`           // ИНН (необязательно)
	OGRN         string `gorm:"size:15"`           // ОГРН (если захочешь использовать)
	Industry     string `gorm:"size:100"`          // Отрасль
	ContactName  string `gorm:"size:255"`          // Если потом введёшь ФИО контакта
	ContactPost  string `gorm:"size:255"`          // Должность контакта
	ContactEmail string `gorm:"size:255"`          // Email контактного лица
	ContactPhone string `gorm:"size:50"`           // Телефон контактного лица
	Notes        string `gorm:"type:text"`         // Комментарии о клиенте / инфраструктуре

	Assets   []Asset
	Projects []Project
}
