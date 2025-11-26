package models

import (
	"time"

	"gorm.io/gorm"
)

type Role string

const (
	RoleEmployee Role = "employee"
	RoleManager  Role = "manager"
	RoleAdmin    Role = "admin"
)

type Employee struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	NRC          *string        `gorm:"uniqueIndex;size:20" json:"nrc,omitempty"`
	Username     *string        `gorm:"uniqueIndex;size:50" json:"username,omitempty"`
	Firstname    string         `gorm:"size:50;not null" json:"firstname"`
	Lastname     string         `gorm:"size:50;not null" json:"lastname"`
	Email        string         `gorm:"uniqueIndex;size:100" json:"email"`
	PasswordHash string         `gorm:"column:password_hash;not null;size:256" json:"-"`
	Department   string         `gorm:"size:50" json:"department"`
	Role         Role           `gorm:"type:varchar(50);default:'employee'" json:"role"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	Leaves []Leave `gorm:"foreignKey:EmployeeID" json:"leaves,omitempty"`
}

func (Employee) TableName() string {
	return "employees"
}
