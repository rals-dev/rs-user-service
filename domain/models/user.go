package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uint       `gorm:"primaryKey;autoIncrement"`
	UUID        uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_users_uuid"`
	Name        string     `gorm:"type:varchar(100);not null"`
	Username    string     `gorm:"type:varchar(20);not null;uniqueIndex:idx_users_username"`
	Password    string     `gorm:"type:varchar(255);not null"`
	PhoneNumber string     `gorm:"type:varchar(15);not null"`
	Email       string     `gorm:"type:varchar(100);not null;uniqueIndex:idx_users_email"`
	RoleID      uint       `gorm:"not null"`
	CreateAt    *time.Time `gorm:"autoCreateTime"`
	UpdateAt    *time.Time `gorm:"autoUpdateTime"`
	Role        Role       `gorm:"foreignKey:RoleID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}
