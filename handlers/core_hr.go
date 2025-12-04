package handlers

import (
	"encoding/json"
	"hrms-api/database"
	"hrms-api/models"
	"hrms-api/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Helper function to get current user from context
func getCurrentUser(c *gin.Context) *models.Employee {
	userID, exists := c.Get("user_id")
	if !exists {
		return nil
	}

	var employee models.Employee
	if err := database.DB.First(&employee, userID).Error; err != nil {
		return nil
	}
	return &employee
}

// Helper function to create audit log entry
func createAuditLog(entityType models.AuditEntityType, entityID uint, action models.AuditAction, performedBy uint, c *gin.Context, oldValues, newValues interface{}) {
	var oldJSON, newJSON, changesJSON []byte

	if oldValues != nil {
		oldJSON, _ = json.Marshal(oldValues)
	}
	if newValues != nil {
		newJSON, _ = json.Marshal(newValues)
	}

	// Calculate changes if both old and new values exist
	if oldValues != nil && newValues != nil {
		// Simple diff calculation - in production, you might want a more sophisticated diff
		changesJSON, _ = json.Marshal(map[string]interface{}{
			"old": oldValues,
			"new": newValues,
		})
	}

	auditLog := models.AuditLog{
		EntityType:    entityType,
		EntityID:      entityID,
		Action:        action,
		PerformedBy:   performedBy,
		IPAddress:     getStringPtr(c.ClientIP()),
		UserAgent:     getStringPtr(c.GetHeader("User-Agent")),
		RequestMethod: getStringPtr(c.Request.Method),
		RequestPath:   getStringPtr(c.Request.URL.Path),
		OldValues:     getStringPtr(string(oldJSON)),
		NewValues:     getStringPtr(string(newJSON)),
		Changes:       getStringPtr(string(changesJSON)),
	}

	database.DB.Create(&auditLog)
}

func getStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ==================== Identity Information Handlers ====================

// GetIdentityInformation retrieves identity information for an employee
// @Summary Get employee identity information
// @Description Get identity information for an employee
// @Tags Core HR - Identity
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {object} models.IdentityInformation
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/employees/{id}/identity [get]
func GetIdentityInformation(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var identity models.IdentityInformation
	if err := database.DB.Where("employee_id = ?", employeeID).First(&identity).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Identity information not found"})
		return
	}

	c.JSON(http.StatusOK, identity)
}

// CreateOrUpdateIdentityInformation creates or updates identity information
// @Summary Create or update identity information
// @Description Create or update identity information for an employee
// @Tags Core HR - Identity
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param request body models.IdentityInformation true "Identity information"
// @Success 200 {object} models.IdentityInformation
// @Success 201 {object} models.IdentityInformation
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/employees/{id}/identity [post]
func CreateOrUpdateIdentityInformation(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req models.IdentityInformation
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.EmployeeID = uint(employeeID)

	var existing models.IdentityInformation
	err := database.DB.Where("employee_id = ?", employeeID).First(&existing).Error

	if err != nil {
		// Create new
		if err := database.DB.Create(&req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create identity information"})
			return
		}
		user := getCurrentUser(c)
		if user != nil {
			createAuditLog(models.AuditEntityIdentity, req.ID, models.AuditActionCreate, user.ID, c, nil, req)
		}
		c.JSON(http.StatusCreated, req)
	} else {
		// Update existing
		oldValues := existing
		req.ID = existing.ID
		if err := database.DB.Save(&req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update identity information"})
			return
		}
		user := getCurrentUser(c)
		if user != nil {
			createAuditLog(models.AuditEntityIdentity, req.ID, models.AuditActionUpdate, user.ID, c, oldValues, req)
		}
		c.JSON(http.StatusOK, req)
	}
}

// ==================== Employment Details Handlers ====================

// GetEmploymentDetails retrieves employment details for an employee
// @Summary Get employee employment details
// @Description Get employment details for an employee
// @Tags Core HR - Employment
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {object} models.EmploymentDetails
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/employees/{id}/employment [get]
func GetEmploymentDetails(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var employment models.EmploymentDetails
	if err := database.DB.Preload("Manager").Where("employee_id = ?", employeeID).First(&employment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employment details not found"})
		return
	}

	c.JSON(http.StatusOK, employment)
}

