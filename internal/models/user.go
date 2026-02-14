package models

import (
	"time"

	"gorm.io/gorm"
)

type Role struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:50;unique;not null" json:"name"` // 'admin', 'manager', 'inventory', 'biller'
	CreatedAt time.Time `json:"created_at"`
	Users     []User    `json:"-"`
}

type User struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	EmployeeID     string         `gorm:"size:20;unique;not null" json:"employee_id"`
	Username       string         `gorm:"size:50;not null" json:"username"`
	Mobile         string         `gorm:"size:15" json:"mobile"`
	PasswordHash   string         `gorm:"size:255;not null" json:"-"`
	RoleID         uint           `json:"role_id"`
	Role           Role           `gorm:"foreignKey:RoleID" json:"role"`
	IsActive       bool           `gorm:"default:true" json:"is_active"`
	InactiveReason string         `gorm:"type:text" json:"inactive_reason"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type LoginHistory struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `json:"user_id"`
	User       User       `gorm:"foreignKey:UserID" json:"user"`
	LoginTime  time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"login_time"`
	LogoutTime *time.Time `json:"logout_time"`
	IPAddress  string     `gorm:"size:45" json:"ip_address"`
}
