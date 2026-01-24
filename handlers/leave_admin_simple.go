package handlers

import (
	"hrms-api/database"
	"hrms-api/models"
	"hrms-api/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// RecordLeaveTakenRequest represents a request to record leave taken by admin
type RecordLeaveTakenRequest struct {
	EmployeeID  uint   `json:"employee_id" binding:"required" example:"1"`
	LeaveTypeID uint   `json:"leave_type_id" binding:"required" example:"1"`
	StartDate   string `json:"start_date" binding:"required" example:"2026-01-15"`
	EndDate     string `json:"end_date" binding:"required" example:"2026-01-17"`
	Remarks     string `json:"remarks,omitempty" example:"Annual vacation"`
}

// RecordLeaveTaken records leave taken by an employee (Admin only)
// @Summary Record leave taken
// @Description Admin records actual leave taken for an employee (no approval workflow)
// @Tags Admin - Leave Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body RecordLeaveTakenRequest true "Leave taken data"
// @Success 201 {object} models.LeaveTaken
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/admin/leave-taken [post]
func RecordLeaveTaken(c *gin.Context) {
	var req RecordLeaveTakenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify employee exists
	var employee models.Employee
	if err := database.DB.First(&employee, req.EmployeeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}

	// Verify leave type exists
	var leaveType models.LeaveType
	if err := database.DB.First(&leaveType, req.LeaveTypeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Leave type not found"})
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format. Use YYYY-MM-DD"})
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format. Use YYYY-MM-DD"})
		return
	}

	// Validate dates
	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End date must be after or equal to start date"})
		return
	}

	// Get current user (admin)
	userID, _ := c.Get("user_id")
	adminID := userID.(uint)

	// Calculate days taken
	daysTaken := float64(int(endDate.Sub(startDate).Hours()/24) + 1)

	// Create leave taken record
	leaveTaken := models.LeaveTaken{
		EmployeeID:  req.EmployeeID,
		LeaveTypeID: req.LeaveTypeID,
		StartDate:   startDate,
		EndDate:     endDate,
		DaysTaken:   daysTaken,
		RecordedBy:  adminID,
		Remarks:     &req.Remarks,
		RecordedAt:  time.Now(),
	}

	// Calculate days taken (in case we need to override)
	leaveTaken.DaysTaken = leaveTaken.CalculateDaysTaken()

	if err := database.DB.Create(&leaveTaken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create leave taken record"})
		return
	}

	// Load associations
	database.DB.Preload("Employee").Preload("LeaveType").Preload("Recorder").First(&leaveTaken, leaveTaken.ID)

	c.JSON(http.StatusCreated, leaveTaken)
}

// GetLeaveBalanceSimple returns the leave balance for an employee using simplified calculation
// @Summary Get leave balance (simplified)
// @Description Get leave balance calculated as Total Accrued - Total Taken
// @Tags Admin - Leave Management
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param leave_type_id query int false "Leave type ID (defaults to Annual leave)"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/admin/employees/{id}/leave-balance [get]
func GetLeaveBalanceSimple(c *gin.Context) {
	employeeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee ID"})
		return
	}

	// Verify employee exists
	var employee models.Employee
	if err := database.DB.First(&employee, employeeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}

	// Get leave type (default to Annual)
	var leaveTypeID uint
	leaveTypeIDStr := c.Query("leave_type_id")
	if leaveTypeIDStr != "" {
		parsed, err := strconv.ParseUint(leaveTypeIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid leave_type_id"})
			return
		}
		leaveTypeID = uint(parsed)
	} else {
		// Default to Annual leave
		var annualLeaveType models.LeaveType
		if err := database.DB.Where("name = ?", "Annual").First(&annualLeaveType).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
			return
		}
		leaveTypeID = annualLeaveType.ID
	}

	// Calculate balance using simplified formula
	balance, err := utils.CalculateLeaveBalanceSimple(uint(employeeID), leaveTypeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate leave balance"})
		return
	}

	// Get total accrued and total taken for details
	var totalAccrued float64
	database.DB.Model(&models.LeaveAccrual{}).
		Where("employee_id = ? AND leave_type_id = ?", employeeID, leaveTypeID).
		Select("COALESCE(SUM(days_accrued), 0)").
		Scan(&totalAccrued)

	var totalTaken float64
	database.DB.Model(&models.LeaveTaken{}).
		Where("employee_id = ? AND leave_type_id = ?", employeeID, leaveTypeID).
		Select("COALESCE(SUM(days_taken), 0)").
		Scan(&totalTaken)

	c.JSON(http.StatusOK, gin.H{
		"employee_id":   employeeID,
		"employee_name": employee.Firstname + " " + employee.Lastname,
		"leave_type_id": leaveTypeID,
		"total_accrued": totalAccrued,
		"total_taken":   totalTaken,
		"balance":       balance,
	})
}

