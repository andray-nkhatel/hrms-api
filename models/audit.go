package models

import (
	"time"
)

type AuditAction string

const (
	AuditActionCreate  AuditAction = "CREATE"
	AuditActionApprove AuditAction = "APPROVE"
	AuditActionReject  AuditAction = "REJECT"
	AuditActionCancel  AuditAction = "CANCEL"
	AuditActionUpdate  AuditAction = "UPDATE"
	AuditActionDelete  AuditAction = "DELETE"
)

type LeaveAudit struct {
	ID          uint        `gorm:"primaryKey" json:"id"`
	LeaveID     uint        `gorm:"not null;index" json:"leave_id"`
	Action      AuditAction `gorm:"type:varchar(20);not null" json:"action"`
	PerformedBy uint        `gorm:"not null;index" json:"performed_by"`
	OldStatus   string      `gorm:"type:varchar(20)" json:"old_status,omitempty"`
	NewStatus   string      `gorm:"type:varchar(20)" json:"new_status,omitempty"`
	Comment     string      `gorm:"type:text" json:"comment,omitempty"`
	IPAddress   string      `gorm:"type:varchar(45)" json:"ip_address,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`

	Leave     Leave    `gorm:"foreignKey:LeaveID" json:"leave,omitempty"`
	Performer Employee `gorm:"foreignKey:PerformedBy" json:"performer,omitempty"`
}

func (LeaveAudit) TableName() string {
	return "leave_audits"
}
