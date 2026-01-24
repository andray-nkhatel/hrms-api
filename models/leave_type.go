package models

import (
	"time"

	"gorm.io/gorm"
)

type LeaveType struct {
	ID                    uint           `gorm:"primaryKey" json:"id"`
	Name                  string         `gorm:"size:50;not null" json:"name"`
	AccrualRate           float64        `gorm:"not null;default:2.0" json:"accrual_rate"` // Days per month (e.g., 2.0)
	MaxDays               int            `gorm:"not null" json:"max_days"`
	AllowCarryOver        bool           `gorm:"default:false" json:"allow_carry_over"`                // Whether carry-over is allowed
	MaxCarryOverDays      *float64       `gorm:"default:0" json:"max_carry_over_days,omitempty"`       // Maximum days that can be carried over (nil = unlimited)
	CarryOverExpiryMonths *int           `gorm:"default:12" json:"carry_over_expiry_months,omitempty"` // Months before carry-over expires (nil = no expiry)
	CarryOverExpiryDate   *time.Time     `gorm:"type:date" json:"carry_over_expiry_date,omitempty"`    // Fixed expiry date (e.g., end of Q1)
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"-"`

	Leaves     []Leave          `gorm:"foreignKey:LeaveTypeID" json:"leaves,omitempty"`
	CarryOvers []LeaveCarryOver `gorm:"foreignKey:LeaveTypeID" json:"carry_overs,omitempty"`
}

func (LeaveType) TableName() string {
	return "leave_types"
}
