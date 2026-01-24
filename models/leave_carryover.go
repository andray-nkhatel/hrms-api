package models

import (
	"time"

	"gorm.io/gorm"
)

// LeaveCarryOver tracks carry-over leave from one year to the next
type LeaveCarryOver struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	EmployeeID      uint           `gorm:"not null;index" json:"employee_id"`
	LeaveTypeID     uint           `gorm:"not null;index" json:"leave_type_id"`
	FromYear        int            `gorm:"not null;index" json:"from_year"` // Year the leave was accrued
	ToYear          int            `gorm:"not null;index" json:"to_year"`   // Year the leave is carried to
	DaysCarriedOver float64        `gorm:"not null;default:0" json:"days_carried_over"`
	DaysUsed        float64        `gorm:"default:0" json:"days_used"`             // Days used from this carry-over
	DaysRemaining   float64        `gorm:"default:0" json:"days_remaining"`        // Remaining days from this carry-over
	ExpiryDate      *time.Time     `gorm:"type:date" json:"expiry_date,omitempty"` // When carry-over expires (if applicable)
	IsExpired       bool           `gorm:"default:false" json:"is_expired"`
	ProcessedAt     time.Time      `gorm:"not null" json:"processed_at"`
	ProcessedBy     *uint          `gorm:"index" json:"processed_by,omitempty"`
	Notes           *string        `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	Employee  Employee  `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	LeaveType LeaveType `gorm:"foreignKey:LeaveTypeID" json:"leave_type,omitempty"`
	Processor *Employee `gorm:"foreignKey:ProcessedBy" json:"processor,omitempty"`
}

func (LeaveCarryOver) TableName() string {
	return "leave_carryovers"
}
