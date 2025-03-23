package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uint       `gorm:"primaryKey;autoIncrement"`
	UUID        uuid.UUID  `gorm:"type:uuid;not null"`
	Name        string     `gorm:"varchar(100);not null"`
	Username    string     `gorm:"varchar(20);not null"`
	Password    string     `gorm:"varchar(255);not null"`
	PhoneNumber string     `gorm:"varchar(15);not null"`
	Email       string     `gorm:"varchar(100);not null"`
	RoleID      uint       `gorm:"type:uint;not null"`
	CreateAt    *time.Time `gorm:"autoCreateTime"`
	UpdateAt    *time.Time `gorm:"autoUpdateTime"`
	Role        Role       `gorm:"foreignKey:role_id;references:id;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}