// CreateOrUpdateEmploymentDetails creates or updates employment details
// @Summary Create or update employment details
// @Description Create or update employment details for an employee
// @Tags Core HR - Employment
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param request body models.EmploymentDetails true "Employment details"
// @Success 200 {object} models.EmploymentDetails
// @Success 201 {object} models.EmploymentDetails
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/employees/{id}/employment [post]
func CreateOrUpdateEmploymentDetails(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req models.EmploymentDetails
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.EmployeeID = uint(employeeID)

	var existing models.EmploymentDetails
	err := database.DB.Where("employee_id = ?", employeeID).First(&existing).Error

	if err != nil {
		if err := database.DB.Create(&req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create employment details"})
			return
		}
		user := getCurrentUser(c)
		if user != nil {
			createAuditLog(models.AuditEntityEmployment, req.ID, models.AuditActionCreate, user.ID, c, nil, req)
		}
		c.JSON(http.StatusCreated, req)
	} else {
		oldValues := existing
		req.ID = existing.ID
		if err := database.DB.Save(&req).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update employment details"})
			return
		}
		user := getCurrentUser(c)
		if user != nil {
			createAuditLog(models.AuditEntityEmployment, req.ID, models.AuditActionUpdate, user.ID, c, oldValues, req)
		}
		c.JSON(http.StatusOK, req)
	}
}

// GetEmploymentHistory retrieves employment history for an employee
// @Summary Get employee employment history
// @Description Get employment history for an employee
// @Tags Core HR - Employment
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {array} models.EmploymentHistory
// @Failure 401 {object} ErrorResponse
// @Router /api/employees/{id}/employment/history [get]
func GetEmploymentHistory(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var history []models.EmploymentHistory
	database.DB.Preload("Changer").Where("employee_id = ?", employeeID).Order("change_date DESC").Find(&history)

	c.JSON(http.StatusOK, history)
}

// ==================== Position Handlers ====================

// GetPositions retrieves all positions
// @Summary Get all positions
// @Description Get list of all active positions
// @Tags Core HR - Positions
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Position
// @Failure 401 {object} ErrorResponse
// @Router /api/positions [get]
func GetPositions(c *gin.Context) {
	var positions []models.Position
	database.DB.Preload("ReportsTo").Where("is_active = ?", true).Find(&positions)
	c.JSON(http.StatusOK, positions)
}

// GetPosition retrieves a specific position
// @Summary Get position by ID
// @Description Get a specific position by ID
// @Tags Core HR - Positions
// @Produce json
// @Security BearerAuth
// @Param id path int true "Position ID"
// @Success 200 {object} models.Position
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/positions/{id} [get]
func GetPosition(c *gin.Context) {
	positionID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var position models.Position
	if err := database.DB.Preload("ReportsTo").First(&position, positionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Position not found"})
		return
	}

	c.JSON(http.StatusOK, position)
}

// CreatePosition creates a new position
// @Summary Create position
// @Description Create a new position (Manager/Admin only)
// @Tags Core HR - Positions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.Position true "Position data"
// @Success 201 {object} models.Position
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/positions [post]
func CreatePosition(c *gin.Context) {
	var req models.Position
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create position"})
		return
	}

	user := getCurrentUser(c)
	if user != nil {
		createAuditLog(models.AuditEntityPosition, req.ID, models.AuditActionCreate, user.ID, c, nil, req)
	}

	c.JSON(http.StatusCreated, req)
}

// UpdatePosition updates a position
// @Summary Update position
// @Description Update an existing position (Manager/Admin only)
// @Tags Core HR - Positions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Position ID"
// @Param request body models.Position true "Position data"
// @Success 200 {object} models.Position
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/positions/{id} [put]
func UpdatePosition(c *gin.Context) {
	positionID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var position models.Position
	if err := database.DB.First(&position, positionID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Position not found"})
		return
	}

	oldValues := position

	if err := c.ShouldBindJSON(&position); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	position.ID = uint(positionID)
	if err := database.DB.Save(&position).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update position"})
		return
	}

	user := getCurrentUser(c)
	if user != nil {
		createAuditLog(models.AuditEntityPosition, position.ID, models.AuditActionUpdate, user.ID, c, oldValues, position)
	}

	c.JSON(http.StatusOK, position)
}

