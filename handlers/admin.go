package handlers

import (
	"encoding/csv"
	"fmt"
	"hrms-api/database"
	"hrms-api/models"
	"hrms-api/utils"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateLeaveTypeRequest represents data for creating a leave type
type CreateLeaveTypeRequest struct {
	Name    string `json:"name" binding:"required" example:"Sabbatical"`
	MaxDays int    `json:"max_days" binding:"required,min=1" example:"30"`
}

// CreateEmployeeRequest represents data for creating an employee/manager (uses NRC)
type CreateEmployeeRequest struct {
	NRC        string      `json:"nrc" binding:"required" example:"555666/77/8"`
	Firstname  string      `json:"firstname" binding:"required" example:"Jane"`
	Lastname   string      `json:"lastname" binding:"required" example:"Smith"`
	Email      string      `json:"email" binding:"omitempty,email" example:"jane@example.com"` // Optional now
	Password   string      `json:"password" binding:"required,min=6" example:"password123"`
	Department string      `json:"department" example:"HR"`
	Role       models.Role `json:"role" binding:"required" example:"manager"`
	HireDate   *string     `json:"hire_date,omitempty" example:"2025-01-15"` // Optional: YYYY-MM-DD format, defaults to today if not provided
}

// CreateAdminRequest represents data for creating an admin (uses username)
type CreateAdminRequest struct {
	Username   string `json:"username" binding:"required" example:"admin2"`
	Firstname  string `json:"firstname" binding:"required" example:"Admin"`
	Lastname   string `json:"lastname" binding:"required" example:"User"`
	Email      string `json:"email" binding:"omitempty,email" example:"admin2@example.com"` // Optional now
	Password   string `json:"password" binding:"required,min=6" example:"password123"`
	Department string `json:"department" example:"Administration"`
}

// UpdateEmployeeRequest represents data for updating an employee
type UpdateEmployeeRequest struct {
	Firstname  string      `json:"firstname" example:"Jane"`
	Lastname   string      `json:"lastname" example:"Doe"`
	Email      string      `json:"email" example:"jane.doe@example.com"`
	Department string      `json:"department" example:"Finance"`
	Role       models.Role `json:"role" example:"admin"`
}

// MessageResponse represents a simple message response
type MessageResponse struct {
	Message string `json:"message" example:"Operation completed successfully"`
}

// GetLeaveTypes returns all leave types
// @Summary Get all leave types
// @Description Get list of all available leave types (Admin only)
// @Tags Admin - Leave Types
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.LeaveType
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/leave-types [get]
func GetLeaveTypes(c *gin.Context) {
	var leaveTypes []models.LeaveType
	if err := database.DB.Find(&leaveTypes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leave types"})
		return
	}

	c.JSON(http.StatusOK, leaveTypes)
}

// CreateLeaveType creates a new leave type
// @Summary Create leave type
// @Description Create a new leave type (Admin only)
// @Tags Admin - Leave Types
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateLeaveTypeRequest true "Leave type data"
// @Success 201 {object} models.LeaveType
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/leave-types [post]
func CreateLeaveType(c *gin.Context) {
	var req CreateLeaveTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	leaveType := models.LeaveType{
		Name:    req.Name,
		MaxDays: req.MaxDays,
	}

	if err := database.DB.Create(&leaveType).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create leave type"})
		return
	}

	c.JSON(http.StatusCreated, leaveType)
}

// UpdateLeaveType updates an existing leave type
// @Summary Update leave type
// @Description Update an existing leave type (Admin only)
// @Tags Admin - Leave Types
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave Type ID"
// @Param request body CreateLeaveTypeRequest true "Leave type data"
// @Success 200 {object} models.LeaveType
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/leave-types/{id} [put]
func UpdateLeaveType(c *gin.Context) {
	leaveTypeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid leave type ID"})
		return
	}

	var leaveType models.LeaveType
	if err := database.DB.First(&leaveType, uint(leaveTypeID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Leave type not found"})
		return
	}

	var req CreateLeaveTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	leaveType.Name = req.Name
	leaveType.MaxDays = req.MaxDays

	if err := database.DB.Save(&leaveType).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update leave type"})
		return
	}

	c.JSON(http.StatusOK, leaveType)
}

