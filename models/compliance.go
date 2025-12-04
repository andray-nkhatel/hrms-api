package models

import (
	"time"

	"gorm.io/gorm"
)

type ComplianceStatus string

const (
	ComplianceStatusCompliant    ComplianceStatus = "compliant"
	ComplianceStatusNonCompliant ComplianceStatus = "non_compliant"
	ComplianceStatusPending      ComplianceStatus = "pending"
	ComplianceStatusExpired      ComplianceStatus = "expired"
)

// ComplianceRequirement represents a compliance requirement that employees must meet
type ComplianceRequirement struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Code           string         `gorm:"uniqueIndex;size:50;not null" json:"code"`
	Name           string         `gorm:"size:200;not null" json:"name"`
	Description    *string        `gorm:"type:text" json:"description,omitempty"`
	Category       *string        `gorm:"size:100" json:"category,omitempty"`
	IsMandatory    bool           `gorm:"default:true" json:"is_mandatory"`
	ValidityPeriod *int           `json:"validity_period,omitempty"` // in days
	ReminderDays   *int           `json:"reminder_days,omitempty"`   // days before expiry to send reminder
	IsActive       bool           `gorm:"default:true" json:"is_active"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	Records []ComplianceRecord `gorm:"foreignKey:RequirementID" json:"records,omitempty"`
}

func (ComplianceRequirement) TableName() string {
	return "compliance_requirements"
}

// ComplianceRecord tracks an employee's compliance with a specific requirement
type ComplianceRecord struct {
	ID                  uint             `gorm:"primaryKey" json:"id"`
	EmployeeID          uint             `gorm:"not null;index" json:"employee_id"`
	RequirementID       uint             `gorm:"not null;index" json:"requirement_id"`
	Status              ComplianceStatus `gorm:"type:varchar(50);default:'pending'" json:"status"`
	IssueDate           *time.Time       `gorm:"type:date" json:"issue_date,omitempty"`
	ExpiryDate          *time.Time       `gorm:"type:date" json:"expiry_date,omitempty"`
	LastVerifiedDate    *time.Time       `gorm:"type:date" json:"last_verified_date,omitempty"`
	VerifiedBy          *uint            `gorm:"index" json:"verified_by,omitempty"`
	DocumentID          *uint            `gorm:"index" json:"document_id,omitempty"`
	Notes               *string          `gorm:"type:text" json:"notes,omitempty"`
	NonComplianceReason *string          `gorm:"type:text" json:"non_compliance_reason,omitempty"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
	DeletedAt           gorm.DeletedAt   `gorm:"index" json:"-"`

	Employee    Employee              `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	Requirement ComplianceRequirement `gorm:"foreignKey:RequirementID" json:"requirement,omitempty"`
	Verifier    *Employee             `gorm:"foreignKey:VerifiedBy" json:"verifier,omitempty"`
	Document    *Document             `gorm:"foreignKey:DocumentID" json:"document,omitempty"`
}

func (ComplianceRecord) TableName() string {
	return "compliance_records"
}
