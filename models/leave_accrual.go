package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// LeaveAccrual tracks monthly leave accruals for employees
// Supports both simplified schema (Year/Month) and full schema (AccrualMonth with balance tracking)
type LeaveAccrual struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	EmployeeID   uint           `gorm:"not null;index" json:"employee_id"`
	LeaveTypeID  uint           `gorm:"not null;index" json:"leave_type_id"`
	Year         int            `gorm:"index" json:"year"`                           // Year (e.g., 2026) - for simplified schema
	Month        int            `gorm:"index" json:"month"`                          // Month (1-12) - for simplified schema
	AccrualMonth *time.Time     `gorm:"index" json:"accrual_month,omitempty"`        // Accrual month (for full schema)
	DaysAccrued  float64        `gorm:"not null;default:0" json:"days_accrued"`      // Days accrued (e.g., 2.0)
	DaysUsed     float64        `gorm:"default:0" json:"days_used,omitempty"`        // Days used in this month
	DaysBalance  float64        `gorm:"default:0" json:"days_balance,omitempty"`     // Running balance
	IsProcessed  bool           `gorm:"default:false" json:"is_processed,omitempty"` // Whether this accrual has been processed
	ProcessedAt  *time.Time     `json:"processed_at,omitempty"`                      // When this accrual was processed
	Notes        *string        `json:"notes,omitempty"`                             // Notes about manual adjustments or processing
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
// Handles both AccrualMonth (time.Time) and Year/Month (int) fields
func (la *LeaveAccrual) GetAccrualMonthKey() string {
	// If AccrualMonth is set, use it
	if la.AccrualMonth != nil {
		return fmt.Sprintf("%d-%02d", la.AccrualMonth.Year(), int(la.AccrualMonth.Month()))
	}
	// Otherwise, use Year and Month fields
	if la.Year > 0 && la.Month > 0 {
		return fmt.Sprintf("%d-%02d", la.Year, la.Month)
	}
	// Fallback: return empty string if neither is set
	return ""
}
