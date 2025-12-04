package models

import (
	"time"

	"gorm.io/gorm"
)

// Position represents a job position in the organization
type Position struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	Code              string         `gorm:"uniqueIndex;size:50;not null" json:"code"`
	Title             string         `gorm:"size:100;not null" json:"title"`
	Description       *string        `gorm:"type:text" json:"description,omitempty"`
	Department        string         `gorm:"size:50;not null" json:"department"`
	Level             *string        `gorm:"size:50" json:"level,omitempty"`
	ReportsToPosition *uint          `gorm:"index" json:"reports_to_position,omitempty"`
	MinSalary         *float64       `json:"min_salary,omitempty"`
	MaxSalary         *float64       `json:"max_salary,omitempty"`
	IsActive          bool           `gorm:"default:true" json:"is_active"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	ReportsTo *Position  `gorm:"foreignKey:ReportsToPosition" json:"reports_to,omitempty"`
	Employees []Employee `gorm:"foreignKey:PositionID" json:"employees,omitempty"`
}

func (Position) TableName() string {
	return "positions"
}

// PositionAssignment tracks employee position assignments
type PositionAssignment struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	EmployeeID      uint           `gorm:"not null;index" json:"employee_id"`
	PositionID      uint           `gorm:"not null;index" json:"position_id"`
	StartDate       time.Time      `gorm:"type:date;not null" json:"start_date"`
	EndDate         *time.Time     `gorm:"type:date" json:"end_date,omitempty"`
	IsPrimary       bool           `gorm:"default:true" json:"is_primary"`
	Salary          *float64       `json:"salary,omitempty"`
	AssignedBy      *uint          `gorm:"index" json:"assigned_by,omitempty"`
	AssignmentNotes *string        `gorm:"type:text" json:"assignment_notes,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	Employee Employee  `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	Position Position  `gorm:"foreignKey:PositionID" json:"position,omitempty"`
	Assigner *Employee `gorm:"foreignKey:AssignedBy" json:"assigner,omitempty"`
}

func (PositionAssignment) TableName() string {
	return "position_assignments"
}
