package handlers

import (
	"hrms-api/database"
	"hrms-api/models"
	"hrms-api/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// LoginRequest represents login credentials (use NRC for employees/managers, username for admins)
type LoginRequest struct {
	NRC      string `json:"nrc,omitempty" example:"123456/78/9"`
	Username string `json:"username,omitempty" example:"admin"`
	Password string `json:"password" binding:"required" example:"password123"`
}

// RegisterRequest represents new employee registration data
type RegisterRequest struct {
	NRC        string      `json:"nrc" binding:"required" example:"123456/78/9"`
	Firstname  string      `json:"firstname" binding:"required" example:"John"`
	Lastname   string      `json:"lastname" binding:"required" example:"Doe"`
	Email      string      `json:"email" binding:"required,email" example:"john@example.com"`
	Password   string      `json:"password" binding:"required,min=6" example:"password123"`
	Department string      `json:"department" example:"IT"`
	Role       models.Role `json:"role" example:"employee"`
}

// AdminLoginRequest represents admin login credentials
type AdminLoginRequest struct {
	Username string `json:"username" binding:"required" example:"admin"`
	Password string `json:"password" binding:"required" example:"password123"`
}

// AuthResponse represents authentication response with token
type AuthResponse struct {
	Token    string          `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	Employee models.Employee `json:"employee"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid credentials"`
}

// Login authenticates an employee/manager with NRC and password
// @Summary Employee/Manager login
// @Description Authenticate employee or manager with NRC and password, returns JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials (use NRC)"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/login [post]
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.NRC == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NRC is required for employee/manager login"})
		return
	}

	var employee models.Employee
	if err := database.DB.Where("nrc = ?", req.NRC).First(&employee).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Prevent admin from logging in via NRC endpoint
	if employee.Role == models.RoleAdmin {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admins must use /auth/admin/login"})
		return
	}

	if !utils.CheckPasswordHash(req.Password, employee.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := utils.GenerateToken(&employee)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Clear password hash from response
	employee.PasswordHash = ""
	c.JSON(http.StatusOK, AuthResponse{
		Token:    token,
		Employee: employee,
	})
}

// AdminLogin authenticates an admin with username and password
// @Summary Admin login
// @Description Authenticate admin with username and password, returns JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body AdminLoginRequest true "Admin login credentials"
// @Success 200 {object} AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/admin/login [post]
func AdminLogin(c *gin.Context) {
	var req AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var employee models.Employee
	if err := database.DB.Where("username = ? AND role = ?", req.Username, models.RoleAdmin).First(&employee).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check password hash
	passwordValid := utils.CheckPasswordHash(req.Password, employee.PasswordHash)
	if !passwordValid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := utils.GenerateToken(&employee)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	employee.PasswordHash = ""
	c.JSON(http.StatusOK, AuthResponse{
		Token:    token,
		Employee: employee,
	})
}

// Register creates a new employee account
// @Summary Register new employee
// @Description Create a new employee account
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Employee registration data"
// @Success 201 {object} AuthResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse "NRC or email already exists"
// @Router /auth/register [post]
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate role if provided
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role. Must be: employee, manager, or admin"})
			return
		}
	} else {
		req.Role = models.RoleEmployee
	}

	// Admins cannot be registered via this endpoint
	if req.Role == models.RoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Admin accounts cannot be created via registration"})
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

	token, err := utils.GenerateToken(&employee)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	employee.PasswordHash = ""
	c.JSON(http.StatusCreated, AuthResponse{
		Token:    token,
		Employee: employee,
	})
}