// AssignPosition assigns a position to an employee
// @Summary Assign position to employee
// @Description Assign a position to an employee (Manager/Admin only)
// @Tags Core HR - Positions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param request body models.PositionAssignment true "Position assignment"
// @Success 201 {object} models.PositionAssignment
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/employees/{id}/positions [post]
func AssignPosition(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req models.PositionAssignment
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.EmployeeID = uint(employeeID)
	user := getCurrentUser(c)
	if user != nil {
		req.AssignedBy = &user.ID
	}

	if err := database.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign position"})
		return
	}

	if user != nil {
		createAuditLog(models.AuditEntityPosition, req.ID, models.AuditActionCreate, user.ID, c, nil, req)
	}

	c.JSON(http.StatusCreated, req)
}

// ==================== Document Handlers ====================

// GetDocuments retrieves documents for an employee
// @Summary Get employee documents
// @Description Get all documents for an employee
// @Tags Core HR - Documents
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {array} models.Document
// @Failure 401 {object} ErrorResponse
// @Router /api/employees/{id}/documents [get]
func GetDocuments(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var documents []models.Document
	database.DB.Preload("Uploader").Preload("Verifier").Where("employee_id = ?", employeeID).Find(&documents)

	c.JSON(http.StatusOK, documents)
}

// CreateDocumentRequest represents the form data for document upload
type CreateDocumentRequest struct {
	DocumentType   models.DocumentType `form:"document_type" binding:"required"`
	Title          string              `form:"title" binding:"required"`
	Description    *string             `form:"description"`
	IssueDate      *string             `form:"issue_date" time_format:"2006-01-02"`
	ExpiryDate     *string             `form:"expiry_date" time_format:"2006-01-02"`
	IsConfidential bool                `form:"is_confidential"`
	Tags           *string             `form:"tags"`
}

