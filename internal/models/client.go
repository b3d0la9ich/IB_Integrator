package models

import "gorm.io/gorm"

type Client struct {
    gorm.Model
    Name         string `gorm:"size:255;not null;uniqueIndex"`
    OrgType      string `gorm:"size:100"`
    INN          string `gorm:"size:12;uniqueIndex"`
    OGRN         string `gorm:"size:15"`
    Industry     string `gorm:"size:100"`
    ContactName  string `gorm:"size:255"`
    ContactPost  string `gorm:"size:255"`
    ContactEmail string `gorm:"size:255;uniqueIndex"`
    ContactPhone string `gorm:"size:50;uniqueIndex"`
    Notes        string `gorm:"type:text"`

    Assets []Asset
}
