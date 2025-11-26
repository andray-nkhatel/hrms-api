package models

import (
	"time"

	"gorm.io/gorm"
)

type LeaveStatus string

const (
	StatusPending   LeaveStatus = "Pending"
	StatusApproved  LeaveStatus = "Approved"
	StatusRejected  LeaveStatus = "Rejected"
	StatusCancelled LeaveStatus = "Cancelled"
)

type Leave struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	EmployeeID      uint           `gorm:"not null;index" json:"employee_id"`
	LeaveTypeID     uint           `gorm:"not null;index" json:"leave_type_id"`
	StartDate       time.Time      `gorm:"type:date;not null" json:"start_date"`
	EndDate         time.Time      `gorm:"type:date;not null" json:"end_date"`
	Reason          string         `gorm:"type:text" json:"reason,omitempty"`
	Status          LeaveStatus    `gorm:"type:varchar(20);default:'Pending'" json:"status"`
	RejectionReason string         `gorm:"type:text" json:"rejection_reason,omitempty"`
	ApprovedBy      *uint          `gorm:"index" json:"approved_by,omitempty"`
	ApprovedAt      *time.Time     `json:"approved_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	Employee  Employee  `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	LeaveType LeaveType `gorm:"foreignKey:LeaveTypeID" json:"leave_type,omitempty"`
	Approver  *Employee `gorm:"foreignKey:ApprovedBy" json:"approver,omitempty"`
}

func (Leave) TableName() string {
	return "leaves"
}

// GetDuration returns the number of days for this leave (inclusive)
func (l *Leave) GetDuration() int {
	if l.EndDate.Before(l.StartDate) {
		return 0
	}
	duration := l.EndDate.Sub(l.StartDate)
	days := int(duration.Hours()/24) + 1
	return days
}