// DeleteLeaveType deletes a leave type
// @Summary Delete leave type
// @Description Delete a leave type (Admin only)
// @Tags Admin - Leave Types
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave Type ID"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/leave-types/{id} [delete]
func DeleteLeaveType(c *gin.Context) {
	leaveTypeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid leave type ID"})
		return
	}

	if err := database.DB.Delete(&models.LeaveType{}, uint(leaveTypeID)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete leave type"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Leave type deleted successfully"})
}

// CreateEmployee creates a new employee/manager account (not admin)
// @Summary Create employee/manager
// @Description Create a new employee or manager account with NRC (Admin only). Use /api/admins for admin accounts.
// @Tags Admin - Employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateEmployeeRequest true "Employee data"
// @Success 201 {object} models.Employee
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse "NRC or email already exists"
// @Router /api/employees [post]
func CreateEmployee(c *gin.Context) {
	var req CreateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate role - only employee or manager allowed here
	if req.Role != models.RoleEmployee && req.Role != models.RoleManager {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Use /api/admins endpoint to create admin accounts"})
		return
	}

	// Check if NRC or email already exists (including soft-deleted records)
	var existingEmployee models.Employee
	emailCheck := req.Email
	if emailCheck == "" {
		emailCheck = "NO_EMAIL_" + req.NRC // Use a placeholder if email is empty
	}
	if err := database.DB.Unscoped().Where("nrc = ? OR (email IS NOT NULL AND email = ?)", req.NRC, emailCheck).First(&existingEmployee).Error; err == nil {
		// If found and it's soft-deleted, permanently delete it to allow reuse
		if existingEmployee.DeletedAt.Valid {
			// Permanently delete the soft-deleted employee to allow NRC/email reuse
			if err := database.DB.Unscoped().Delete(&existingEmployee).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clean up deleted employee record"})
				return
			}
		} else {
			// Active employee with this NRC/email exists
			c.JSON(http.StatusConflict, gin.H{"error": "NRC or email already exists"})
			return
		}
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	nrc := req.NRC
	var email *string
	if req.Email != "" {
		email = &req.Email
	}
	employee := models.Employee{
		NRC:          &nrc,
		Firstname:    req.Firstname,
		Lastname:     req.Lastname,
		Email:        email,
		PasswordHash: hashedPassword,
		Department:   req.Department,
		Role:         req.Role,
	}

	if err := database.DB.Create(&employee).Error; err != nil {
		// Check for duplicate key constraint violation
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "NRC or email already exists in the database"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create employee: " + err.Error()})
		return
	}

	// Automatically create EmploymentDetails with hire date
	var hireDate *time.Time
	if req.HireDate != nil && *req.HireDate != "" {
		// Parse provided hire date
		parsedDate, err := time.Parse("2006-01-02", *req.HireDate)
		if err != nil {
			// If parsing fails, use today's date instead of failing
			today := time.Now()
			hireDate = &today
		} else {
			hireDate = &parsedDate
		}
	} else {
		// Default to today if not provided
		today := time.Now()
		hireDate = &today
	}

	// Create EmploymentDetails
	employmentDetails := models.EmploymentDetails{
		EmployeeID:       employee.ID,
		EmploymentType:   models.EmploymentTypeFullTime,
		EmploymentStatus: models.EmploymentStatusActive,
		HireDate:         hireDate,
		StartDate:        hireDate, // Set start date same as hire date
	}

	if err := database.DB.Create(&employmentDetails).Error; err != nil {
		// Log error but don't fail employee creation
		// EmploymentDetails can be added later via the employment endpoint
		c.JSON(http.StatusCreated, gin.H{
			"employee": employee,
			"warning":  "Employee created but employment details could not be created. Please add them manually.",
		})
		return
	}

	employee.PasswordHash = ""
	c.JSON(http.StatusCreated, employee)
}

