package models

import (
	"time"

	"gorm.io/gorm"
)

type OnboardingStatus string

const (
	OnboardingStatusPending    OnboardingStatus = "pending"
	OnboardingStatusInProgress OnboardingStatus = "in_progress"
	OnboardingStatusCompleted  OnboardingStatus = "completed"
	OnboardingStatusCancelled  OnboardingStatus = "cancelled"
)

// OnboardingProcess tracks the onboarding process for new employees
type OnboardingProcess struct {
	ID              uint             `gorm:"primaryKey" json:"id"`
	EmployeeID      uint             `gorm:"not null;uniqueIndex" json:"employee_id"`
	StartDate       time.Time        `gorm:"type:date;not null" json:"start_date"`
	ExpectedEndDate *time.Time       `gorm:"type:date" json:"expected_end_date,omitempty"`
	ActualEndDate   *time.Time       `gorm:"type:date" json:"actual_end_date,omitempty"`
	Status          OnboardingStatus `gorm:"type:varchar(50);default:'pending'" json:"status"`
	AssignedTo      *uint            `gorm:"index" json:"assigned_to,omitempty"`
	InitiatedBy     *uint            `gorm:"index" json:"initiated_by,omitempty"`
	Notes           *string          `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	DeletedAt       gorm.DeletedAt   `gorm:"index" json:"-"`

	Employee  Employee         `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	Assignee  *Employee        `gorm:"foreignKey:AssignedTo" json:"assignee,omitempty"`
	Initiator *Employee        `gorm:"foreignKey:InitiatedBy" json:"initiator,omitempty"`
	Tasks     []OnboardingTask `gorm:"foreignKey:OnboardingProcessID" json:"tasks,omitempty"`
}

func (OnboardingProcess) TableName() string {
	return "onboarding_processes"
}

type OnboardingTaskStatus string

const (
	OnboardingTaskStatusPending    OnboardingTaskStatus = "pending"
	OnboardingTaskStatusInProgress OnboardingTaskStatus = "in_progress"
	OnboardingTaskStatusCompleted  OnboardingTaskStatus = "completed"
	OnboardingTaskStatusSkipped    OnboardingTaskStatus = "skipped"
)

// OnboardingTask represents a task in the onboarding process
type OnboardingTask struct {
	ID                  uint                 `gorm:"primaryKey" json:"id"`
	OnboardingProcessID uint                 `gorm:"not null;index" json:"onboarding_process_id"`
	TaskName            string               `gorm:"size:200;not null" json:"task_name"`
	Description         *string              `gorm:"type:text" json:"description,omitempty"`
	Status              OnboardingTaskStatus `gorm:"type:varchar(50);default:'pending'" json:"status"`
	DueDate             *time.Time           `gorm:"type:date" json:"due_date,omitempty"`
	CompletedDate       *time.Time           `json:"completed_date,omitempty"`
	AssignedTo          *uint                `gorm:"index" json:"assigned_to,omitempty"`
	CompletedBy         *uint                `gorm:"index" json:"completed_by,omitempty"`
	IsRequired          bool                 `gorm:"default:true" json:"is_required"`
	Order               int                  `gorm:"default:0" json:"order"`
	Notes               *string              `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`

	OnboardingProcess OnboardingProcess `gorm:"foreignKey:OnboardingProcessID" json:"onboarding_process,omitempty"`
	Assignee          *Employee         `gorm:"foreignKey:AssignedTo" json:"assignee,omitempty"`
	Completer         *Employee         `gorm:"foreignKey:CompletedBy" json:"completer,omitempty"`
}

func (OnboardingTask) TableName() string {
	return "onboarding_tasks"
}

// OffboardingProcess tracks the offboarding process for departing employees
type OffboardingProcess struct {
	ID              uint             `gorm:"primaryKey" json:"id"`
	EmployeeID      uint             `gorm:"not null;uniqueIndex" json:"employee_id"`
	StartDate       time.Time        `gorm:"type:date;not null" json:"start_date"`
	ExpectedEndDate *time.Time       `gorm:"type:date" json:"expected_end_date,omitempty"`
	ActualEndDate   *time.Time       `gorm:"type:date" json:"actual_end_date,omitempty"`
	Status          OnboardingStatus `gorm:"type:varchar(50);default:'pending'" json:"status"`
	Reason          *string          `gorm:"type:text" json:"reason,omitempty"`
	AssignedTo      *uint            `gorm:"index" json:"assigned_to,omitempty"`
	InitiatedBy     *uint            `gorm:"index" json:"initiated_by,omitempty"`
	Notes           *string          `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	DeletedAt       gorm.DeletedAt   `gorm:"index" json:"-"`

	Employee  Employee          `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	Assignee  *Employee         `gorm:"foreignKey:AssignedTo" json:"assignee,omitempty"`
	Initiator *Employee         `gorm:"foreignKey:InitiatedBy" json:"initiator,omitempty"`
	Tasks     []OffboardingTask `gorm:"foreignKey:OffboardingProcessID" json:"tasks,omitempty"`
}

func (OffboardingProcess) TableName() string {
	return "offboarding_processes"
}

// OffboardingTask represents a task in the offboarding process
type OffboardingTask struct {
	ID                   uint                 `gorm:"primaryKey" json:"id"`
	OffboardingProcessID uint                 `gorm:"not null;index" json:"offboarding_process_id"`
	TaskName             string               `gorm:"size:200;not null" json:"task_name"`
	Description          *string              `gorm:"type:text" json:"description,omitempty"`
	Status               OnboardingTaskStatus `gorm:"type:varchar(50);default:'pending'" json:"status"`
	DueDate              *time.Time           `gorm:"type:date" json:"due_date,omitempty"`
	CompletedDate        *time.Time           `json:"completed_date,omitempty"`
	AssignedTo           *uint                `gorm:"index" json:"assigned_to,omitempty"`
	CompletedBy          *uint                `gorm:"index" json:"completed_by,omitempty"`
	IsRequired           bool                 `gorm:"default:true" json:"is_required"`
	Order                int                  `gorm:"default:0" json:"order"`
	Notes                *string              `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`

	OffboardingProcess OffboardingProcess `gorm:"foreignKey:OffboardingProcessID" json:"offboarding_process,omitempty"`
	Assignee           *Employee          `gorm:"foreignKey:AssignedTo" json:"assignee,omitempty"`
	Completer          *Employee          `gorm:"foreignKey:CompletedBy" json:"completer,omitempty"`
}

func (OffboardingTask) TableName() string {
	return "offboarding_tasks"
}
