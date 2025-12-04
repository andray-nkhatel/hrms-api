package models

import (
	"time"

	"gorm.io/gorm"
)

type Role string

const (
	RoleEmployee Role = "employee"
	RoleManager  Role = "manager"
	RoleAdmin    Role = "admin"
)

type Employee struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	NRC          *string        `gorm:"uniqueIndex;size:20" json:"nrc,omitempty"`
	Username     *string        `gorm:"uniqueIndex;size:50" json:"username,omitempty"`
	Firstname    string         `gorm:"size:50;not null" json:"firstname"`
	Lastname     string         `gorm:"size:50;not null" json:"lastname"`
	Email        string         `gorm:"uniqueIndex;size:100" json:"email"`
	PasswordHash string         `gorm:"column:password_hash;not null;size:256" json:"-"`
	Department   string         `gorm:"size:50" json:"department"`
	PositionID   *uint          `gorm:"index" json:"position_id,omitempty"`
	Role         Role           `gorm:"type:varchar(50);default:'employee'" json:"role"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Leaves              []Leave              `gorm:"foreignKey:EmployeeID" json:"leaves,omitempty"`
	Identity            *IdentityInformation `gorm:"foreignKey:EmployeeID" json:"identity,omitempty"`
	Employment          *EmploymentDetails   `gorm:"foreignKey:EmployeeID" json:"employment,omitempty"`
	EmploymentHistory   []EmploymentHistory  `gorm:"foreignKey:EmployeeID" json:"employment_history,omitempty"`
	Position            *Position            `gorm:"foreignKey:PositionID" json:"position,omitempty"`
	PositionAssignments []PositionAssignment `gorm:"foreignKey:EmployeeID" json:"position_assignments,omitempty"`
	Documents           []Document           `gorm:"foreignKey:EmployeeID" json:"documents,omitempty"`
	LifecycleEvents     []WorkLifecycleEvent `gorm:"foreignKey:EmployeeID" json:"lifecycle_events,omitempty"`
	OnboardingProcess   *OnboardingProcess   `gorm:"foreignKey:EmployeeID" json:"onboarding_process,omitempty"`
	OffboardingProcess  *OffboardingProcess  `gorm:"foreignKey:EmployeeID" json:"offboarding_process,omitempty"`
	ComplianceRecords   []ComplianceRecord   `gorm:"foreignKey:EmployeeID" json:"compliance_records,omitempty"`
	AuditLogs           []AuditLog           `gorm:"foreignKey:PerformedBy" json:"audit_logs,omitempty"`
}

func (Employee) TableName() string {
	return "employees"
}