// CreateDocument handles file upload and creates a new document record
// @Summary Upload and create document
// @Description Upload a file and create a new document record for an employee. Accepts multipart/form-data with file upload.
// @Tags Core HR - Documents
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param file formData file true "Document file to upload"
// @Param document_type formData string true "Document type (id, contract, resume, certificate, license, performance, disciplinary, compliance, other)"
// @Param title formData string true "Document title"
// @Param description formData string false "Document description"
// @Param issue_date formData string false "Issue date (YYYY-MM-DD)"
// @Param expiry_date formData string false "Expiry date (YYYY-MM-DD)"
// @Param is_confidential formData boolean false "Is document confidential"
// @Param tags formData string false "Document tags (comma-separated)"
// @Success 201 {object} models.Document
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 413 {object} ErrorResponse "File too large"
// @Failure 415 {object} ErrorResponse "Unsupported file type"
// @Failure 500 {object} ErrorResponse
// @Router /api/employees/{id}/documents [post]
func CreateDocument(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	// Parse form data
	var formData CreateDocumentRequest
	if err := c.ShouldBind(&formData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data: " + err.Error()})
		return
	}

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required: " + err.Error()})
		return
	}

	// Validate file extension
	if err := utils.ValidateFileExtension(file.Filename); err != nil {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": err.Error()})
		return
	}

	// Validate file size
	if err := utils.ValidateFileSize(file.Size); err != nil {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
		return
	}

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file"})
		return
	}
	defer src.Close()

	// Detect MIME type
	mimeType := utils.GetFileMimeType(file.Filename)
	if err := utils.ValidateMimeType(mimeType); err != nil {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": err.Error()})
		return
	}

	// Generate secure filename
	secureFilename, err := utils.GenerateSecureFileName(file.Filename, uint(employeeID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate filename"})
		return
	}

	// Save file to storage
	relativePath, fileSize, err := utils.SaveFile(src, secureFilename, uint(employeeID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file: " + err.Error()})
		return
	}

	// Parse dates if provided
	var issueDate, expiryDate *time.Time
	if formData.IssueDate != nil && *formData.IssueDate != "" {
		parsed, err := time.Parse("2006-01-02", *formData.IssueDate)
		if err == nil {
			issueDate = &parsed
		}
	}
	if formData.ExpiryDate != nil && *formData.ExpiryDate != "" {
		parsed, err := time.Parse("2006-01-02", *formData.ExpiryDate)
		if err == nil {
			expiryDate = &parsed
		}
	}

	// Create document record
	user := getCurrentUser(c)
	document := models.Document{
		EmployeeID:     uint(employeeID),
		DocumentType:   formData.DocumentType,
		Title:          formData.Title,
		Description:    formData.Description,
		FileName:       file.Filename, // Original filename
		FilePath:       relativePath,  // Stored file path
		FileSize:       &fileSize,
		MimeType:       &mimeType,
		IssueDate:      issueDate,
		ExpiryDate:     expiryDate,
		Status:         models.DocumentStatusActive,
		IsConfidential: formData.IsConfidential,
		Tags:           formData.Tags,
	}

	if user != nil {
		document.UploadedBy = &user.ID
	}

	if err := database.DB.Create(&document).Error; err != nil {
		// Clean up file if database save fails
		utils.DeleteFile(relativePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create document record"})
		return
	}

	// Create audit log
	if user != nil {
		createAuditLog(models.AuditEntityDocument, document.ID, models.AuditActionCreate, user.ID, c, nil, document)
	}

	// Load associations
	database.DB.Preload("Uploader").Preload("Verifier").First(&document, document.ID)

	c.JSON(http.StatusCreated, document)
}

// DownloadDocument downloads a document file
// @Summary Download document file
// @Description Download the actual file for a document
// @Tags Core HR - Documents
// @Produce application/octet-stream
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param doc_id path int true "Document ID"
// @Success 200 {file} file
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/employees/{id}/documents/{doc_id}/download [get]
func DownloadDocument(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	documentID, _ := strconv.ParseUint(c.Param("doc_id"), 10, 32)

	// Get document from database
	var document models.Document
	if err := database.DB.Where("id = ? AND employee_id = ?", documentID, employeeID).First(&document).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// Check if file exists
	if !utils.FileExists(document.FilePath) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document file not found on server"})
		return
	}

	// Get full file path
	fullPath := utils.GetFullFilePath(document.FilePath)

	// Set headers for file download
	c.Header("Content-Disposition", `attachment; filename="`+document.FileName+`"`)
	if document.MimeType != nil {
		c.Header("Content-Type", *document.MimeType)
	}

	// Send file
	c.File(fullPath)
}

// DeleteDocument deletes a document and its file
// @Summary Delete document
// @Description Delete a document record and its associated file
// @Tags Core HR - Documents
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param doc_id path int true "Document ID"
// @Success 200 {object} MessageResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/employees/{id}/documents/{doc_id} [delete]
func DeleteDocument(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	documentID, _ := strconv.ParseUint(c.Param("doc_id"), 10, 32)

	// Get document from database
	var document models.Document
	if err := database.DB.Where("id = ? AND employee_id = ?", documentID, employeeID).First(&document).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Document not found"})
		return
	}

	// Delete file from storage
	if utils.FileExists(document.FilePath) {
		if err := utils.DeleteFile(document.FilePath); err != nil {
			// Log error but continue with database deletion
			// In production, you might want to handle this differently
		}
	}

	// Delete from database
	oldValues := document
	if err := database.DB.Delete(&document).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete document"})
		return
	}

	// Create audit log
	user := getCurrentUser(c)
	if user != nil {
		createAuditLog(models.AuditEntityDocument, document.ID, models.AuditActionDelete, user.ID, c, oldValues, nil)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Document deleted successfully"})
}

// ==================== Work Lifecycle Handlers ====================

// GetLifecycleEvents retrieves lifecycle events for an employee
// @Summary Get employee lifecycle events
// @Description Get all lifecycle events for an employee
// @Tags Core HR - Lifecycle
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {array} models.WorkLifecycleEvent
// @Failure 401 {object} ErrorResponse
// @Router /api/employees/{id}/lifecycle [get]
func GetLifecycleEvents(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var events []models.WorkLifecycleEvent
	database.DB.Preload("Initiator").Preload("Approver").Where("employee_id = ?", employeeID).Order("event_date DESC").Find(&events)

	c.JSON(http.StatusOK, events)
}

