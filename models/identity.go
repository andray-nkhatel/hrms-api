package models

import (
	"time"

	"gorm.io/gorm"
)

// IdentityInformation stores comprehensive identity information for employees
type IdentityInformation struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	EmployeeID        uint           `gorm:"not null;uniqueIndex" json:"employee_id"`
	DateOfBirth       *time.Time     `gorm:"type:date" json:"date_of_birth,omitempty"`
	Gender            *string        `gorm:"size:20" json:"gender,omitempty"`
	Nationality       *string        `gorm:"size:50" json:"nationality,omitempty"`
	MaritalStatus     *string        `gorm:"size:20" json:"marital_status,omitempty"`
	PhoneNumber       *string        `gorm:"size:20" json:"phone_number,omitempty"`
	MobileNumber      *string        `gorm:"size:20" json:"mobile_number,omitempty"`
	Address           *string        `gorm:"type:text" json:"address,omitempty"`
	City              *string        `gorm:"size:50" json:"city,omitempty"`
	State             *string        `gorm:"size:50" json:"state,omitempty"`
	PostalCode        *string        `gorm:"size:20" json:"postal_code,omitempty"`
	Country           *string        `gorm:"size:50" json:"country,omitempty"`
	EmergencyContact  *string        `gorm:"size:100" json:"emergency_contact,omitempty"`
	EmergencyPhone    *string        `gorm:"size:20" json:"emergency_phone,omitempty"`
	EmergencyRelation *string        `gorm:"size:50" json:"emergency_relation,omitempty"`
	BloodGroup        *string        `gorm:"size:10" json:"blood_group,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	Employee Employee `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
}

func (IdentityInformation) TableName() string {
	return "identity_information"
}