// CreateAdmin creates a new admin account with username
// @Summary Create admin
// @Description Create a new admin account with username (Admin only)
// @Tags Admin - Employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateAdminRequest true "Admin data"
// @Success 201 {object} models.Employee
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse "Username or email already exists"
// @Router /api/admins [post]
func CreateAdmin(c *gin.Context) {
	var req CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if username or email already exists (including soft-deleted records)
	var existingEmployee models.Employee
	emailCheck := req.Email
	if emailCheck == "" {
		emailCheck = "NO_EMAIL_" + req.Username // Use a placeholder if email is empty
	}
	if err := database.DB.Unscoped().Where("username = ? OR (email IS NOT NULL AND email = ?)", req.Username, emailCheck).First(&existingEmployee).Error; err == nil {
		// If found and it's soft-deleted, permanently delete it to allow reuse
		if existingEmployee.DeletedAt.Valid {
			// Permanently delete the soft-deleted employee to allow username/email reuse
			if err := database.DB.Unscoped().Delete(&existingEmployee).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clean up deleted employee record"})
				return
			}
		} else {
			// Active employee with this username/email exists
			c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists"})
			return
		}
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	username := req.Username
	var email *string
	if req.Email != "" {
		email = &req.Email
	}
	employee := models.Employee{
		Username:     &username,
		Firstname:    req.Firstname,
		Lastname:     req.Lastname,
		Email:        email,
		PasswordHash: hashedPassword,
		Department:   req.Department,
		Role:         models.RoleAdmin,
	}

	if err := database.DB.Create(&employee).Error; err != nil {
		// Check for duplicate key constraint violation
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists in the database"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create admin: " + err.Error()})
		return
	}

	employee.PasswordHash = ""
	c.JSON(http.StatusCreated, employee)
}

// GetEmployees returns all employees
// @Summary Get all employees
// @Description Get list of all employees (Admin only). Supports search query parameter for filtering by name.
// @Tags Admin - Employees
// @Produce json
// @Security BearerAuth
// @Param search query string false "Search term to filter employees by name (firstname, lastname, or full name)"
// @Success 200 {array} models.Employee
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/employees [get]
func GetEmployees(c *gin.Context) {
	var employees []models.Employee
	query := database.DB.Where("role != ?", models.RoleAdmin) // Exclude admin users
	
	// Support search parameter for filtering by name
	search := c.Query("search")
	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"LOWER(firstname) LIKE ? OR LOWER(lastname) LIKE ? OR LOWER(CONCAT(firstname, ' ', lastname)) LIKE ?",
			searchPattern, searchPattern, searchPattern,
		)
	}
	
	if err := query.Preload("Employment").
		Select("id", "nrc", "username", "firstname", "lastname", "email", "department", "role", "created_at", "updated_at").
		Find(&employees).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch employees"})
		return
	}

	c.JSON(http.StatusOK, employees)
}

// GetEmployee returns a specific employee by ID
// @Summary Get employee by ID
// @Description Get a specific employee by ID (Admin only)
// @Tags Admin - Employees
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {object} models.Employee
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/employees/{id} [get]
func GetEmployee(c *gin.Context) {
	employeeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee ID"})
		return
	}

	var employee models.Employee
	if err := database.DB.Select("id", "nrc", "username", "firstname", "lastname", "email", "department", "role", "created_at", "updated_at").
		First(&employee, uint(employeeID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}

	c.JSON(http.StatusOK, employee)
}