// CreateLifecycleEvent creates a new lifecycle event
// @Summary Create lifecycle event
// @Description Create a new lifecycle event for an employee (Manager/Admin only)
// @Tags Core HR - Lifecycle
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param request body models.WorkLifecycleEvent true "Lifecycle event data"
// @Success 201 {object} models.WorkLifecycleEvent
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/employees/{id}/lifecycle [post]
func CreateLifecycleEvent(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req models.WorkLifecycleEvent
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.EmployeeID = uint(employeeID)
	user := getCurrentUser(c)
	if user != nil {
		req.InitiatedBy = &user.ID
	}

	if err := database.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create lifecycle event"})
		return
	}

	if user != nil {
		createAuditLog(models.AuditEntityLifecycle, req.ID, models.AuditActionCreate, user.ID, c, nil, req)
	}

	c.JSON(http.StatusCreated, req)
}

// ==================== Onboarding Handlers ====================

// GetOnboardingProcess retrieves onboarding process for an employee
// @Summary Get employee onboarding process
// @Description Get onboarding process for an employee
// @Tags Core HR - Onboarding
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {object} models.OnboardingProcess
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/employees/{id}/onboarding [get]
func GetOnboardingProcess(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var process models.OnboardingProcess
	if err := database.DB.Preload("Tasks").Preload("Assignee").Preload("Initiator").Where("employee_id = ?", employeeID).First(&process).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Onboarding process not found"})
		return
	}

	c.JSON(http.StatusOK, process)
}

// CreateOnboardingProcess creates a new onboarding process
// @Summary Create onboarding process
// @Description Create a new onboarding process for an employee (Manager/Admin only)
// @Tags Core HR - Onboarding
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param request body models.OnboardingProcess true "Onboarding process data"
// @Success 201 {object} models.OnboardingProcess
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/employees/{id}/onboarding [post]
func CreateOnboardingProcess(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req models.OnboardingProcess
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.EmployeeID = uint(employeeID)
	user := getCurrentUser(c)
	if user != nil {
		req.InitiatedBy = &user.ID
	}

	if err := database.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create onboarding process"})
		return
	}

	if user != nil {
		createAuditLog(models.AuditEntityOnboarding, req.ID, models.AuditActionCreate, user.ID, c, nil, req)
	}

	c.JSON(http.StatusCreated, req)
}

// ==================== Offboarding Handlers ====================

// GetOffboardingProcess retrieves offboarding process for an employee
// @Summary Get employee offboarding process
// @Description Get offboarding process for an employee
// @Tags Core HR - Offboarding
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {object} models.OffboardingProcess
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/employees/{id}/offboarding [get]
func GetOffboardingProcess(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var process models.OffboardingProcess
	if err := database.DB.Preload("Tasks").Preload("Assignee").Preload("Initiator").Where("employee_id = ?", employeeID).First(&process).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Offboarding process not found"})
		return
	}

	c.JSON(http.StatusOK, process)
}

// CreateOffboardingProcess creates a new offboarding process
// @Summary Create offboarding process
// @Description Create a new offboarding process for an employee (Manager/Admin only)
// @Tags Core HR - Offboarding
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param request body models.OffboardingProcess true "Offboarding process data"
// @Success 201 {object} models.OffboardingProcess
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/employees/{id}/offboarding [post]
func CreateOffboardingProcess(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req models.OffboardingProcess
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.EmployeeID = uint(employeeID)
	user := getCurrentUser(c)
	if user != nil {
		req.InitiatedBy = &user.ID
	}

	if err := database.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create offboarding process"})
		return
	}

	if user != nil {
		createAuditLog(models.AuditEntityOffboarding, req.ID, models.AuditActionCreate, user.ID, c, nil, req)
	}

	c.JSON(http.StatusCreated, req)
}

// ==================== Compliance Handlers ====================

