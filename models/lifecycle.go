package models

import (
	"time"
)

type LifecycleEventType string

const (
	LifecycleEventHired        LifecycleEventType = "hired"
	LifecycleEventOnboarded    LifecycleEventType = "onboarded"
	LifecycleEventPromoted     LifecycleEventType = "promoted"
	LifecycleEventTransferred  LifecycleEventType = "transferred"
	LifecycleEventDemoted      LifecycleEventType = "demoted"
	LifecycleEventResigned     LifecycleEventType = "resigned"
	LifecycleEventTerminated   LifecycleEventType = "terminated"
	LifecycleEventRetired      LifecycleEventType = "retired"
	LifecycleEventOffboarded   LifecycleEventType = "offboarded"
	LifecycleEventStatusChange LifecycleEventType = "status_change"
)

// WorkLifecycleEvent tracks significant events in an employee's work lifecycle
type WorkLifecycleEvent struct {
	ID             uint               `gorm:"primaryKey" json:"id"`
	EmployeeID     uint               `gorm:"not null;index" json:"employee_id"`
	EventType      LifecycleEventType `gorm:"type:varchar(50);not null" json:"event_type"`
	EventDate      time.Time          `gorm:"type:date;not null" json:"event_date"`
	EffectiveDate  *time.Time         `gorm:"type:date" json:"effective_date,omitempty"`
	PreviousValue  *string            `gorm:"type:text" json:"previous_value,omitempty"`
	NewValue       *string            `gorm:"type:text" json:"new_value,omitempty"`
	Description    *string            `gorm:"type:text" json:"description,omitempty"`
	InitiatedBy    *uint              `gorm:"index" json:"initiated_by,omitempty"`
	ApprovedBy     *uint              `gorm:"index" json:"approved_by,omitempty"`
	ApprovedAt     *time.Time         `json:"approved_at,omitempty"`
	IsCompleted    bool               `gorm:"default:false" json:"is_completed"`
	CompletionDate *time.Time         `json:"completion_date,omitempty"`
	Notes          *string            `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`

	Employee  Employee  `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	Initiator *Employee `gorm:"foreignKey:InitiatedBy" json:"initiator,omitempty"`
	Approver  *Employee `gorm:"foreignKey:ApprovedBy" json:"approver,omitempty"`
}

func (WorkLifecycleEvent) TableName() string {
	return "work_lifecycle_events"
}