// UpdateEmployee updates an employee
// @Summary Update employee
// @Description Update an employee's information (Admin only)
// @Tags Admin - Employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param request body UpdateEmployeeRequest true "Employee data to update"
// @Success 200 {object} models.Employee
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/employees/{id} [put]
func UpdateEmployee(c *gin.Context) {
	employeeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee ID"})
		return
	}

	var employee models.Employee
	if err := database.DB.First(&employee, uint(employeeID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}

	var req struct {
		Firstname                   string      `json:"firstname"`
		Lastname                    string      `json:"lastname"`
		Email                       *string     `json:"email" binding:"omitempty,email"`
		NRC                         *string     `json:"nrc"`
		Department                  string      `json:"department"`
		Role                        models.Role `json:"role"`
		Phone                       *string     `json:"phone"`
		Mobile                      *string     `json:"mobile"`
		Address                     *string     `json:"address"`
		City                        *string     `json:"city"`
		PostalCode                  *string     `json:"postal_code"`
		DateOfBirth                 *string     `json:"date_of_birth"`
		Gender                      *string     `json:"gender"`
		JobTitle                    *string     `json:"job_title"`
		Position                    *string     `json:"position"` // Maps to JobTitle for backward compatibility
		EmploymentStatus            *string     `json:"employment_status"`
		EmergencyContactName        *string     `json:"emergency_contact_name"`
		EmergencyContactPhone       *string     `json:"emergency_contact_phone"`
		EmergencyContactRelationship *string    `json:"emergency_contact_relationship"`
		BankName                    *string     `json:"bank_name"`
		BankAccountNumber           *string     `json:"bank_account_number"`
		TaxID                       *string     `json:"tax_id"`
		Notes                       *string     `json:"notes"`
		HireDate                    *string     `json:"hire_date"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Firstname != "" {
		employee.Firstname = req.Firstname
	}
	if req.Lastname != "" {
		employee.Lastname = req.Lastname
	}
	if req.Email != nil {
		employee.Email = req.Email
	}
	if req.Department != "" {
		employee.Department = req.Department
	}
	if req.Role != "" {
		validRoles := []models.Role{models.RoleEmployee, models.RoleManager, models.RoleAdmin}
		valid := false
		for _, r := range validRoles {
			if req.Role == r {
				valid = true
				break
			}
		}
		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
			return
		}
		employee.Role = req.Role
	}

	if err := database.DB.Save(&employee).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update employee"})
		return
	}

	employee.PasswordHash = ""
	c.JSON(http.StatusOK, employee)
}

// ChangePassword allows an employee to change their own password
// @Summary Change password
// @Description Change password for the authenticated user (requires current password)
// @Tags Admin - Employees
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param request body ChangePasswordRequest true "Password change data"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/employees/{id}/password [put]
func ChangePassword(c *gin.Context) {
	employeeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee ID"})
		return
	}

	// Get current user ID from token
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Users can only change their own password
	if uint(employeeID) != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only change your own password"})
		return
	}

	var employee models.Employee
	if err := database.DB.First(&employee, uint(employeeID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify current password
	if !utils.CheckPasswordHash(req.CurrentPassword, employee.PasswordHash) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Update password
	employee.PasswordHash = hashedPassword
	if err := database.DB.Save(&employee).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required" example:"oldPassword123"`
	NewPassword     string `json:"new_password" binding:"required,min=6" example:"newPassword123"`
}

// DeleteEmployee deletes an employee
// @Summary Delete employee
// @Description Delete an employee (Admin only)
// @Tags Admin - Employees
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/employees/{id} [delete]
func DeleteEmployee(c *gin.Context) {
	employeeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee ID"})
		return
	}

	if err := database.DB.Delete(&models.Employee{}, uint(employeeID)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete employee"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Employee deleted successfully"})
}

// DownloadEmployeeTemplate returns a CSV template for bulk employee upload
// @Summary Download employee CSV template
// @Description Download a CSV template for bulk employee upload (Admin only)
// @Tags Admin - Employees
// @Produce text/csv
// @Security BearerAuth
// @Success 200 {file} file "CSV template file"
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/employees/template [get]
func DownloadEmployeeTemplate(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=employee_template.csv")

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Header row
	writer.Write([]string{"nrc", "firstname", "lastname", "email", "password", "department", "role"})
	// Example rows
	writer.Write([]string{"123456/78/9", "John", "Doe", "john.doe@example.com", "password123", "IT", "employee"})
	writer.Write([]string{"987654/32/1", "Jane", "Smith", "jane.smith@example.com", "password123", "HR", "manager"})
}

// BulkUploadResponse represents the response for bulk upload
type BulkUploadResponse struct {
	Total   int      `json:"total" example:"10"`
	Success int      `json:"success" example:"8"`
	Failed  int      `json:"failed" example:"2"`
	Errors  []string `json:"errors,omitempty" example:"Row 3: NRC already exists"`
}

