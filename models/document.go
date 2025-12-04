package models

import (
	"time"

	"gorm.io/gorm"
)

type DocumentType string

const (
	DocumentTypeID           DocumentType = "id"
	DocumentTypeContract     DocumentType = "contract"
	DocumentTypeResume       DocumentType = "resume"
	DocumentTypeCertificate  DocumentType = "certificate"
	DocumentTypeLicense      DocumentType = "license"
	DocumentTypePerformance  DocumentType = "performance"
	DocumentTypeDisciplinary DocumentType = "disciplinary"
	DocumentTypeCompliance   DocumentType = "compliance"
	DocumentTypeOther        DocumentType = "other"
)

type DocumentStatus string

const (
	DocumentStatusActive   DocumentStatus = "active"
	DocumentStatusExpired  DocumentStatus = "expired"
	DocumentStatusPending  DocumentStatus = "pending"
	DocumentStatusArchived DocumentStatus = "archived"
)

// Document represents a document associated with an employee
type Document struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	EmployeeID     uint           `gorm:"not null;index" json:"employee_id"`
	DocumentType   DocumentType   `gorm:"type:varchar(50);not null" json:"document_type"`
	Title          string         `gorm:"size:200;not null" json:"title"`
	Description    *string        `gorm:"type:text" json:"description,omitempty"`
	FileName       string         `gorm:"size:255;not null" json:"file_name"`
	FilePath       string         `gorm:"size:500;not null" json:"file_path"`
	FileSize       *int64         `json:"file_size,omitempty"`
	MimeType       *string        `gorm:"size:100" json:"mime_type,omitempty"`
	IssueDate      *time.Time     `gorm:"type:date" json:"issue_date,omitempty"`
	ExpiryDate     *time.Time     `gorm:"type:date" json:"expiry_date,omitempty"`
	Status         DocumentStatus `gorm:"type:varchar(50);default:'active'" json:"status"`
	IsConfidential bool           `gorm:"default:false" json:"is_confidential"`
	UploadedBy     *uint          `gorm:"index" json:"uploaded_by,omitempty"`
	VerifiedBy     *uint          `gorm:"index" json:"verified_by,omitempty"`
	VerifiedAt     *time.Time     `json:"verified_at,omitempty"`
	Tags           *string        `gorm:"size:200" json:"tags,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	Employee Employee  `gorm:"foreignKey:EmployeeID" json:"employee,omitempty"`
	Uploader *Employee `gorm:"foreignKey:UploadedBy" json:"uploader,omitempty"`
	Verifier *Employee `gorm:"foreignKey:VerifiedBy" json:"verifier,omitempty"`
}

func (Document) TableName() string {
	return "documents"
}
