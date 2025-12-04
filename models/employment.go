package models

import (
	"time"

	"gorm.io/gorm"
)

type EmploymentStatus string

const (
	EmploymentStatusActive     EmploymentStatus = "active"
	EmploymentStatusOnLeave    EmploymentStatus = "on_leave"
	EmploymentStatusSuspended  EmploymentStatus = "suspended"
	EmploymentStatusTerminated EmploymentStatus = "terminated"
	EmploymentStatusResigned   EmploymentStatus = "resigned"
)

type EmploymentType string

const (
	EmploymentTypeFullTime   EmploymentType = "full_time"
	EmploymentTypePartTime   EmploymentType = "part_time"
	EmploymentTypeContract   EmploymentType = "contract"
	EmploymentTypeInternship EmploymentType = "internship"
	EmploymentTypeConsultant EmploymentType = "consultant"
)

// EmploymentDetails stores comprehensive employment information
type EmploymentDetails struct {
	ID                uint             `gorm:"primaryKey" json:"id"`
	EmployeeID        uint             `gorm:"not null;uniqueIndex" json:"employee_id"`
	EmployeeNumber    *string          `gorm:"uniqueIndex;size:50" json:"employee_number,omitempty"`
	EmploymentType    EmploymentType   `gorm:"type:varchar(50)" json:"employment_type"`
	EmploymentStatus  EmploymentStatus `gorm:"type:varchar(50);default:'active'" json:"employment_status"`
	HireDate          *time.Time       `gorm:"type:date" json:"hire_date,omitempty"`
	StartDate         *time.Time       `gorm:"type:date" json:"start_date,omitempty"`
	EndDate           *time.Time       `gorm:"type:date" json:"end_date,omitempty"`
	TerminationDate   *time.Time       `gorm:"type:date" json:"termination_date,omitempty"`
	TerminationReason *string          `gorm:"type:text" json:"termination_reason,omitempty"`
	ManagerID         *uint            `gorm:"index" json:"manager_id,omitempty"`
	WorkLocation      *string          `gorm:"size:100" json:"work_location,omitempty"`
	WorkSchedule      *string          `gorm:"size:50" json:"work_schedule,omitempty"`
	ProbationEndDate  *time.Time       `gorm:"type:date" json:"probation_end_date,omitempty"`
	ProbationStatus   *string          `gorm:"size:20" json:"probation_status,omitempty"`
	NoticePeriod      *int             `json:"notice_period,omitempty"` // in days
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
	DeletedAt         gorm.DeletedAt   `gorm:"index" json:"-"`

	Employee Employee  `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	Manager  *Employee `gorm:"foreignKey:ManagerID" json:"manager,omitempty"`
}

func (EmploymentDetails) TableName() string {
	return "employment_details"
}

// EmploymentHistory tracks employment history and changes
type EmploymentHistory struct {
	ID                 uint              `gorm:"primaryKey" json:"id"`
	EmployeeID         uint              `gorm:"not null;index" json:"employee_id"`
	PreviousStatus     *EmploymentStatus `gorm:"type:varchar(50)" json:"previous_status,omitempty"`
	NewStatus          EmploymentStatus  `gorm:"type:varchar(50);not null" json:"new_status"`
	PreviousPosition   *string           `gorm:"size:100" json:"previous_position,omitempty"`
	NewPosition        *string           `gorm:"size:100" json:"new_position,omitempty"`
	PreviousDepartment *string           `gorm:"size:50" json:"previous_department,omitempty"`
	NewDepartment      *string           `gorm:"size:50" json:"new_department,omitempty"`
	ChangeDate         time.Time         `gorm:"type:date;not null" json:"change_date"`
	ChangeReason       *string           `gorm:"type:text" json:"change_reason,omitempty"`
	ChangedBy          *uint             `gorm:"index" json:"changed_by,omitempty"`
	Notes              *string           `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`

	Employee Employee  `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	Changer  *Employee `gorm:"foreignKey:ChangedBy" json:"changer,omitempty"`
}

func (EmploymentHistory) TableName() string {
	return "employment_history"
}