// BulkUploadEmployees uploads employees from a CSV file
// @Summary Bulk upload employees
// @Description Upload multiple employees from a CSV file (Admin only)
// @Tags Admin - Employees
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "CSV file with employee data"
// @Success 200 {object} BulkUploadResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/employees/bulk [post]
func BulkUploadEmployees(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header row
	header, err := reader.Read()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CSV file"})
		return
	}

	// Validate header
	expectedHeader := []string{"nrc", "firstname", "lastname", "email", "password", "department", "role"}
	if len(header) < len(expectedHeader) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CSV format. Download the template for correct format."})
		return
	}

	var total, success, failed int
	var errors []string

	rowNum := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("Row %d: Failed to parse", rowNum+1))
			failed++
			rowNum++
			continue
		}

		total++
		rowNum++

		if len(record) < 7 {
			errors = append(errors, fmt.Sprintf("Row %d: Incomplete data", rowNum))
			failed++
			continue
		}

		nrc := strings.TrimSpace(record[0])
		firstname := strings.TrimSpace(record[1])
		lastname := strings.TrimSpace(record[2])
		email := strings.TrimSpace(record[3])
		password := strings.TrimSpace(record[4])
		department := strings.TrimSpace(record[5])
		role := strings.ToLower(strings.TrimSpace(record[6]))

		// Validate required fields
		if nrc == "" || firstname == "" || lastname == "" || email == "" || password == "" {
			errors = append(errors, fmt.Sprintf("Row %d: Missing required fields", rowNum))
			failed++
			continue
		}

		// Validate role
		if role != "employee" && role != "manager" {
			errors = append(errors, fmt.Sprintf("Row %d: Invalid role '%s' (must be employee or manager)", rowNum, role))
			failed++
			continue
		}

		// Check if NRC or email already exists
		var existing models.Employee
		emailCheck := email
		if emailCheck == "" {
			emailCheck = "NO_EMAIL_" + nrc // Use a placeholder if email is empty
		}
		if err := database.DB.Where("nrc = ? OR (email IS NOT NULL AND email = ?)", nrc, emailCheck).First(&existing).Error; err == nil {
			errors = append(errors, fmt.Sprintf("Row %d: NRC or email already exists", rowNum))
			failed++
			continue
		}

		hashedPassword, err := utils.HashPassword(password)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Row %d: Failed to process password", rowNum))
			failed++
			continue
		}

		var emailPtr *string
		if email != "" {
			emailPtr = &email
		}
		employee := models.Employee{
			NRC:          &nrc,
			Firstname:    firstname,
			Lastname:     lastname,
			Email:        emailPtr,
			PasswordHash: hashedPassword,
			Department:   department,
			Role:         models.Role(role),
		}

		if err := database.DB.Create(&employee).Error; err != nil {
			errors = append(errors, fmt.Sprintf("Row %d: Failed to create employee", rowNum))
			failed++
			continue
		}

		success++
	}

	c.JSON(http.StatusOK, BulkUploadResponse{
		Total:   total,
		Success: success,
		Failed:  failed,
		Errors:  errors,
	})
}

