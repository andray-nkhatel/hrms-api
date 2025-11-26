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
	Email      string      `json:"email" binding:"required,email" example:"jane@example.com"`
	Password   string      `json:"password" binding:"required,min=6" example:"password123"`
	Department string      `json:"department" example:"HR"`
	Role       models.Role `json:"role" binding:"required" example:"manager"`
}

// CreateAdminRequest represents data for creating an admin (uses username)
type CreateAdminRequest struct {
	Username   string `json:"username" binding:"required" example:"admin2"`
	Firstname  string `json:"firstname" binding:"required" example:"Admin"`
	Lastname   string `json:"lastname" binding:"required" example:"User"`
	Email      string `json:"email" binding:"required,email" example:"admin2@example.com"`
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

	// Check if NRC or email already exists
	var existingEmployee models.Employee
	if err := database.DB.Where("nrc = ? OR email = ?", req.NRC, req.Email).First(&existingEmployee).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "NRC or email already exists"})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	nrc := req.NRC
	employee := models.Employee{
		NRC:          &nrc,
		Firstname:    req.Firstname,
		Lastname:     req.Lastname,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Department:   req.Department,
		Role:         req.Role,
	}

	if err := database.DB.Create(&employee).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create employee"})
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

	// Check if username or email already exists
	var existingEmployee models.Employee
	if err := database.DB.Where("username = ? OR email = ?", req.Username, req.Email).First(&existingEmployee).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username or email already exists"})
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	username := req.Username
	employee := models.Employee{
		Username:     &username,
		Firstname:    req.Firstname,
		Lastname:     req.Lastname,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Department:   req.Department,
		Role:         models.RoleAdmin,
	}

	if err := database.DB.Create(&employee).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create admin"})
		return
	}

	employee.PasswordHash = ""
	c.JSON(http.StatusCreated, employee)
}

// GetEmployees returns all employees
// @Summary Get all employees
// @Description Get list of all employees (Admin only)
// @Tags Admin - Employees
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Employee
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/employees [get]
func GetEmployees(c *gin.Context) {
	var employees []models.Employee
	if err := database.DB.Select("id", "nrc", "username", "firstname", "lastname", "email", "department", "role", "created_at", "updated_at").
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
		Firstname  string      `json:"firstname"`
		Lastname   string      `json:"lastname"`
		Email      string      `json:"email" binding:"omitempty,email"`
		Department string      `json:"department"`
		Role       models.Role `json:"role"`
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
	if req.Email != "" {
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
		if err := database.DB.Where("nrc = ? OR email = ?", nrc, email).First(&existing).Error; err == nil {
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

		employee := models.Employee{
			NRC:          &nrc,
			Firstname:    firstname,
			Lastname:     lastname,
			Email:        email,
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
