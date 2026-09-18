package models

import "time"

type Role struct{
	ID uint `gorm:"primaryKey;autoIncrement"`
	Code string `gorm:"type:varchar(15);not null;uniqueIndex:idx_roles_code"`
	Name string `gorm:"type:varchar(20);not null"`
	CreateAt *time.Time
	UpdateAt *time.Time
}