// GetComplianceRequirements retrieves all compliance requirements
// @Summary Get all compliance requirements
// @Description Get list of all active compliance requirements
// @Tags Core HR - Compliance
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.ComplianceRequirement
// @Failure 401 {object} ErrorResponse
// @Router /api/compliance/requirements [get]
func GetComplianceRequirements(c *gin.Context) {
	var requirements []models.ComplianceRequirement
	database.DB.Where("is_active = ?", true).Find(&requirements)
	c.JSON(http.StatusOK, requirements)
}

// CreateComplianceRequirement creates a new compliance requirement
// @Summary Create compliance requirement
// @Description Create a new compliance requirement (Manager/Admin only)
// @Tags Core HR - Compliance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ComplianceRequirement true "Compliance requirement data"
// @Success 201 {object} models.ComplianceRequirement
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/compliance/requirements [post]
func CreateComplianceRequirement(c *gin.Context) {
	var req models.ComplianceRequirement
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create compliance requirement"})
		return
	}

	user := getCurrentUser(c)
	if user != nil {
		createAuditLog(models.AuditEntityCompliance, req.ID, models.AuditActionCreate, user.ID, c, nil, req)
	}

	c.JSON(http.StatusCreated, req)
}

// GetComplianceRecords retrieves compliance records for an employee
// @Summary Get employee compliance records
// @Description Get all compliance records for an employee
// @Tags Core HR - Compliance
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {array} models.ComplianceRecord
// @Failure 401 {object} ErrorResponse
// @Router /api/employees/{id}/compliance [get]
func GetComplianceRecords(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var records []models.ComplianceRecord
	database.DB.Preload("Requirement").Preload("Verifier").Preload("Document").Where("employee_id = ?", employeeID).Find(&records)

	c.JSON(http.StatusOK, records)
}

// CreateComplianceRecord creates a new compliance record
// @Summary Create compliance record
// @Description Create a new compliance record for an employee (Manager/Admin only)
// @Tags Core HR - Compliance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param request body models.ComplianceRecord true "Compliance record data"
// @Success 201 {object} models.ComplianceRecord
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/employees/{id}/compliance [post]
func CreateComplianceRecord(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req models.ComplianceRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.EmployeeID = uint(employeeID)

	if err := database.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create compliance record"})
		return
	}

	user := getCurrentUser(c)
	if user != nil {
		createAuditLog(models.AuditEntityCompliance, req.ID, models.AuditActionCreate, user.ID, c, nil, req)
	}

	c.JSON(http.StatusCreated, req)
}

// ==================== Audit Log Handlers ====================

// GetAuditLogs retrieves audit logs with optional filtering
// @Summary Get audit logs
// @Description Get audit logs with optional filtering by entity type, entity ID, or performed by
// @Tags Core HR - Audit
// @Produce json
// @Security BearerAuth
// @Param entity_type query string false "Entity type filter"
// @Param entity_id query int false "Entity ID filter"
// @Param performed_by query int false "Performed by user ID filter"
// @Success 200 {array} models.AuditLog
// @Failure 401 {object} ErrorResponse
// @Router /api/audit-logs [get]
func GetAuditLogs(c *gin.Context) {
	var logs []models.AuditLog
	query := database.DB.Preload("Performer")

	if entityType := c.Query("entity_type"); entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}

	if entityID := c.Query("entity_id"); entityID != "" {
		query = query.Where("entity_id = ?", entityID)
	}

	if performedBy := c.Query("performed_by"); performedBy != "" {
		query = query.Where("performed_by = ?", performedBy)
	}

	query.Order("created_at DESC").Limit(100).Find(&logs)

	c.JSON(http.StatusOK, logs)
}

// GetEmployeeAuditLogs retrieves audit logs for a specific employee
// @Summary Get employee audit logs
// @Description Get audit logs related to a specific employee
// @Tags Core HR - Audit
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {array} models.AuditLog
// @Failure 401 {object} ErrorResponse
// @Router /api/employees/{id}/audit-logs [get]
func GetEmployeeAuditLogs(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var logs []models.AuditLog
	database.DB.Preload("Performer").
		Where("(entity_type = ? AND entity_id = ?) OR performed_by = ?",
			models.AuditEntityEmployee, employeeID, employeeID).
		Order("created_at DESC").
		Limit(100).
		Find(&logs)

	c.JSON(http.StatusOK, logs)
}