// ExportEmployees exports all employees data to PDF
// @Summary Export all employees
// @Description Export all employees data to PDF format (Admin only)
// @Tags Admin - Employees
// @Produce application/pdf
// @Security BearerAuth
// @Success 200 {file} file "PDF file"
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/employees/export [get]
func ExportEmployees(c *gin.Context) {
	// Get all employees (excluding admin users)
	var employees []models.Employee
	if err := database.DB.Where("role != ?", models.RoleAdmin).Find(&employees).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch employees"})
		return
	}

	// Convert to export format
	exportData := make([]utils.EmployeeDataExport, 0, len(employees))
	for _, emp := range employees {
		// Get employment details for start date and tenure
		var employment models.EmploymentDetails
		var startDate string
		var tenure string
		if err := database.DB.Where("employee_id = ?", emp.ID).First(&employment).Error; err == nil {
			if employment.StartDate != nil {
				startDate = employment.StartDate.Format("2006-01-02")
			} else if employment.HireDate != nil {
				startDate = employment.HireDate.Format("2006-01-02")
			}
			// Calculate tenure
			if employment.StartDate != nil {
				start := *employment.StartDate
				now := time.Now()
				years := now.Year() - start.Year()
				months := int(now.Month()) - int(start.Month())
				if months < 0 {
					years--
					months += 12
				}
				if years > 0 {
					tenure = fmt.Sprintf("%d years, %d months", years, months)
				} else {
					tenure = fmt.Sprintf("%d months", months)
				}
			}
		}

		exportData = append(exportData, utils.EmployeeDataExport{
			ID:                        emp.ID,
			EmployeeNumber:            getStringValue(emp.EmployeeNumber),
			NRC:                       getStringValue(emp.NRC),
			Username:                  getStringValue(emp.Username),
			Firstname:                 emp.Firstname,
			Lastname:                  emp.Lastname,
			Email:                     getStringValue(emp.Email),
			Department:                emp.Department,
			Role:                      string(emp.Role),
			Phone:                     getStringValue(emp.Phone),
			Mobile:                    getStringValue(emp.Mobile),
			Address:                   getStringValue(emp.Address),
			City:                      getStringValue(emp.City),
			PostalCode:                getStringValue(emp.PostalCode),
			DateOfBirth:               formatDate(emp.DateOfBirth),
			Gender:                    getStringValue(emp.Gender),
			JobTitle:                  getStringValue(emp.JobTitle),
			EmploymentStatus:          getStringValue(emp.EmploymentStatus),
			StartDate:                 startDate,
			Tenure:                    tenure,
			EmergencyContactName:      getStringValue(emp.EmergencyContactName),
			EmergencyContactPhone:     getStringValue(emp.EmergencyContactPhone),
			EmergencyContactRelationship: getStringValue(emp.EmergencyContactRelationship),
			BankName:                  getStringValue(emp.BankName),
			BankAccountNumber:         getStringValue(emp.BankAccountNumber),
			TaxID:                     getStringValue(emp.TaxID),
			Notes:                     getStringValue(emp.Notes),
		})
	}

	// Generate PDF
	fileData, err := utils.ExportEmployeesToPDF(exportData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
		return
	}

	filename := fmt.Sprintf("employees_%s.pdf", time.Now().Format("20060102_150405"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/pdf")
	c.Data(http.StatusOK, "application/pdf", fileData)
}

// ExportEmployee exports single employee data to PDF
// @Summary Export employee
// @Description Export single employee detailed data to PDF format (Admin only)
// @Tags Admin - Employees
// @Produce application/pdf
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {file} file "PDF file"
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/employees/{id}/export [get]
func ExportEmployee(c *gin.Context) {
	employeeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee ID"})
		return
	}

	var employee models.Employee
	if err := database.DB.First(&employee, uint(employeeID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}

	// Get employment details
	var employment models.EmploymentDetails
	var startDate string
	var tenure string
	if err := database.DB.Where("employee_id = ?", employee.ID).First(&employment).Error; err == nil {
		if employment.StartDate != nil {
			startDate = employment.StartDate.Format("2006-01-02")
		} else if employment.HireDate != nil {
			startDate = employment.HireDate.Format("2006-01-02")
		}
		// Calculate tenure
		if employment.StartDate != nil {
			start := *employment.StartDate
			now := time.Now()
			years := now.Year() - start.Year()
			months := int(now.Month()) - int(start.Month())
			if months < 0 {
				years--
				months += 12
			}
			if years > 0 {
				tenure = fmt.Sprintf("%d years, %d months", years, months)
			} else {
				tenure = fmt.Sprintf("%d months", months)
			}
		}
	}

	exportData := utils.EmployeeDataExport{
		ID:                        employee.ID,
		EmployeeNumber:            getStringValue(employee.EmployeeNumber),
		NRC:                       getStringValue(employee.NRC),
		Username:                  getStringValue(employee.Username),
		Firstname:                 employee.Firstname,
		Lastname:                  employee.Lastname,
		Email:                     getStringValue(employee.Email),
		Department:                employee.Department,
		Role:                      string(employee.Role),
		Phone:                     getStringValue(employee.Phone),
		Mobile:                    getStringValue(employee.Mobile),
		Address:                   getStringValue(employee.Address),
		City:                      getStringValue(employee.City),
		PostalCode:                getStringValue(employee.PostalCode),
		DateOfBirth:               formatDate(employee.DateOfBirth),
		Gender:                    getStringValue(employee.Gender),
		JobTitle:                  getStringValue(employee.JobTitle),
		EmploymentStatus:          getStringValue(employee.EmploymentStatus),
		StartDate:                  startDate,
		Tenure:                    tenure,
		EmergencyContactName:      getStringValue(employee.EmergencyContactName),
		EmergencyContactPhone:     getStringValue(employee.EmergencyContactPhone),
		EmergencyContactRelationship: getStringValue(employee.EmergencyContactRelationship),
		BankName:                  getStringValue(employee.BankName),
		BankAccountNumber:         getStringValue(employee.BankAccountNumber),
		TaxID:                     getStringValue(employee.TaxID),
		Notes:                     getStringValue(employee.Notes),
	}

	// Generate PDF
	fileData, err := utils.ExportEmployeeToPDF(exportData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate PDF"})
		return
	}

	filename := fmt.Sprintf("employee_%s_%s_%s.pdf", employee.Firstname, employee.Lastname, time.Now().Format("20060102"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/pdf")
	c.Data(http.StatusOK, "application/pdf", fileData)
}

// Helper functions
func getStringValue(s *string) string {
	if s == nil {
		return "-"
	}
	return *s
}

func formatDate(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("2006-01-02")
}
