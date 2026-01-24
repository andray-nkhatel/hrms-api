package models

import (
	"time"

	"gorm.io/gorm"
)

// LeaveTaken represents leave taken by an employee, recorded by admin
// No approval workflow - admin records actual leave taken
type LeaveTaken struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	EmployeeID  uint           `gorm:"not null;index" json:"employee_id"`
	LeaveTypeID uint           `gorm:"not null;index" json:"leave_type_id"`
	StartDate   time.Time      `gorm:"type:date;not null" json:"start_date"`
	EndDate     time.Time      `gorm:"type:date;not null" json:"end_date"`
	DaysTaken   float64        `gorm:"not null" json:"days_taken"`        // Calculated days taken
	RecordedBy  uint           `gorm:"not null;index" json:"recorded_by"` // Admin user ID
	Remarks     *string        `gorm:"type:text" json:"remarks,omitempty"`
	RecordedAt  time.Time      `gorm:"not null" json:"recorded_at"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Employee  Employee  `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	LeaveType LeaveType `gorm:"foreignKey:LeaveTypeID" json:"leave_type,omitempty"`
	Recorder  Employee  `gorm:"foreignKey:RecordedBy" json:"recorder,omitempty"`
}

func (LeaveTaken) TableName() string {
	return "leave_taken"
}

// CalculateDaysTaken calculates the number of days taken (inclusive)
func (lt *LeaveTaken) CalculateDaysTaken() float64 {
	if lt.EndDate.Before(lt.StartDate) {
		return 0
	}
	duration := lt.EndDate.Sub(lt.StartDate)
	days := float64(duration.Hours()/24) + 1.0
	return days
}
