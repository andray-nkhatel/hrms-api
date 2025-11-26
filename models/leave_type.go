package models

import (
	"time"

	"gorm.io/gorm"
)

type LeaveType struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:50;not null" json:"name"`
	MaxDays   int       `gorm:"not null" json:"max_days"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	
	Leaves []Leave `gorm:"foreignKey:LeaveTypeID" json:"leaves,omitempty"`
}

func (LeaveType) TableName() string {
	return "leave_types"
}

