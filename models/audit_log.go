package models

import (
	"time"
)

type AuditEntityType string

const (
	AuditEntityEmployee    AuditEntityType = "employee"
	AuditEntityIdentity    AuditEntityType = "identity"
	AuditEntityEmployment  AuditEntityType = "employment"
	AuditEntityPosition    AuditEntityType = "position"
	AuditEntityDocument    AuditEntityType = "document"
	AuditEntityCompliance  AuditEntityType = "compliance"
	AuditEntityOnboarding  AuditEntityType = "onboarding"
	AuditEntityOffboarding AuditEntityType = "offboarding"
	AuditEntityLifecycle   AuditEntityType = "lifecycle"
	AuditEntityLeave       AuditEntityType = "leave"
	AuditEntityLeaveType   AuditEntityType = "leave_type"
)

// AuditLog provides comprehensive audit logging for all HR operations
type AuditLog struct {
	ID            uint            `gorm:"primaryKey" json:"id"`
	EntityType    AuditEntityType `gorm:"type:varchar(50);not null;index" json:"entity_type"`
	EntityID      uint            `gorm:"not null;index" json:"entity_id"`
	Action        AuditAction     `gorm:"type:varchar(50);not null" json:"action"`
	PerformedBy   uint            `gorm:"not null;index" json:"performed_by"`
	IPAddress     *string         `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	UserAgent     *string         `gorm:"type:text" json:"user_agent,omitempty"`
	RequestMethod *string         `gorm:"type:varchar(10)" json:"request_method,omitempty"`
	RequestPath   *string         `gorm:"type:varchar(500)" json:"request_path,omitempty"`
	OldValues     *string         `gorm:"type:jsonb" json:"old_values,omitempty"` // JSON representation of old values
	NewValues     *string         `gorm:"type:jsonb" json:"new_values,omitempty"` // JSON representation of new values
	Changes       *string         `gorm:"type:jsonb" json:"changes,omitempty"`    // JSON representation of what changed
	Comment       *string         `gorm:"type:text" json:"comment,omitempty"`
	CreatedAt     time.Time       `gorm:"index" json:"created_at"`

	Performer Employee `gorm:"foreignKey:PerformedBy" json:"performer,omitempty"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