// GetEmployeeLeaveHistory returns all leave taken records for an employee
// @Summary Get employee leave history
// @Description Get all leave taken records for an employee (Admin only)
// @Tags Admin - Leave Management
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param leave_type_id query int false "Filter by leave type ID"
// @Success 200 {array} models.LeaveTaken
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/admin/employees/{id}/leave-taken [get]
func GetEmployeeLeaveHistory(c *gin.Context) {
	employeeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee ID"})
		return
	}

	// Verify employee exists
	var employee models.Employee
	if err := database.DB.First(&employee, employeeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}

	// Build query
	query := database.DB.Where("employee_id = ?", employeeID).
		Preload("Employee").
		Preload("LeaveType").
		Preload("Recorder").
		Order("start_date DESC, recorded_at DESC")

	// Apply leave type filter if provided
	leaveTypeIDStr := c.Query("leave_type_id")
	if leaveTypeIDStr != "" {
		leaveTypeID, err := strconv.ParseUint(leaveTypeIDStr, 10, 32)
		if err == nil {
			query = query.Where("leave_type_id = ?", leaveTypeID)
		}
	}

	var leaveTaken []models.LeaveTaken
	if err := query.Find(&leaveTaken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leave history"})
		return
	}

	c.JSON(http.StatusOK, leaveTaken)
}

// GetAllEmployeesLeaveBalancesSimple returns leave balances for all employees
// @Summary Get all employees leave balances
// @Description Get leave balances for all employees using simplified calculation (Admin only)
// @Tags Admin - Leave Management
// @Produce json
// @Security BearerAuth
// @Param leave_type_id query int false "Leave type ID (defaults to Annual leave)"
// @Success 200 {array} map[string]interface{}
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/admin/employees/leave-balances [get]
func GetAllEmployeesLeaveBalancesSimple(c *gin.Context) {
	// Get leave type (default to Annual)
	var leaveTypeID uint
	leaveTypeIDStr := c.Query("leave_type_id")
	if leaveTypeIDStr != "" {
		parsed, err := strconv.ParseUint(leaveTypeIDStr, 10, 32)
		if err == nil {
			leaveTypeID = uint(parsed)
		}
	}
	if leaveTypeID == 0 {
		// Default to Annual leave
		var annualLeaveType models.LeaveType
		if err := database.DB.Where("name = ?", "Annual").First(&annualLeaveType).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
			return
		}
		leaveTypeID = annualLeaveType.ID
	}

	// Get all active employees (exclude admins)
	var employees []models.Employee
	if err := database.DB.Where("role != ? AND status = ?", models.RoleAdmin, "active").Find(&employees).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch employees"})
		return
	}

	balances := make([]map[string]interface{}, 0, len(employees))

	for _, emp := range employees {
		// Calculate balance
		balance, err := utils.CalculateLeaveBalanceSimple(emp.ID, leaveTypeID)
		if err != nil {
			continue // Skip this employee if calculation fails
		}

		// Get totals for details
		var totalAccrued float64
		database.DB.Model(&models.LeaveAccrual{}).
			Where("employee_id = ? AND leave_type_id = ?", emp.ID, leaveTypeID).
			Select("COALESCE(SUM(days_accrued), 0)").
			Scan(&totalAccrued)

		var totalTaken float64
		database.DB.Model(&models.LeaveTaken{}).
			Where("employee_id = ? AND leave_type_id = ?", emp.ID, leaveTypeID).
			Select("COALESCE(SUM(days_taken), 0)").
			Scan(&totalTaken)

		balances = append(balances, map[string]interface{}{
			"employee_id":     emp.ID,
			"employee_name":   emp.Firstname + " " + emp.Lastname,
			"employee_number": emp.EmployeeNumber,
			"department":      emp.Department,
			"total_accrued":   totalAccrued,
			"total_taken":     totalTaken,
			"balance":         balance,
		})
	}

	c.JSON(http.StatusOK, balances)
}
