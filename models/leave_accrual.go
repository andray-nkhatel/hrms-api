package models

import (
	"time"

	"gorm.io/gorm"
)

// LeaveAccrual tracks monthly leave accruals for employees
type LeaveAccrual struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	EmployeeID   uint           `gorm:"not null;index" json:"employee_id"`
	LeaveTypeID  uint           `gorm:"not null;index" json:"leave_type_id"`
	AccrualMonth time.Time      `gorm:"type:date;not null;index" json:"accrual_month"` // First day of the month
	DaysAccrued  float64        `gorm:"not null;default:0" json:"days_accrued"`        // Can be fractional (e.g., 2.0)
	DaysUsed     float64        `gorm:"default:0" json:"days_used"`
	DaysBalance  float64        `gorm:"default:0" json:"days_balance"`
	IsProcessed  bool           `gorm:"default:false" json:"is_processed"`
	ProcessedAt  *time.Time     `json:"processed_at,omitempty"`
	Notes        *string        `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	Employee  Employee  `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	LeaveType LeaveType `gorm:"foreignKey:LeaveTypeID" json:"leave_type,omitempty"`
}

func (LeaveAccrual) TableName() string {
	return "leave_accruals"
}

// GetAccrualMonthKey returns a string key for the accrual month (YYYY-MM format)
func (la *LeaveAccrual) GetAccrualMonthKey() string {
	return la.AccrualMonth.Format("2006-01")
}
