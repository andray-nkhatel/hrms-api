package handlers

import (
	"encoding/csv"
	"fmt"
	"hrms-api/database"
	"hrms-api/models"
	"hrms-api/utils"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// LeaveAccrualResponse represents accrual information
type LeaveAccrualResponse struct {
	Month       string  `json:"month" example:"2025-01"`
	DaysAccrued float64 `json:"days_accrued" example:"2.0"`
	DaysUsed    float64 `json:"days_used" example:"1.5"`
	DaysBalance float64 `json:"days_balance" example:"0.5"`
	IsProcessed bool    `json:"is_processed"`
	ProcessedAt *string `json:"processed_at,omitempty"`
}

// AnnualLeaveBalanceResponse represents detailed annual leave balance
type AnnualLeaveBalanceResponse struct {
	EmployeeID        uint                   `json:"employee_id"`
	EmployeeName      string                 `json:"employee_name"`
	TotalAccrued      float64                `json:"total_accrued" example:"24.0"`
	TotalUsed         float64                `json:"total_used" example:"5.0"`
	AllTimeNetBalance float64                `json:"all_time_net_balance" example:"19.0"` // TotalAccrued - TotalUsed (all-time net)
	CurrentBalance    float64                `json:"current_balance" example:"19.0"`      // Current available (includes carry-over)
	CarryOverBalance  float64                `json:"carry_over_balance" example:"5.0"`
	Accruals          []LeaveAccrualResponse `json:"accruals"`
	PendingLeaves     int                    `json:"pending_leaves"`
	UpcomingLeaves    int                    `json:"upcoming_leaves"`
}

// LeaveCalendarResponse represents leave calendar data
type LeaveCalendarResponse struct {
	Date         string  `json:"date" example:"2025-12-15"`
	EmployeeID   uint    `json:"employee_id"`
	EmployeeName string  `json:"employee_name"`
	Department   string  `json:"department"`
	LeaveType    string  `json:"leave_type"`
	LeaveID      uint    `json:"leave_id"`
	StartDate    string  `json:"start_date"`
	EndDate      string  `json:"end_date"`
	Status       string  `json:"status"`
	FormFilePath *string `json:"form_file_path,omitempty"`
	FormFileName *string `json:"form_file_name,omitempty"`
}

// DepartmentLeaveReport represents leave statistics by department
type DepartmentLeaveReport struct {
	Department      string  `json:"department"`
	TotalEmployees  int     `json:"total_employees"`
	TotalAccrued    float64 `json:"total_accrued"`
	TotalUsed       float64 `json:"total_used"`
	TotalBalance    float64 `json:"total_balance"`
	PendingRequests int     `json:"pending_requests"`
	UpcomingLeaves  int     `json:"upcoming_leaves"`
}

// GetAnnualLeaveBalance gets detailed annual leave balance for an employee
// @Summary Get annual leave balance details
// @Description Get detailed annual leave balance including accruals for an employee (HR/Admin only)
// @Tags HR - Leave Management
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Success 200 {object} AnnualLeaveBalanceResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/hr/employees/{id}/annual-leave-balance [get]
func GetAnnualLeaveBalance(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var employee models.Employee
	if err := database.DB.First(&employee, employeeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}

	// Get Annual leave type
	var annualLeaveType models.LeaveType
	if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
		return
	}

	// Ensure accruals are up to date
	if err := utils.EnsureAccrualsUpToDate(uint(employeeID), annualLeaveType.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process accruals"})
		return
	}

	// Get all accruals
	// Order by accrual_month if available, otherwise by year and month
	var accruals []models.LeaveAccrual
	database.DB.Where("employee_id = ? AND leave_type_id = ?", employeeID, annualLeaveType.ID).
		Order("COALESCE(accrual_month, MAKE_DATE(year::integer, month::integer, 1)) DESC, year DESC, month DESC").
		Find(&accruals)

	// Get employee start date to exclude first month accruals
	var employeeStartDate time.Time
	var employment models.EmploymentDetails
	if err := database.DB.Where("employee_id = ?", employeeID).First(&employment).Error; err == nil {
		if employment.HireDate != nil {
			employeeStartDate = *employment.HireDate
		} else if employment.StartDate != nil {
			employeeStartDate = *employment.StartDate
		} else {
			employeeStartDate = employee.CreatedAt
		}
	} else {
		employeeStartDate = employee.CreatedAt
	}
	firstMonthStart := time.Date(employeeStartDate.Year(), employeeStartDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Calculate totals (exclude first month accruals, but include initial balance adjustments)
	var totalAccrued float64
	accrualResponses := make([]LeaveAccrualResponse, 0, len(accruals))

	for _, acc := range accruals {
		// Get accrual month for comparison
		var accrualMonth time.Time
		if acc.AccrualMonth != nil {
			accrualMonth = *acc.AccrualMonth
		} else if acc.Year > 0 && acc.Month > 0 {
			accrualMonth = time.Date(acc.Year, time.Month(acc.Month), 1, 0, 0, 0, 0, time.UTC)
		}

		// Skip regular accruals in the first month of employment
		// BUT include initial balance adjustments (identified by Notes containing "Initial balance" or "set-initial")
		isInitialBalance := acc.Notes != nil && 
			(*acc.Notes != "" && (strings.Contains(*acc.Notes, "Initial balance") || 
			 strings.Contains(*acc.Notes, "set-initial") || 
			 strings.Contains(*acc.Notes, "Set initial")))
		
		if !accrualMonth.IsZero() && accrualMonth.Equal(firstMonthStart) && !isInitialBalance {
			continue
		}

		totalAccrued += acc.DaysAccrued

		processedAtStr := ""
		if acc.ProcessedAt != nil {
			processedAtStr = acc.ProcessedAt.Format(time.RFC3339)
		}

		accrualResponses = append(accrualResponses, LeaveAccrualResponse{
			Month:       acc.GetAccrualMonthKey(),
			DaysAccrued: acc.DaysAccrued,
			DaysUsed:    acc.DaysUsed,
			DaysBalance: acc.DaysBalance,
			IsProcessed: acc.IsProcessed,
			ProcessedAt: &processedAtStr,
		})
	}

	// Calculate total used directly from approved leave records (source of truth)
	// This ensures accuracy even if accrual records have incorrect DaysUsed values
	var totalUsed float64
	var approvedLeaves []models.Leave
	database.DB.Where("employee_id = ? AND leave_type_id = ? AND status = ?",
		employeeID, annualLeaveType.ID, models.StatusApproved).Find(&approvedLeaves)
	for _, leave := range approvedLeaves {
		totalUsed += float64(leave.GetDuration())
	}

	// Get carry-over balance
	var carryOverBalance float64
	if annualLeaveType.AllowCarryOver {
		carryOverBalance, _ = utils.GetCarryOverBalance(uint(employeeID), annualLeaveType.ID)
	}

	// Get total current balance (accrual + carry-over) - this is what's actually available
	currentBalance, _ := utils.GetCurrentLeaveBalance(uint(employeeID), annualLeaveType.ID)

	// Calculate all-time net balance using actual accrual records (includes initial balance adjustments)
	// This reflects the actual accrued amount including any manual adjustments from onboarding
	// AllTimeNetBalance = Total Accrued (from records) - Total Used (from approved leaves)
	allTimeNetBalance := totalAccrued - totalUsed
	// Don't set to 0 if negative - negative values are valid (overdrawn)

	// Get pending and upcoming leaves
	var pendingLeaves, upcomingLeaves int64
	now := time.Now()
	database.DB.Model(&models.Leave{}).
		Where("employee_id = ? AND leave_type_id = ? AND status = ?", employeeID, annualLeaveType.ID, models.StatusPending).
		Count(&pendingLeaves)
	database.DB.Model(&models.Leave{}).
		Where("employee_id = ? AND leave_type_id = ? AND status = ? AND start_date > ?",
			employeeID, annualLeaveType.ID, models.StatusApproved, now).
		Count(&upcomingLeaves)

	response := AnnualLeaveBalanceResponse{
		EmployeeID:        uint(employeeID),
		EmployeeName:      employee.Firstname + " " + employee.Lastname,
		TotalAccrued:      totalAccrued,
		TotalUsed:         totalUsed,
		AllTimeNetBalance: allTimeNetBalance,
		CurrentBalance:    currentBalance,
		CarryOverBalance:  carryOverBalance,
		Accruals:          accrualResponses,
		PendingLeaves:     int(pendingLeaves),
		UpcomingLeaves:    int(upcomingLeaves),
	}

	c.JSON(http.StatusOK, response)
}

// GetLeaveCalendar gets leave calendar for a date range
// @Summary Get leave calendar
// @Description Get leave calendar showing all approved leaves in a date range (HR/Admin only)
// @Tags HR - Leave Management
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "Start date (YYYY-MM-DD)" default:"current month start"
// @Param end_date query string false "End date (YYYY-MM-DD)" default:"current month end"
// @Param department query string false "Filter by department"
// @Success 200 {array} LeaveCalendarResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/hr/leaves/calendar [get]
func GetLeaveCalendar(c *gin.Context) {
	// Parse date range
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")
	department := c.Query("department")

	var startDate, endDate time.Time
	var err error

	if startDateStr == "" {
		now := time.Now()
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		startDate, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format"})
			return
		}
	}

	if endDateStr == "" {
		now := time.Now()
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		endDate = startOfMonth.AddDate(0, 1, 0).AddDate(0, 0, -1)
	} else {
		endDate, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format"})
			return
		}
	}

	// Get approved leaves in date range
	// Optimize query: filter by status first (indexed), then date range
	// Use overlapping date range logic: leave overlaps if start_date <= endDate AND end_date >= startDate
	// Use Joins to ensure Employee and LeaveType data is loaded
	// Exclude admin users and soft-deleted employees (same filter as employee list)
	query := database.DB.Model(&models.Leave{}).
		Select("leaves.*, employees.firstname, employees.lastname, employees.department, leave_types.name as leave_type_name").
		Joins("INNER JOIN employees ON leaves.employee_id = employees.id").
		Joins("LEFT JOIN leave_types ON leaves.leave_type_id = leave_types.id").
		Where("leaves.status = ?", models.StatusApproved).
		Where("leaves.start_date <= ?", endDate).
		Where("leaves.end_date >= ?", startDate).
		Where("employees.role != ?", models.RoleAdmin).
		Where("employees.deleted_at IS NULL")

	if department != "" {
		query = query.Where("employees.department = ?", department)
	}

	var results []struct {
		models.Leave
		FirstName      string `gorm:"column:firstname"`
		LastName       string `gorm:"column:lastname"`
		Department     string `gorm:"column:department"`
		LeaveTypeName  string `gorm:"column:leave_type_name"`
	}

	if err := query.Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaves"})
		return
	}

	// Generate calendar entries for each day
	calendar := make([]LeaveCalendarResponse, 0)
	currentDate := startDate
	for !currentDate.After(endDate) {
		for _, result := range results {
			leave := result.Leave
			if !currentDate.Before(leave.StartDate) && !currentDate.After(leave.EndDate) {
				// Get employee name and department from joined data
				employeeName := ""
				departmentName := ""
				if result.FirstName != "" && result.LastName != "" {
					employeeName = result.FirstName + " " + result.LastName
					departmentName = result.Department
				}
				
				leaveTypeName := result.LeaveTypeName
				
				calendar = append(calendar, LeaveCalendarResponse{
					Date:         currentDate.Format("2006-01-02"),
					EmployeeID:   leave.EmployeeID,
					EmployeeName: employeeName,
					Department:   departmentName,
					LeaveType:    leaveTypeName,
					LeaveID:      leave.ID,
					StartDate:    leave.StartDate.Format("2006-01-02"),
					EndDate:      leave.EndDate.Format("2006-01-02"),
					Status:       string(leave.Status),
					FormFilePath: leave.FormFilePath,
					FormFileName: leave.FormFileName,
				})
			}
		}
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	c.JSON(http.StatusOK, calendar)
}

// GetDepartmentLeaveReport gets leave statistics by department
// @Summary Get department leave report
// @Description Get leave statistics aggregated by department (HR/Admin only)
// @Tags HR - Leave Management
// @Produce json
// @Security BearerAuth
// @Success 200 {array} DepartmentLeaveReport
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/hr/leaves/department-report [get]
func GetDepartmentLeaveReport(c *gin.Context) {
	// Get all departments
	var departments []string
	database.DB.Model(&models.Employee{}).
		Where("department IS NOT NULL AND department != ''").
		Distinct("department").
		Pluck("department", &departments)

	reports := make([]DepartmentLeaveReport, 0, len(departments))

	// Get Annual leave type
	var annualLeaveType models.LeaveType
	if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
		return
	}

	for _, dept := range departments {
		// Count employees in department
		var totalEmployees int64
		database.DB.Model(&models.Employee{}).Where("department = ?", dept).Count(&totalEmployees)

		// Get employees in department
		var employees []models.Employee
		database.DB.Where("department = ?", dept).Find(&employees)

		var totalAccrued, totalUsed, totalBalance float64
		var pendingRequests, upcomingLeaves int64

		for _, emp := range employees {
			// Ensure accruals are up to date
			utils.EnsureAccrualsUpToDate(emp.ID, annualLeaveType.ID)

			// Get current balance
			balance, _ := utils.GetCurrentLeaveBalance(emp.ID, annualLeaveType.ID)
			totalBalance += balance

			// Get accruals for totals
			var accruals []models.LeaveAccrual
			database.DB.Where("employee_id = ? AND leave_type_id = ?", emp.ID, annualLeaveType.ID).Find(&accruals)
			for _, acc := range accruals {
				totalAccrued += acc.DaysAccrued
				totalUsed += acc.DaysUsed
			}

			// Count pending and upcoming
			var pending, upcoming int64
			now := time.Now()
			database.DB.Model(&models.Leave{}).
				Where("employee_id = ? AND leave_type_id = ? AND status = ?", emp.ID, annualLeaveType.ID, models.StatusPending).
				Count(&pending)
			database.DB.Model(&models.Leave{}).
				Where("employee_id = ? AND leave_type_id = ? AND status = ? AND start_date > ?",
					emp.ID, annualLeaveType.ID, models.StatusApproved, now).
				Count(&upcoming)

			pendingRequests += pending
			upcomingLeaves += upcoming
		}

		reports = append(reports, DepartmentLeaveReport{
			Department:      dept,
			TotalEmployees:  int(totalEmployees),
			TotalAccrued:    totalAccrued,
			TotalUsed:       totalUsed,
			TotalBalance:    totalBalance,
			PendingRequests: int(pendingRequests),
			UpcomingLeaves:  int(upcomingLeaves),
		})
	}

	c.JSON(http.StatusOK, reports)
}

// ProcessAccrualsRequest represents a request to process accruals
type ProcessAccrualsRequest struct {
	Month       string `json:"month" example:"2024-12"` // Month to process (YYYY-MM), optional
	EmployeeIDs []uint `json:"employee_ids,omitempty"`  // Optional: specific employee IDs to process. If empty, processes all employees
}

// ProcessMonthlyAccruals processes leave accruals for employees for a specific month
// @Summary Process monthly accruals
// @Description Process leave accruals for all employees or selected employees for a specific month (Manager/Admin only)
// @Tags HR - Leave Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ProcessAccrualsRequest false "Accrual processing request"
// @Param month query string false "Month to process (YYYY-MM) - deprecated, use request body"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/hr/leaves/process-accruals [post]
func ProcessMonthlyAccruals(c *gin.Context) {
	var req ProcessAccrualsRequest

	// Try to bind JSON body first
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			// If JSON binding fails, try query parameter for backward compatibility
			monthStr := c.Query("month")
			if monthStr != "" {
				req.Month = monthStr
			}
		}
	} else {
		// No body, use query parameter for backward compatibility
		req.Month = c.Query("month")
	}

	var processMonth time.Time
	var err error

	if req.Month == "" {
		// Default to current month (accruals are processed at end of month)
		now := time.Now()
		processMonth = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		processMonth, err = time.Parse("2006-01", req.Month)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid month format. Use YYYY-MM"})
			return
		}
	}

	// Get Annual leave type
	var annualLeaveType models.LeaveType
	if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
		return
	}

	// Get employees to process
	var employees []models.Employee
	if len(req.EmployeeIDs) > 0 {
		// Process only selected employees (but still exclude admins/inactive)
		if err := database.DB.
			Where("id IN ?", req.EmployeeIDs).
			Where("role != ? AND status = ?", models.RoleAdmin, "active").
			Find(&employees).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch selected employees"})
			return
		}
		if len(employees) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No valid employees found for the provided IDs"})
			return
		}
	} else {
		// Process all active, non-admin employees
		if err := database.DB.
			Where("role != ? AND status = ?", models.RoleAdmin, "active").
			Find(&employees).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch employees"})
			return
		}
	}

	processed := 0
	errors := 0
	var errorDetails []string

	for _, emp := range employees {
		if err := utils.ProcessMonthlyAccrual(emp.ID, annualLeaveType.ID, processMonth); err != nil {
			errors++
			errorDetails = append(errorDetails, fmt.Sprintf("Employee %d (%s %s): %v", emp.ID, emp.Firstname, emp.Lastname, err))
			continue
		}
		processed++
	}

	response := gin.H{
		"message":   "Accrual processing completed",
		"month":     processMonth.Format("2006-01"),
		"processed": processed,
		"errors":    errors,
		"total":     len(employees),
	}

	if len(errorDetails) > 0 {
		response["error_details"] = errorDetails
	}

	c.JSON(http.StatusOK, response)
}

// GetUpcomingLeaves gets all upcoming approved leaves
// @Summary Get upcoming leaves
// @Description Get all upcoming approved leaves within specified days (HR/Admin only)
// @Tags HR - Leave Management
// @Produce json
// @Security BearerAuth
// @Param days query int false "Number of days to look ahead" default:"30"
// @Success 200 {array} models.Leave
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/hr/leaves/upcoming [get]
func GetUpcomingLeaves(c *gin.Context) {
	daysStr := c.Query("days")
	days := 30 // default

	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	now := time.Now()
	endDate := now.AddDate(0, 0, days)

	var leaves []models.Leave
	database.DB.Where("status = ? AND start_date >= ? AND start_date <= ?",
		models.StatusApproved, now, endDate).
		Preload("Employee").
		Preload("LeaveType").
		Order("start_date ASC").
		Find(&leaves)

	c.JSON(http.StatusOK, leaves)
}

// AdjustLeaveBalanceRequest represents a request to manually adjust leave balance
type AdjustLeaveBalanceRequest struct {
	Days           float64 `json:"days" binding:"required" example:"2.5"`
	Reason         string  `json:"reason" binding:"required" example:"Manual adjustment for carryover"`
	AdjustmentDate *string `json:"adjustment_date,omitempty" example:"2025-12-01"`
}

// AdjustLeaveBalance manually adjusts an employee's annual leave balance
// @Summary Adjust leave balance
// @Description Manually adjust an employee's annual leave balance (add or subtract days) (Admin only)
// @Tags HR - Leave Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param request body AdjustLeaveBalanceRequest true "Balance adjustment"
// @Success 200 {object} AnnualLeaveBalanceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/hr/employees/{id}/annual-leave-balance/adjust [post]
func AdjustLeaveBalance(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req AdjustLeaveBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get Annual leave type
	var annualLeaveType models.LeaveType
	if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
		return
	}

	// Ensure accruals are up to date
	if err := utils.EnsureAccrualsUpToDate(uint(employeeID), annualLeaveType.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process accruals"})
		return
	}

	// Get the latest accrual record
	// Order by accrual_month if available, otherwise by year and month
	var latestAccrual models.LeaveAccrual
	if err := database.DB.Where("employee_id = ? AND leave_type_id = ?", employeeID, annualLeaveType.ID).
		Order("COALESCE(accrual_month, MAKE_DATE(year::integer, month::integer, 1)) DESC, year DESC, month DESC").
		First(&latestAccrual).Error; err != nil {
		// No accrual record exists, create one for current month
		now := time.Now()
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		latestAccrual = models.LeaveAccrual{
			EmployeeID:   uint(employeeID),
			LeaveTypeID:  annualLeaveType.ID,
			AccrualMonth: &monthStart,
			DaysAccrued:  0,
			DaysUsed:     0,
			DaysBalance:  0,
			IsProcessed:  true,
			ProcessedAt:  &now,
		}
		database.DB.Create(&latestAccrual)
	}

	// Adjust balance
	oldBalance := latestAccrual.DaysBalance
	latestAccrual.DaysBalance += req.Days
	// Allow negative balances (overdrawn) to be visible - don't force to 0

	// DaysUsed should always reflect actual approved leave records
	// Manual balance adjustments don't change DaysUsed - it will be recalculated from actual leaves
	// when ProcessMonthlyAccrual runs next

	// Mark as processed to prevent automatic recalculation from overwriting manual adjustments
	// This ensures manual adjustments are preserved when accruals are processed
	now := time.Now()
	latestAccrual.IsProcessed = true
	latestAccrual.ProcessedAt = &now

	// Add adjustment to notes
	notes := fmt.Sprintf("Manual adjustment: %+.2f days. Previous balance: %.2f, New balance: %.2f. DaysUsed remains: %.2f (calculated from actual leave records). Reason: %s",
		req.Days, oldBalance, latestAccrual.DaysBalance, latestAccrual.DaysUsed, req.Reason)
	if latestAccrual.Notes != nil && *latestAccrual.Notes != "" {
		notes = *latestAccrual.Notes + "\n" + notes
	}
	latestAccrual.Notes = &notes

	if err := database.DB.Save(&latestAccrual).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to adjust balance"})
		return
	}

	// Create audit log
	user := getCurrentUser(c)
	if user != nil {
		createAuditLog(models.AuditEntityEmployee, uint(employeeID), models.AuditActionUpdate, user.ID, c,
			map[string]interface{}{"balance": oldBalance},
			map[string]interface{}{"balance": latestAccrual.DaysBalance, "adjustment": req.Days, "reason": req.Reason})
	}

	// Return updated balance
	GetAnnualLeaveBalance(c)
}

// SetInitialBalanceRequest represents a request to set the initial balance (for onboarding)
type SetInitialBalanceRequest struct {
	Balance     float64  `json:"balance" binding:"required" example:"15.5"` // The absolute balance to set
	DaysUsed    *float64 `json:"days_used,omitempty" example:"8.5"`         // Optional: Total days used (for historical tracking)
	DaysAccrued *float64 `json:"days_accrued,omitempty" example:"24.0"`     // Optional: Total days accrued (for historical tracking)
	Reason      string   `json:"reason" binding:"required" example:"Initial balance from old system"`
	AsOfMonth   string   `json:"as_of_month,omitempty" example:"2025-12"` // Optional: Month this balance is as of (YYYY-MM format)
	ResetAll    bool     `json:"reset_all,omitempty" example:"false"`     // Optional: If true, delete all old accruals and start fresh
}

// SetInitialBalance sets the initial balance for an employee (for onboarding from old system)
// This sets the balance to an absolute value rather than adjusting it
// @Summary Set initial balance
// @Description Set the initial/annual leave balance to a specific value (for onboarding employees from old system)
// @Tags HR - Leave Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param request body SetInitialBalanceRequest true "Initial balance data"
// @Success 200 {object} AnnualLeaveBalanceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/hr/employees/{id}/annual-leave-balance/set-initial [post]
func SetInitialBalance(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req SetInitialBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Balance < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Balance cannot be negative"})
		return
	}

	// Get Annual leave type
	var annualLeaveType models.LeaveType
	if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
		return
	}

	// If reset_all is true, delete all existing accruals first
	if req.ResetAll {
		if err := database.DB.Where("employee_id = ? AND leave_type_id = ?", employeeID, annualLeaveType.ID).
			Delete(&models.LeaveAccrual{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset accruals: " + err.Error()})
			return
		}
	}

	// Get employee start date to validate against first month
	var employeeStartDate time.Time
	var employment models.EmploymentDetails
	if err := database.DB.Where("employee_id = ?", employeeID).First(&employment).Error; err == nil {
		if employment.HireDate != nil {
			employeeStartDate = *employment.HireDate
		} else if employment.StartDate != nil {
			employeeStartDate = *employment.StartDate
		} else {
			var emp models.Employee
			if err := database.DB.First(&emp, employeeID).Error; err == nil {
				employeeStartDate = emp.CreatedAt
			}
		}
	} else {
		var emp models.Employee
		if err := database.DB.First(&emp, employeeID).Error; err == nil {
			employeeStartDate = emp.CreatedAt
		}
	}
	firstMonthStart := time.Date(employeeStartDate.Year(), employeeStartDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Determine the month to use
	var monthStart time.Time
	if req.AsOfMonth != "" {
		parsed, err := time.Parse("2006-01", req.AsOfMonth)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid as_of_month format. Use YYYY-MM"})
			return
		}
		monthStart = time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		// Default to current month
		now := time.Now()
		monthStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}

	// Prevent creating accrual records for the first month of employment
	if monthStart.Equal(firstMonthStart) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cannot set initial balance for the employee's first month of employment. Accrual starts from the second month.",
		})
		return
	}

	// Get or create accrual record for this month
	var accrual models.LeaveAccrual
	err := database.DB.Where("employee_id = ? AND leave_type_id = ? AND accrual_month = ?",
		employeeID, annualLeaveType.ID, monthStart).First(&accrual).Error

	now := time.Now()
	if err != nil {
		// Create new accrual record
		// If days_accrued and days_used are provided, use them; otherwise calculate
		var daysAccrued, daysUsed float64
		if req.DaysAccrued != nil {
			daysAccrued = *req.DaysAccrued
		}
		if req.DaysUsed != nil {
			daysUsed = *req.DaysUsed
		}

		// If not provided, we need to calculate DaysAccrued properly
		// The balance represents what's available now, so we need to know what was used
		// If DaysUsed is not provided, we'll calculate it from existing approved leaves
		if req.DaysAccrued == nil && req.DaysUsed == nil {
			// Calculate total days used from approved leave records
			var existingLeaves []models.Leave
			database.DB.Where("employee_id = ? AND leave_type_id = ? AND status = ?",
				uint(employeeID), annualLeaveType.ID, models.StatusApproved).Find(&existingLeaves)
			
			var totalUsedFromLeaves float64
			for _, leave := range existingLeaves {
				totalUsedFromLeaves += float64(leave.GetDuration())
			}
			
			// DaysAccrued = Current Balance + Total Used (because balance = accrued - used)
			daysAccrued = req.Balance + totalUsedFromLeaves
			daysUsed = totalUsedFromLeaves
		} else if req.DaysAccrued == nil && req.DaysUsed != nil {
			// DaysUsed provided but DaysAccrued not - calculate it
			daysAccrued = req.Balance + *req.DaysUsed
			daysUsed = *req.DaysUsed
		} else if req.DaysAccrued != nil && req.DaysUsed == nil {
			// DaysAccrued provided but DaysUsed not - calculate it
			daysUsed = *req.DaysAccrued - req.Balance
			if daysUsed < 0 {
				daysUsed = 0
			}
			daysAccrued = *req.DaysAccrued
		} else {
			// Both provided - use them
			daysAccrued = *req.DaysAccrued
			daysUsed = *req.DaysUsed
		}

		accrual = models.LeaveAccrual{
			EmployeeID:   uint(employeeID),
			LeaveTypeID:  annualLeaveType.ID,
			AccrualMonth: &monthStart,
			DaysAccrued:  daysAccrued,
			DaysUsed:     daysUsed,
			DaysBalance:  req.Balance, // Set to the requested balance
			IsProcessed:  true,
			ProcessedAt:  &now,
			Notes:        &req.Reason,
		}

		if err := database.DB.Create(&accrual).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create accrual record"})
			return
		}
	} else {
		// Update existing accrual record
		oldBalance := accrual.DaysBalance

		// Update balance to the requested value
		accrual.DaysBalance = req.Balance

		// Update days accrued and used
		// If explicitly provided, use those values
		// If not provided, calculate based on balance to ensure consistency
		if req.DaysAccrued != nil {
			accrual.DaysAccrued = *req.DaysAccrued
		} else {
			// If not provided, calculate from balance and used days
			if req.DaysUsed != nil {
				accrual.DaysAccrued = req.Balance + *req.DaysUsed
			} else {
				// If neither provided, assume balance = accrued - used
				// So accrued = balance + existing used
				accrual.DaysAccrued = req.Balance + accrual.DaysUsed
			}
		}

		if req.DaysUsed != nil {
			accrual.DaysUsed = *req.DaysUsed
		} else if req.DaysAccrued != nil {
			// If accrued was provided but used wasn't, calculate used from balance
			accrual.DaysUsed = *req.DaysAccrued - req.Balance
			if accrual.DaysUsed < 0 {
				accrual.DaysUsed = 0
			}
		} else {
			// If neither was provided, keep existing used but adjust accrued to match balance
			// Ensure: balance = accrued - used, so accrued = balance + used
			accrual.DaysAccrued = req.Balance + accrual.DaysUsed
		}

		// Add note
		notes := fmt.Sprintf("Initial balance set: %.2f days (was %.2f). Reason: %s", req.Balance, oldBalance, req.Reason)
		if accrual.Notes != nil && *accrual.Notes != "" {
			notes = *accrual.Notes + "\n" + notes
		}
		accrual.Notes = &notes

		accrual.IsProcessed = true
		accrual.ProcessedAt = &now

		if err := database.DB.Save(&accrual).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update accrual record"})
			return
		}
	}

	// Create audit log
	user := getCurrentUser(c)
	if user != nil {
		auditData := map[string]interface{}{
			"initial_balance": req.Balance,
			"reason":          req.Reason,
		}
		if req.DaysAccrued != nil {
			auditData["days_accrued"] = *req.DaysAccrued
		}
		if req.DaysUsed != nil {
			auditData["days_used"] = *req.DaysUsed
		}
		createAuditLog(models.AuditEntityEmployee, uint(employeeID), models.AuditActionUpdate, user.ID, c,
			nil, auditData)
	}

	// Return updated balance
	GetAnnualLeaveBalance(c)
}

// ManualAccrualRequest represents a request to manually add accrual
type ManualAccrualRequest struct {
	Month  string  `json:"month" binding:"required" example:"2025-12"` // YYYY-MM format
	Days   float64 `json:"days" binding:"required" example:"2.0"`
	Reason string  `json:"reason" binding:"required" example:"Manual accrual adjustment"`
}

// AddManualAccrual manually adds an accrual record for an employee
// @Summary Add manual accrual
// @Description Manually add an accrual record for a specific month (Admin only)
// @Tags HR - Leave Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param request body ManualAccrualRequest true "Manual accrual data"
// @Success 201 {object} models.LeaveAccrual
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/hr/employees/{id}/annual-leave-balance/accrual [post]
func AddManualAccrual(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req ManualAccrualRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse month
	monthStart, err := time.Parse("2006-01", req.Month)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid month format. Use YYYY-MM"})
		return
	}
	monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Get Annual leave type
	var annualLeaveType models.LeaveType
	if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
		return
	}

	// Check if accrual already exists
	var existing models.LeaveAccrual
	if err := database.DB.Where("employee_id = ? AND leave_type_id = ? AND accrual_month = ?",
		employeeID, annualLeaveType.ID, monthStart).First(&existing).Error; err == nil {
		// Update existing
		existing.DaysAccrued += req.Days
		existing.DaysBalance += req.Days
		notes := fmt.Sprintf("Manual accrual added: +%.2f days. Reason: %s", req.Days, req.Reason)
		if existing.Notes != nil && *existing.Notes != "" {
			notes = *existing.Notes + "\n" + notes
		}
		existing.Notes = &notes
		now := time.Now()
		existing.ProcessedAt = &now
		existing.IsProcessed = true

		if err := database.DB.Save(&existing).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update accrual"})
			return
		}
		c.JSON(http.StatusOK, existing)
		return
	}

	// Get previous month's balance
	prevMonth := monthStart.AddDate(0, -1, 0)
	var prevAccrual models.LeaveAccrual
	prevBalance := 0.0
	database.DB.Where("employee_id = ? AND leave_type_id = ? AND accrual_month = ?",
		employeeID, annualLeaveType.ID, prevMonth).First(&prevAccrual)
	if prevAccrual.ID > 0 {
		prevBalance = prevAccrual.DaysBalance
	}

	// Calculate days used in this month
	daysUsed := utils.CalculateDaysUsedInMonth(uint(employeeID), annualLeaveType.ID, monthStart)

	// Create new accrual
	now := time.Now()
	accrual := models.LeaveAccrual{
		EmployeeID:   uint(employeeID),
		LeaveTypeID:  annualLeaveType.ID,
		AccrualMonth: &monthStart,
		DaysAccrued:  req.Days,
		DaysUsed:     daysUsed,
		DaysBalance:  prevBalance + req.Days - daysUsed,
		IsProcessed:  true,
		ProcessedAt:  &now,
		Notes:        &req.Reason,
	}

	if err := database.DB.Create(&accrual).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create accrual"})
		return
	}

	// Create audit log
	user := getCurrentUser(c)
	if user != nil {
		createAuditLog(models.AuditEntityEmployee, uint(employeeID), models.AuditActionCreate, user.ID, c,
			nil, map[string]interface{}{"accrual": req.Days, "month": req.Month, "reason": req.Reason})
	}

	c.JSON(http.StatusCreated, accrual)
}

// BulkAccrualRequest represents a request to add multiple accruals at once
type BulkAccrualRequest struct {
	Accruals []ManualAccrualRequest `json:"accruals" binding:"required" example:"[{\"month\":\"2024-01\",\"days\":2.0,\"reason\":\"Historical accrual\"}]"`
}

// BulkAccrualResponse represents the response from bulk accrual operation
type BulkAccrualResponse struct {
	TotalRequested int                     `json:"total_requested"`
	SuccessCount   int                     `json:"success_count"`
	ErrorCount     int                     `json:"error_count"`
	Results        []BulkAccrualItemResult `json:"results"`
}

// BulkAccrualItemResult represents the result of a single accrual in bulk operation
type BulkAccrualItemResult struct {
	Month   string               `json:"month"`
	Success bool                 `json:"success"`
	Message string               `json:"message,omitempty"`
	Accrual *models.LeaveAccrual `json:"accrual,omitempty"`
}

// BulkAddManualAccruals adds multiple accrual records for an employee at once
// @Summary Bulk add manual accruals
// @Description Add multiple accrual records for an employee at once (useful for onboarding existing employees)
// @Tags HR - Leave Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param request body BulkAccrualRequest true "Bulk accrual data"
// @Success 200 {object} BulkAccrualResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/hr/employees/{id}/annual-leave-balance/accruals/bulk [post]
func BulkAddManualAccruals(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req BulkAccrualRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Accruals) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one accrual must be provided"})
		return
	}

	// Get Annual leave type
	var annualLeaveType models.LeaveType
	if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
		return
	}

	response := BulkAccrualResponse{
		TotalRequested: len(req.Accruals),
		SuccessCount:   0,
		ErrorCount:     0,
		Results:        make([]BulkAccrualItemResult, 0, len(req.Accruals)),
	}

	// Process each accrual
	for _, accrualReq := range req.Accruals {
		result := BulkAccrualItemResult{
			Month: accrualReq.Month,
		}

		// Parse month
		monthStart, err := time.Parse("2006-01", accrualReq.Month)
		if err != nil {
			result.Success = false
			result.Message = "Invalid month format. Use YYYY-MM"
			response.ErrorCount++
			response.Results = append(response.Results, result)
			continue
		}
		monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, time.UTC)

		// Check if accrual already exists
		var existing models.LeaveAccrual
		if err := database.DB.Where("employee_id = ? AND leave_type_id = ? AND accrual_month = ?",
			employeeID, annualLeaveType.ID, monthStart).First(&existing).Error; err == nil {
			// Update existing
			existing.DaysAccrued += accrualReq.Days
			existing.DaysBalance += accrualReq.Days
			notes := fmt.Sprintf("Manual accrual added: +%.2f days. Reason: %s", accrualReq.Days, accrualReq.Reason)
			if existing.Notes != nil && *existing.Notes != "" {
				notes = *existing.Notes + "\n" + notes
			}
			existing.Notes = &notes
			now := time.Now()
			existing.ProcessedAt = &now
			existing.IsProcessed = true

			if err := database.DB.Save(&existing).Error; err != nil {
				result.Success = false
				result.Message = "Failed to update existing accrual"
				response.ErrorCount++
				response.Results = append(response.Results, result)
				continue
			}

			result.Success = true
			result.Message = "Updated existing accrual"
			result.Accrual = &existing
			response.SuccessCount++
			response.Results = append(response.Results, result)
			continue
		}

		// Get previous month's balance
		prevMonth := monthStart.AddDate(0, -1, 0)
		var prevAccrual models.LeaveAccrual
		prevBalance := 0.0
		database.DB.Where("employee_id = ? AND leave_type_id = ? AND accrual_month = ?",
			employeeID, annualLeaveType.ID, prevMonth).First(&prevAccrual)
		if prevAccrual.ID > 0 {
			prevBalance = prevAccrual.DaysBalance
		}

		// Calculate days used in this month
		daysUsed := utils.CalculateDaysUsedInMonth(uint(employeeID), annualLeaveType.ID, monthStart)

		// Create new accrual
		now := time.Now()
		accrual := models.LeaveAccrual{
			EmployeeID:   uint(employeeID),
			LeaveTypeID:  annualLeaveType.ID,
			AccrualMonth: &monthStart,
			DaysAccrued:  accrualReq.Days,
			DaysUsed:     daysUsed,
			DaysBalance:  prevBalance + accrualReq.Days - daysUsed,
			IsProcessed:  true,
			ProcessedAt:  &now,
			Notes:        &accrualReq.Reason,
		}

		if err := database.DB.Create(&accrual).Error; err != nil {
			result.Success = false
			result.Message = "Failed to create accrual"
			response.ErrorCount++
			response.Results = append(response.Results, result)
			continue
		}

		result.Success = true
		result.Message = "Accrual created successfully"
		result.Accrual = &accrual
		response.SuccessCount++
		response.Results = append(response.Results, result)
	}

	// Create audit log for bulk operation
	user := getCurrentUser(c)
	if user != nil {
		createAuditLog(models.AuditEntityEmployee, uint(employeeID), models.AuditActionCreate, user.ID, c,
			nil, map[string]interface{}{
				"bulk_accruals": len(req.Accruals),
				"success_count": response.SuccessCount,
				"error_count":   response.ErrorCount,
			})
	}

	c.JSON(http.StatusOK, response)
}

// GetAllEmployeesLeaveBalances gets annual leave balances for all employees
// @Summary Get all employees leave balances
// @Description Get annual leave balances for all employees with filtering options (HR/Admin only)
// @Tags HR - Leave Management
// @Produce json
// @Security BearerAuth
// @Param department query string false "Filter by department"
// @Param status query string false "Filter by employment status (active, on_leave, etc.)"
// @Success 200 {array} AnnualLeaveBalanceResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/hr/employees/annual-leave-balances [get]
func GetAllEmployeesLeaveBalances(c *gin.Context) {
	department := c.Query("department")
	status := c.Query("status")

	// Get Annual leave type
	var annualLeaveType models.LeaveType
	if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
		return
	}

	// Build query - exclude admin users
	query := database.DB.Model(&models.Employee{}).Where("role != ?", models.RoleAdmin)
	if department != "" {
		query = query.Where("department = ?", department)
	}

	var employees []models.Employee
	query.Find(&employees)

	// If status filter, need to check employment details
	if status != "" {
		var filteredEmployees []models.Employee
		for _, emp := range employees {
			var employment models.EmploymentDetails
			if err := database.DB.Where("employee_id = ?", emp.ID).First(&employment).Error; err == nil {
				if string(employment.EmploymentStatus) == status {
					filteredEmployees = append(filteredEmployees, emp)
				}
			} else if status == "active" {
				// If no employment record, assume active
				filteredEmployees = append(filteredEmployees, emp)
			}
		}
		employees = filteredEmployees
	}

	balances := make([]AnnualLeaveBalanceResponse, 0, len(employees))

	for _, emp := range employees {
		// Ensure accruals are up to date
		utils.EnsureAccrualsUpToDate(emp.ID, annualLeaveType.ID)

		// Get all accruals
		// Order by accrual_month if available, otherwise by year and month
		var accruals []models.LeaveAccrual
		database.DB.Where("employee_id = ? AND leave_type_id = ?", emp.ID, annualLeaveType.ID).
			Order("COALESCE(accrual_month, MAKE_DATE(year::integer, month::integer, 1)) DESC, year DESC, month DESC").
			Find(&accruals)

		// Get employee start date to exclude first month accruals
		var employeeStartDate time.Time
		var employment models.EmploymentDetails
		if err := database.DB.Where("employee_id = ?", emp.ID).First(&employment).Error; err == nil {
			if employment.HireDate != nil {
				employeeStartDate = *employment.HireDate
			} else if employment.StartDate != nil {
				employeeStartDate = *employment.StartDate
			} else {
				employeeStartDate = emp.CreatedAt
			}
		} else {
			employeeStartDate = emp.CreatedAt
		}
		firstMonthStart := time.Date(employeeStartDate.Year(), employeeStartDate.Month(), 1, 0, 0, 0, 0, time.UTC)

		var totalAccrued float64
		accrualResponses := make([]LeaveAccrualResponse, 0, len(accruals))

		for _, acc := range accruals {
			// Get accrual month for comparison
			var accrualMonth time.Time
			if acc.AccrualMonth != nil {
				accrualMonth = *acc.AccrualMonth
			} else if acc.Year > 0 && acc.Month > 0 {
				accrualMonth = time.Date(acc.Year, time.Month(acc.Month), 1, 0, 0, 0, 0, time.UTC)
			}

			// Skip regular accruals in the first month of employment
			// BUT include initial balance adjustments (identified by Notes containing "Initial balance" or "set-initial")
			isInitialBalance := acc.Notes != nil && 
				(*acc.Notes != "" && (strings.Contains(*acc.Notes, "Initial balance") || 
				 strings.Contains(*acc.Notes, "set-initial") || 
				 strings.Contains(*acc.Notes, "Set initial")))
			
			if !accrualMonth.IsZero() && accrualMonth.Equal(firstMonthStart) && !isInitialBalance {
				continue
			}

			totalAccrued += acc.DaysAccrued

			processedAtStr := ""
			if acc.ProcessedAt != nil {
				processedAtStr = acc.ProcessedAt.Format(time.RFC3339)
			}

			accrualResponses = append(accrualResponses, LeaveAccrualResponse{
				Month:       acc.GetAccrualMonthKey(),
				DaysAccrued: acc.DaysAccrued,
				DaysUsed:    acc.DaysUsed,
				DaysBalance: acc.DaysBalance,
				IsProcessed: acc.IsProcessed,
				ProcessedAt: &processedAtStr,
			})
		}

		// Calculate total used directly from approved leave records (source of truth)
		// This ensures accuracy even if accrual records have incorrect DaysUsed values
		var totalUsed float64
		var approvedLeaves []models.Leave
		database.DB.Where("employee_id = ? AND leave_type_id = ? AND status = ?",
			emp.ID, annualLeaveType.ID, models.StatusApproved).Find(&approvedLeaves)
		for _, leave := range approvedLeaves {
			totalUsed += float64(leave.GetDuration())
		}

		// Get carry-over balance
		var carryOverBalance float64
		if annualLeaveType.AllowCarryOver {
			carryOverBalance, _ = utils.GetCarryOverBalance(emp.ID, annualLeaveType.ID)
		}

		// Get total current balance (accrual + carry-over) - this is what's actually available
		currentBalance, _ := utils.GetCurrentLeaveBalance(emp.ID, annualLeaveType.ID)

		// Calculate all-time net balance using actual accrual records (includes initial balance adjustments)
		// This reflects the actual accrued amount including any manual adjustments from onboarding
		// AllTimeNetBalance = Total Accrued (from records) - Total Used (from approved leaves)
		allTimeNetBalance := totalAccrued - totalUsed
		// Don't set to 0 if negative - negative values are valid (overdrawn)

		// Get pending and upcoming leaves
		var pendingLeaves, upcomingLeaves int64
		now := time.Now()
		database.DB.Model(&models.Leave{}).
			Where("employee_id = ? AND leave_type_id = ? AND status = ?", emp.ID, annualLeaveType.ID, models.StatusPending).
			Count(&pendingLeaves)
		database.DB.Model(&models.Leave{}).
			Where("employee_id = ? AND leave_type_id = ? AND status = ? AND start_date > ?",
				emp.ID, annualLeaveType.ID, models.StatusApproved, now).
			Count(&upcomingLeaves)

		balances = append(balances, AnnualLeaveBalanceResponse{
			EmployeeID:        emp.ID,
			EmployeeName:      emp.Firstname + " " + emp.Lastname,
			TotalAccrued:      totalAccrued,
			TotalUsed:         totalUsed,
			AllTimeNetBalance: allTimeNetBalance,
			CurrentBalance:    currentBalance,
			CarryOverBalance:  carryOverBalance,
			Accruals:          accrualResponses,
			PendingLeaves:     int(pendingLeaves),
			UpcomingLeaves:    int(upcomingLeaves),
		})
	}

	c.JSON(http.StatusOK, balances)
}

// ExportAnnualLeaveBalances exports annual leave balances to Excel or PDF
// @Summary Export annual leave balances
// @Description Export annual leave balances for all employees to Excel or PDF format (Admin only)
// @Tags HR - Leave Management
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,application/pdf
// @Security BearerAuth
// @Param format query string true "Export format (excel or pdf)" Enums(excel, pdf) default:"excel"
// @Param department query string false "Filter by department"
// @Param status query string false "Filter by employment status"
// @Success 200 {file} file "Excel or PDF file"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/hr/employees/annual-leave-balances/export [get]
func ExportAnnualLeaveBalances(c *gin.Context) {
	format := c.Query("format")
	if format == "" {
		format = "excel"
	}
	if format != "excel" && format != "pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format. Use 'excel' or 'pdf'"})
		return
	}

	department := c.Query("department")
	status := c.Query("status")

	// Get Annual leave type
	var annualLeaveType models.LeaveType
	if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
		return
	}

	// Build query (same logic as GetAllEmployeesLeaveBalances) - exclude admin users
	query := database.DB.Model(&models.Employee{}).Where("role != ?", models.RoleAdmin)
	if department != "" {
		query = query.Where("department = ?", department)
	}

	var employees []models.Employee
	query.Find(&employees)

	// If status filter, need to check employment details
	if status != "" {
		var filteredEmployees []models.Employee
		for _, emp := range employees {
			var employment models.EmploymentDetails
			if err := database.DB.Where("employee_id = ?", emp.ID).First(&employment).Error; err == nil {
				if string(employment.EmploymentStatus) == status {
					filteredEmployees = append(filteredEmployees, emp)
				}
			} else if status == "active" {
				filteredEmployees = append(filteredEmployees, emp)
			}
		}
		employees = filteredEmployees
	}

	// Get balances (same logic as GetAllEmployeesLeaveBalances)
	balances := make([]AnnualLeaveBalanceResponse, 0, len(employees))

	for _, emp := range employees {
		utils.EnsureAccrualsUpToDate(emp.ID, annualLeaveType.ID)

		var accruals []models.LeaveAccrual
		database.DB.Where("employee_id = ? AND leave_type_id = ?", emp.ID, annualLeaveType.ID).
			Order("accrual_month DESC").
			Find(&accruals)

		// Get employee start date to exclude first month accruals
		var employeeStartDate time.Time
		var employment models.EmploymentDetails
		if err := database.DB.Where("employee_id = ?", emp.ID).First(&employment).Error; err == nil {
			if employment.HireDate != nil {
				employeeStartDate = *employment.HireDate
			} else if employment.StartDate != nil {
				employeeStartDate = *employment.StartDate
			} else {
				employeeStartDate = emp.CreatedAt
			}
		} else {
			employeeStartDate = emp.CreatedAt
		}
		firstMonthStart := time.Date(employeeStartDate.Year(), employeeStartDate.Month(), 1, 0, 0, 0, 0, time.UTC)

		var totalAccrued float64
		for _, acc := range accruals {
			// Get accrual month for comparison
			var accrualMonth time.Time
			if acc.AccrualMonth != nil {
				accrualMonth = *acc.AccrualMonth
			} else if acc.Year > 0 && acc.Month > 0 {
				accrualMonth = time.Date(acc.Year, time.Month(acc.Month), 1, 0, 0, 0, 0, time.UTC)
			}

			// Skip regular accruals in the first month of employment
			// BUT include initial balance adjustments (identified by Notes containing "Initial balance" or "set-initial")
			isInitialBalance := acc.Notes != nil && 
				(*acc.Notes != "" && (strings.Contains(*acc.Notes, "Initial balance") || 
				 strings.Contains(*acc.Notes, "set-initial") || 
				 strings.Contains(*acc.Notes, "Set initial")))
			
			if !accrualMonth.IsZero() && accrualMonth.Equal(firstMonthStart) && !isInitialBalance {
				continue
			}

			totalAccrued += acc.DaysAccrued
		}

		// Calculate total used directly from approved leave records (source of truth)
		// This ensures accuracy even if accrual records have incorrect DaysUsed values
		var totalUsed float64
		var approvedLeaves []models.Leave
		database.DB.Where("employee_id = ? AND leave_type_id = ? AND status = ?",
			emp.ID, annualLeaveType.ID, models.StatusApproved).Find(&approvedLeaves)
		for _, leave := range approvedLeaves {
			totalUsed += float64(leave.GetDuration())
		}

		// Get carry-over balance
		var carryOverBalance float64
		if annualLeaveType.AllowCarryOver {
			carryOverBalance, _ = utils.GetCarryOverBalance(emp.ID, annualLeaveType.ID)
		}

		// Get total current balance (accrual + carry-over) - this is what's actually available
		currentBalance, _ := utils.GetCurrentLeaveBalance(emp.ID, annualLeaveType.ID)

		// Calculate all-time net balance using actual accrual records (includes initial balance adjustments)
		// This reflects the actual accrued amount including any manual adjustments from onboarding
		// AllTimeNetBalance = Total Accrued (from records) - Total Used (from approved leaves)
		allTimeNetBalance := totalAccrued - totalUsed
		// Don't set to 0 if negative - negative values are valid (overdrawn)

		var pendingLeaves, upcomingLeaves int64
		now := time.Now()
		database.DB.Model(&models.Leave{}).
			Where("employee_id = ? AND leave_type_id = ? AND status = ?", emp.ID, annualLeaveType.ID, models.StatusPending).
			Count(&pendingLeaves)
		database.DB.Model(&models.Leave{}).
			Where("employee_id = ? AND leave_type_id = ? AND status = ? AND start_date > ?",
				emp.ID, annualLeaveType.ID, models.StatusApproved, now).
			Count(&upcomingLeaves)

		balances = append(balances, AnnualLeaveBalanceResponse{
			EmployeeID:        emp.ID,
			EmployeeName:      emp.Firstname + " " + emp.Lastname,
			TotalAccrued:      totalAccrued,
			TotalUsed:         totalUsed,
			AllTimeNetBalance: allTimeNetBalance,
			CurrentBalance:    currentBalance,
			CarryOverBalance:  carryOverBalance,
			PendingLeaves:     int(pendingLeaves),
			UpcomingLeaves:    int(upcomingLeaves),
		})
	}

	// Convert to export format
	exportData := make([]utils.EmployeeBalanceData, 0, len(balances))
	for _, balance := range balances {
		// Get employee department
		var employee models.Employee
		database.DB.First(&employee, balance.EmployeeID)

		exportData = append(exportData, utils.EmployeeBalanceData{
			EmployeeID:     balance.EmployeeID,
			EmployeeName:   balance.EmployeeName,
			Department:     employee.Department,
			TotalAccrued:   balance.TotalAccrued,
			TotalUsed:      balance.TotalUsed,
			CurrentBalance: balance.CurrentBalance,
			PendingLeaves:  balance.PendingLeaves,
			UpcomingLeaves: balance.UpcomingLeaves,
		})
	}

	preparedData := utils.PrepareBalancesForExport(exportData)

	// Generate file based on format
	var fileData []byte
	var filename string
	var contentType string
	var err error

	if format == "excel" {
		fileData, err = utils.ExportAnnualLeaveBalancesToExcel(preparedData)
		filename = fmt.Sprintf("annual_leave_balances_%s.xlsx", time.Now().Format("20060102_150405"))
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	} else {
		fileData, err = utils.ExportAnnualLeaveBalancesToPDF(preparedData)
		filename = fmt.Sprintf("annual_leave_balances_%s.pdf", time.Now().Format("20060102_150405"))
		contentType = "application/pdf"
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate export file"})
		return
	}

	// Set headers for file download
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", contentType)
	c.Data(http.StatusOK, contentType, fileData)
}

// ExportEmployeeAnnualLeave exports single employee annual leave report to Excel or PDF
// @Summary Export employee annual leave report
// @Description Export annual leave report for a specific employee to Excel or PDF format (HR/Admin only)
// @Tags HR - Leave Management
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param format query string true "Export format (excel or pdf)" Enums(excel, pdf) default:"excel"
// @Success 200 {file} file "Excel or PDF file"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/hr/employees/{id}/annual-leave-balance/export [get]
func ExportEmployeeAnnualLeave(c *gin.Context) {
	employeeID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid employee ID"})
		return
	}

	format := c.Query("format")
	if format == "" {
		format = "excel"
	}
	if format != "excel" && format != "pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format. Use 'excel' or 'pdf'"})
		return
	}

	// Get employee
	var employee models.Employee
	if err := database.DB.First(&employee, employeeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Employee not found"})
		return
	}

	// Get Annual leave type
	var annualLeaveType models.LeaveType
	if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
		return
	}

	// Ensure accruals are up to date
	if err := utils.EnsureAccrualsUpToDate(uint(employeeID), annualLeaveType.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process accruals"})
		return
	}

	// Get all accruals
	var accruals []models.LeaveAccrual
	database.DB.Where("employee_id = ? AND leave_type_id = ?", employeeID, annualLeaveType.ID).
		Order("COALESCE(accrual_month, MAKE_DATE(year::integer, month::integer, 1)) ASC, year ASC, month ASC").
		Find(&accruals)

	// Get employee start date to exclude first month accruals
	var employeeStartDate time.Time
	var employment models.EmploymentDetails
	if err := database.DB.Where("employee_id = ?", employeeID).First(&employment).Error; err == nil {
		if employment.HireDate != nil {
			employeeStartDate = *employment.HireDate
		} else if employment.StartDate != nil {
			employeeStartDate = *employment.StartDate
		} else {
			employeeStartDate = employee.CreatedAt
		}
	} else {
		employeeStartDate = employee.CreatedAt
	}
	firstMonthStart := time.Date(employeeStartDate.Year(), employeeStartDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Calculate totals and prepare accrual data
	var totalAccrued float64
	accrualExports := make([]utils.AccrualExport, 0, len(accruals))

	for _, acc := range accruals {
		// Get accrual month for comparison
		var accrualMonth time.Time
		if acc.AccrualMonth != nil {
			accrualMonth = *acc.AccrualMonth
		} else if acc.Year > 0 && acc.Month > 0 {
			accrualMonth = time.Date(acc.Year, time.Month(acc.Month), 1, 0, 0, 0, 0, time.UTC)
		}

		// Skip regular accruals in the first month of employment
		// BUT include initial balance adjustments (identified by Notes containing "Initial balance" or "set-initial")
		isInitialBalance := acc.Notes != nil && 
			(*acc.Notes != "" && (strings.Contains(*acc.Notes, "Initial balance") || 
			 strings.Contains(*acc.Notes, "set-initial") || 
			 strings.Contains(*acc.Notes, "Set initial")))
		
		if !accrualMonth.IsZero() && accrualMonth.Equal(firstMonthStart) && !isInitialBalance {
			continue
		}

		totalAccrued += acc.DaysAccrued

		processedAtStr := ""
		if acc.ProcessedAt != nil {
			processedAtStr = acc.ProcessedAt.Format("2006-01-02 15:04:05")
		}

		accrualExports = append(accrualExports, utils.AccrualExport{
			Month:       acc.GetAccrualMonthKey(),
			DaysAccrued: acc.DaysAccrued,
			DaysUsed:    acc.DaysUsed,
			DaysBalance: acc.DaysBalance,
			IsProcessed: acc.IsProcessed,
			ProcessedAt: processedAtStr,
		})
	}

	// Calculate total used from approved leaves
	var totalUsed float64
	var approvedLeaves []models.Leave
	database.DB.Where("employee_id = ? AND leave_type_id = ? AND status = ?",
		employeeID, annualLeaveType.ID, models.StatusApproved).
		Order("start_date DESC").
		Find(&approvedLeaves)

	leaveExports := make([]utils.LeaveExport, 0, len(approvedLeaves))
	for _, leave := range approvedLeaves {
		totalUsed += float64(leave.GetDuration())
		leaveExports = append(leaveExports, utils.LeaveExport{
			StartDate: leave.StartDate.Format("2006-01-02"),
			EndDate:   leave.EndDate.Format("2006-01-02"),
			Duration:  float64(leave.GetDuration()),
			Reason:    leave.Reason,
		})
	}

	// Get carry-over balance
	var carryOverBalance float64
	if annualLeaveType.AllowCarryOver {
		carryOverBalance, _ = utils.GetCarryOverBalance(uint(employeeID), annualLeaveType.ID)
	}

	// Get current balance
	currentBalance, _ := utils.GetCurrentLeaveBalance(uint(employeeID), annualLeaveType.ID)

	// Calculate all-time net balance using actual accrual records (includes initial balance adjustments)
	// This reflects the actual accrued amount including any manual adjustments from onboarding
	// AllTimeNetBalance = Total Accrued (from records) - Total Used (from approved leaves)
	allTimeNetBalance := totalAccrued - totalUsed

	// Prepare report data
	report := utils.EmployeeAnnualLeaveReport{
		EmployeeID:        uint(employeeID),
		EmployeeName:      employee.Firstname + " " + employee.Lastname,
		Department:        employee.Department,
		TotalAccrued:      totalAccrued,
		TotalUsed:         totalUsed,
		CurrentBalance:    currentBalance,
		CarryOverBalance:  carryOverBalance,
		AllTimeNetBalance: allTimeNetBalance,
		Accruals:          accrualExports,
		ApprovedLeaves:    leaveExports,
	}

	// Generate file based on format
	var fileData []byte
	var filename string
	var contentType string

	if format == "excel" {
		fileData, err = utils.ExportEmployeeAnnualLeaveToExcel(report)
		filename = fmt.Sprintf("annual_leave_report_%s_%d.xlsx", strings.ReplaceAll(report.EmployeeName, " ", "_"), employeeID)
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	} else {
		fileData, err = utils.ExportEmployeeAnnualLeaveToPDF(report)
		filename = fmt.Sprintf("annual_leave_report_%s_%d.pdf", strings.ReplaceAll(report.EmployeeName, " ", "_"), employeeID)
		contentType = "application/pdf"
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate export file"})
		return
	}

	// Set headers for file download
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", contentType)
	c.Data(http.StatusOK, contentType, fileData)
}

// MonthlyLeaveReportResponse represents monthly leave report data
type MonthlyLeaveReportResponse struct {
	Number     int     `json:"number"`
	Name       string  `json:"name"`
	Position   string  `json:"position"`
	Opening    float64 `json:"opening"`
	DaysEarned float64 `json:"days_earned"`
	Total      float64 `json:"total"`
	DaysTaken  float64 `json:"days_taken"`
	Net        float64 `json:"net"`
}

// GetMonthlyLeaveReport gets monthly leave report for a specific month
// @Summary Get monthly leave report
// @Description Get monthly leave report in CSV format matching the legacy system (HR/Admin only)
// @Tags HR - Leave Management
// @Produce json
// @Security BearerAuth
// @Param month query string true "Month in YYYY-MM format (e.g., 2025-02)"
// @Success 200 {array} MonthlyLeaveReportResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/hr/leaves/monthly-report [get]
func GetMonthlyLeaveReport(c *gin.Context) {
	monthStr := c.Query("month")
	if monthStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Month parameter is required (format: YYYY-MM)"})
		return
	}

	month, err := time.Parse("2006-01", monthStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid month format. Use YYYY-MM (e.g., 2025-02)"})
		return
	}

	// Get Annual leave type
	var annualLeaveType models.LeaveType
	if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
		return
	}

	// Generate monthly report
	reportData, err := utils.GetMonthlyLeaveReport(month, annualLeaveType.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate monthly report"})
		return
	}

	// Convert to response format
	response := make([]MonthlyLeaveReportResponse, 0, len(reportData))
	for _, data := range reportData {
		response = append(response, MonthlyLeaveReportResponse{
			Number:     data.Number,
			Name:       data.Name,
			Position:   data.Position,
			Opening:    data.Opening,
			DaysEarned: data.DaysEarned,
			Total:      data.Total,
			DaysTaken:  data.DaysTaken,
			Net:        data.Net,
		})
	}

	c.JSON(http.StatusOK, response)
}

// ExportMonthlyLeaveReport exports monthly leave report to Excel format
// @Summary Export monthly leave report
// @Description Export monthly leave report to Excel format matching CSV structure (HR/Admin only)
// @Tags HR - Leave Management
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security BearerAuth
// @Param month query string true "Month in YYYY-MM format (e.g., 2025-02)"
// @Param organization query string false "Organization name (default: 'CHUDLEIGH HOUSE SCHOOL')"
// @Success 200 {file} file "Excel file"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/hr/leaves/monthly-report/export [get]
func ExportMonthlyLeaveReport(c *gin.Context) {
	monthStr := c.Query("month")
	if monthStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Month parameter is required (format: YYYY-MM)"})
		return
	}

	month, err := time.Parse("2006-01", monthStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid month format. Use YYYY-MM (e.g., 2025-02)"})
		return
	}

	organizationName := c.Query("organization")
	if organizationName == "" {
		organizationName = "CHUDLEIGH HOUSE SCHOOL"
	}

	// Get Annual leave type
	var annualLeaveType models.LeaveType
	if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
		return
	}

	// Generate monthly report
	reportData, err := utils.GetMonthlyLeaveReport(month, annualLeaveType.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate monthly report"})
		return
	}

	// Export to Excel
	fileData, err := utils.ExportMonthlyLeaveReportToExcel(reportData, month, organizationName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate export file"})
		return
	}

	// Set headers for file download
	filename := fmt.Sprintf("leave_days_%s.xlsx", month.Format("200601"))
	contentType := "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", contentType)
	c.Data(http.StatusOK, contentType, fileData)
}

// ProcessYearEndCarryOverRequest represents a request to process year-end carry-over
type ProcessYearEndCarryOverRequest struct {
	LeaveTypeID uint `json:"leave_type_id" binding:"required" example:"1"`
	FromYear    int  `json:"from_year" binding:"required" example:"2024"`
}

// ProcessYearEndCarryOver processes carry-over for all employees at year-end
// @Summary Process year-end carry-over
// @Description Process carry-over for all employees for a specific year (HR/Admin only)
// @Tags HR - Leave Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ProcessYearEndCarryOverRequest true "Carry-over processing request"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/hr/leaves/process-carryover [post]
func ProcessYearEndCarryOver(c *gin.Context) {
	var req ProcessYearEndCarryOverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate year
	if req.FromYear < 2000 || req.FromYear > 2100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid year"})
		return
	}

	// Get leave type
	var leaveType models.LeaveType
	if err := database.DB.First(&leaveType, req.LeaveTypeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Leave type not found"})
		return
	}

	if !leaveType.AllowCarryOver {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Carry-over is not enabled for this leave type"})
		return
	}

	// Get current user
	userID, _ := c.Get("user_id")
	processedBy := userID.(uint)

	// Process carry-over for all employees
	processed, skipped, errors := utils.ProcessCarryOverForAllEmployees(req.LeaveTypeID, req.FromYear, &processedBy)

	response := gin.H{
		"message":   "Carry-over processing completed",
		"from_year": req.FromYear,
		"processed": processed,
		"skipped":   skipped,
	}

	if len(errors) > 0 {
		response["errors"] = errors
		response["error_count"] = len(errors)
	}

	c.JSON(http.StatusOK, response)
}

// GetCarryOverHistory gets carry-over history for an employee
// @Summary Get carry-over history
// @Description Get carry-over history for an employee (HR/Admin only)
// @Tags HR - Leave Management
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param leave_type_id query int false "Leave type ID (defaults to Annual leave)"
// @Success 200 {array} models.LeaveCarryOver
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/hr/employees/{id}/carryover-history [get]
func GetCarryOverHistory(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

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
		if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
			return
		}
		leaveTypeID = annualLeaveType.ID
	}

	carryOvers, err := utils.GetCarryOverHistory(uint(employeeID), leaveTypeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch carry-over history"})
		return
	}

	c.JSON(http.StatusOK, carryOvers)
}

// GetCarryOverBalance gets current carry-over balance for an employee
// @Summary Get carry-over balance
// @Description Get current carry-over balance for an employee (HR/Admin only)
// @Tags HR - Leave Management
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param leave_type_id query int false "Leave type ID (defaults to Annual leave)"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/hr/employees/{id}/carryover-balance [get]
func GetCarryOverBalance(c *gin.Context) {
	employeeID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

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
		if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
			return
		}
		leaveTypeID = annualLeaveType.ID
	}

	balance, err := utils.GetCarryOverBalance(uint(employeeID), leaveTypeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate carry-over balance"})
		return
	}

	// Get detailed carry-over records
	carryOvers, err := utils.GetCarryOverHistory(uint(employeeID), leaveTypeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch carry-over details"})
		return
	}

	var activeCarryOvers []models.LeaveCarryOver
	var expiredCarryOvers []models.LeaveCarryOver
	now := time.Now()

	for _, co := range carryOvers {
		if co.IsExpired || (co.ExpiryDate != nil && co.ExpiryDate.Before(now)) {
			expiredCarryOvers = append(expiredCarryOvers, co)
		} else {
			activeCarryOvers = append(activeCarryOvers, co)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_balance":      balance,
		"active_carryovers":  activeCarryOvers,
		"expired_carryovers": expiredCarryOvers,
	})
}

// ExpireCarryOvers manually expires carry-overs that have passed their expiry date
// @Summary Expire carry-overs
// @Description Manually expire carry-overs that have passed their expiry date (HR/Admin only)
// @Tags HR - Leave Management
// @Produce json
// @Security BearerAuth
// @Success 200 {object} MessageResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/hr/leaves/expire-carryovers [post]
func ExpireCarryOvers(c *gin.Context) {
	if err := utils.ExpireCarryOvers(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to expire carry-overs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Carry-overs expired successfully"})
}

// BulkImportLeaveBalancesRequest represents a request to import leave balances from CSV
type BulkImportLeaveBalancesRequest struct {
	Month    string `json:"month,omitempty" example:"2022-02"`   // Optional: Override month from CSV (YYYY-MM format)
	ResetAll bool   `json:"reset_all,omitempty" example:"false"` // Optional: Delete all existing accruals before import
}

// BulkImportLeaveBalancesResponse represents the response from bulk import
type BulkImportLeaveBalancesResponse struct {
	Total   int            `json:"total"`
	Success int            `json:"success"`
	Failed  int            `json:"failed"`
	Results []ImportResult `json:"results"`
	Month   string         `json:"month"`
}

// ImportResult represents the result of importing a single employee's balance
type ImportResult struct {
	EmployeeName string  `json:"employee_name"`
	Success      bool    `json:"success"`
	Error        string  `json:"error,omitempty"`
	Balance      float64 `json:"balance,omitempty"`
}

// BulkImportLeaveBalances imports leave balances from CSV file (matching the legacy format)
// @Summary Bulk import leave balances from CSV
// @Description Import leave balances for multiple employees from CSV file matching the legacy system format (Admin only)
// @Tags HR - Leave Management
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "CSV file with leave balance data"
// @Param month formData string false "Override month from CSV (YYYY-MM format)"
// @Param reset_all formData bool false "Delete all existing accruals before import"
// @Success 200 {object} BulkImportLeaveBalancesResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/hr/leave-balances/import [post]
func BulkImportLeaveBalances(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Get optional parameters
	monthOverride := c.PostForm("month")
	resetAll := c.PostForm("reset_all") == "true"

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	// Read and skip empty lines until we find the month line
	var monthStr string

	for {
		record, err := reader.Read()
		if err == io.EOF {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CSV format: could not find month or header row"})
			return
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read CSV file: " + err.Error()})
			return
		}

		lineText := strings.Join(record, " ")

		// Look for month line (e.g., "FOR THE MONTH OF FEBRUARY 2022")
		if strings.Contains(strings.ToUpper(lineText), "FOR THE MONTH OF") {
			monthStr = extractMonthFromLine(lineText)
			if monthStr == "" && monthOverride == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Could not extract month from CSV. Please provide month parameter."})
				return
			}
			if monthOverride != "" {
				monthStr = monthOverride
			}
		}

		// Look for header row (NAME, POSITION, OPENING, etc.)
		if len(record) >= 4 {
			firstCol := strings.TrimSpace(strings.ToUpper(record[1]))
			if firstCol == "NAME" {
				break
			}
		}
	}

	if monthStr == "" {
		if monthOverride == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Could not determine month from CSV. Please provide month parameter."})
			return
		}
		monthStr = monthOverride
	}

	// Parse month
	monthStart, err := time.Parse("2006-01", monthStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid month format. Use YYYY-MM"})
		return
	}
	monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Get Annual leave type
	var annualLeaveType models.LeaveType
	if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
		return
	}

	// If reset_all, delete all existing accruals
	if resetAll {
		if err := database.DB.Where("leave_type_id = ?", annualLeaveType.ID).Delete(&models.LeaveAccrual{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset accruals: " + err.Error()})
			return
		}
	}

	var results []ImportResult
	total := 0
	success := 0
	failed := 0

	// Read data rows
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			failed++
			results = append(results, ImportResult{
				EmployeeName: "Unknown",
				Success:      false,
				Error:        "Failed to parse row: " + err.Error(),
			})
			continue
		}

		// Skip empty rows or rows without enough columns
		// Need at least 8 columns: index, NAME, POSITION, OPENING, DAYS EARNED, TOTAL, DAYS TAKEN, NET
		if len(record) < 8 {
			continue
		}

		// Extract data (format: index, NAME, POSITION, OPENING, DAYS EARNED, TOTAL, DAYS TAKEN, NET)
		employeeName := strings.TrimSpace(record[1])
		if employeeName == "" || employeeName == "NAME" || strings.ToUpper(employeeName) == "NAME" {
			continue // Skip header or empty rows
		}

		total++

		// Parse numeric values (handle cases where some columns might be missing)
		openingStr := ""
		daysEarnedStr := ""
		totalStr := ""
		daysTakenStr := ""
		netStr := ""

		if len(record) > 3 {
			openingStr = strings.TrimSpace(record[3])
		}
		if len(record) > 4 {
			daysEarnedStr = strings.TrimSpace(record[4])
		}
		if len(record) > 5 {
			totalStr = strings.TrimSpace(record[5])
		}
		if len(record) > 6 {
			daysTakenStr = strings.TrimSpace(record[6])
		}
		if len(record) > 7 {
			netStr = strings.TrimSpace(record[7])
		}

		// Parse opening balance (handle negative values and empty)
		opening := parseFloat(openingStr)
		daysEarned := parseFloat(daysEarnedStr)
		totalDays := parseFloat(totalStr)
		daysTaken := parseFloat(daysTakenStr)
		netBalance := parseFloat(netStr)

		// If net is empty but we have total and days taken, calculate it
		if netBalance == 0 && totalDays > 0 {
			netBalance = totalDays - daysTaken
		}
		// If total is empty but we have opening and days earned, calculate it
		if totalDays == 0 && opening != 0 {
			totalDays = opening + daysEarned
		}

		// Find employee by name
		var employee models.Employee
		nameParts := strings.Fields(employeeName)
		var firstname, lastname string

		if len(nameParts) >= 2 {
			firstname = nameParts[0]
			lastname = strings.Join(nameParts[1:], " ")
		} else {
			// Single name - use as firstname, lastname empty
			firstname = employeeName
			lastname = ""
		}

		// Try to match by firstname and lastname
		err = database.DB.Where("LOWER(firstname) = LOWER(?) AND LOWER(lastname) = LOWER(?)", firstname, lastname).
			First(&employee).Error

		// If not found, try matching by full name in either field
		if err != nil && len(nameParts) >= 2 {
			err = database.DB.Where("LOWER(CONCAT(firstname, ' ', lastname)) = LOWER(?)", employeeName).
				Or("LOWER(firstname) LIKE LOWER(?) OR LOWER(lastname) LIKE LOWER(?)",
					"%"+firstname+"%", "%"+lastname+"%").
				First(&employee).Error
		}

		// If still not found, try single name match
		if err != nil && len(nameParts) == 1 {
			err = database.DB.Where("LOWER(firstname) = LOWER(?) OR LOWER(lastname) = LOWER(?)", employeeName, employeeName).
				First(&employee).Error
		}

		// If employee not found, create them
		if err != nil {
			// Extract position/department from CSV
			position := ""
			if len(record) > 2 {
				position = strings.TrimSpace(record[2])
			}

			// Generate email from name (firstname.lastname@company.com)
			email := generateEmailFromName(firstname, lastname)

			// Check if email already exists, if so, add a number
			var existingEmail models.Employee
			emailCounter := 1
			for database.DB.Where("email = ?", email).First(&existingEmail).Error == nil {
				if lastname != "" {
					email = fmt.Sprintf("%s.%s.%d@company.com", strings.ToLower(firstname), strings.ToLower(lastname), emailCounter)
				} else {
					email = fmt.Sprintf("%s.%d@company.com", strings.ToLower(firstname), emailCounter)
				}
				emailCounter++
				if emailCounter > 1000 {
					// Fallback to timestamp-based email
					email = fmt.Sprintf("%s.%s.%d@company.com", strings.ToLower(firstname), strings.ToLower(lastname), time.Now().Unix())
					break
				}
			}

			// Generate default password (can be changed later)
			defaultPassword := "Welcome123!" // Default password for imported employees
			hashedPassword, err := utils.HashPassword(defaultPassword)
			if err != nil {
				failed++
				results = append(results, ImportResult{
					EmployeeName: employeeName,
					Success:      false,
					Error:        "Failed to create employee: " + err.Error(),
				})
				continue
			}

			// Create employee
			emailPtr := &email
			employee = models.Employee{
				Firstname:    firstname,
				Lastname:     lastname,
				Email:        emailPtr,
				PasswordHash: hashedPassword,
				Department:   position,
				Role:         models.RoleEmployee,
			}

			if err := database.DB.Create(&employee).Error; err != nil {
				failed++
				results = append(results, ImportResult{
					EmployeeName: employeeName,
					Success:      false,
					Error:        "Failed to create employee: " + err.Error(),
				})
				continue
			}

			// Create EmploymentDetails with hire date (use the month from CSV as hire date)
			hireDate := time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, time.UTC)
			employmentDetails := models.EmploymentDetails{
				EmployeeID:       employee.ID,
				EmploymentType:   models.EmploymentTypeFullTime,
				EmploymentStatus: models.EmploymentStatusActive,
				HireDate:         &hireDate,
				StartDate:        &hireDate,
			}

			// Create employment details (don't fail if this fails, employee is already created)
			if err := database.DB.Create(&employmentDetails).Error; err != nil {
				// Log but don't fail - employment details can be added later
				// This is a non-critical error
			}
		}

		// Get or create accrual record for this month
		var accrual models.LeaveAccrual
		err = database.DB.Where("employee_id = ? AND leave_type_id = ? AND accrual_month = ?",
			employee.ID, annualLeaveType.ID, monthStart).First(&accrual).Error

		now := time.Now()
		if err != nil {
			// Create new accrual record
			accrual = models.LeaveAccrual{
				EmployeeID:   employee.ID,
				LeaveTypeID:  annualLeaveType.ID,
				AccrualMonth: &monthStart,
				DaysAccrued:  totalDays,
				DaysUsed:     daysTaken,
				DaysBalance:  netBalance,
				IsProcessed:  true,
				ProcessedAt:  &now,
				Notes:        getStringPtrCSV(fmt.Sprintf("Imported from CSV: Opening=%.2f, Earned=%.2f, Total=%.2f, Taken=%.2f, Net=%.2f", opening, daysEarned, totalDays, daysTaken, netBalance)),
			}

			if err := database.DB.Create(&accrual).Error; err != nil {
				failed++
				results = append(results, ImportResult{
					EmployeeName: employeeName,
					Success:      false,
					Error:        "Failed to create accrual: " + err.Error(),
				})
				continue
			}
		} else {
			// Update existing accrual record
			accrual.DaysAccrued = totalDays
			accrual.DaysUsed = daysTaken
			accrual.DaysBalance = netBalance
			accrual.IsProcessed = true
			accrual.ProcessedAt = &now

			notes := fmt.Sprintf("Updated from CSV: Opening=%.2f, Earned=%.2f, Total=%.2f, Taken=%.2f, Net=%.2f", opening, daysEarned, totalDays, daysTaken, netBalance)
			if accrual.Notes != nil && *accrual.Notes != "" {
				notes = *accrual.Notes + "\n" + notes
			}
			accrual.Notes = &notes

			if err := database.DB.Save(&accrual).Error; err != nil {
				failed++
				results = append(results, ImportResult{
					EmployeeName: employeeName,
					Success:      false,
					Error:        "Failed to update accrual: " + err.Error(),
				})
				continue
			}
		}

		success++
		results = append(results, ImportResult{
			EmployeeName: employeeName,
			Success:      true,
			Balance:      netBalance,
		})
	}

	c.JSON(http.StatusOK, BulkImportLeaveBalancesResponse{
		Total:   total,
		Success: success,
		Failed:  failed,
		Results: results,
		Month:   monthStr,
	})
}

// Helper functions for CSV parsing

func extractMonthFromLine(line string) string {
	// Look for patterns like "FEBRUARY 2022" or "FEB 2022"
	re := regexp.MustCompile(`(?i)(JANUARY|FEBRUARY|MARCH|APRIL|MAY|JUNE|JULY|AUGUST|SEPTEMBER|OCTOBER|NOVEMBER|DECEMBER|JAN|FEB|MAR|APR|MAY|JUN|JUL|AUG|SEP|OCT|NOV|DEC)\s+(\d{4})`)
	matches := re.FindStringSubmatch(line)
	if len(matches) == 3 {
		monthName := strings.ToUpper(matches[1])
		year := matches[2]

		monthMap := map[string]string{
			"JANUARY": "01", "JAN": "01",
			"FEBRUARY": "02", "FEB": "02",
			"MARCH": "03", "MAR": "03",
			"APRIL": "04", "APR": "04",
			"MAY":  "05",
			"JUNE": "06", "JUN": "06",
			"JULY": "07", "JUL": "07",
			"AUGUST": "08", "AUG": "08",
			"SEPTEMBER": "09", "SEP": "09",
			"OCTOBER": "10", "OCT": "10",
			"NOVEMBER": "11", "NOV": "11",
			"DECEMBER": "12", "DEC": "12",
		}

		if monthNum, ok := monthMap[monthName]; ok {
			return fmt.Sprintf("%s-%s", year, monthNum)
		}
	}
	return ""
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	// Handle empty or dash values
	if s == "" || s == "-" || s == " - " || s == " -" || s == "- " {
		return 0
	}
	// Remove any non-numeric characters except minus and decimal point
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "")

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}

func getStringPtrCSV(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func generateEmailFromName(firstname, lastname string) string {
	// Generate email: firstname.lastname@company.com
	// Remove special characters and spaces
	firstname = strings.ToLower(strings.TrimSpace(firstname))
	lastname = strings.ToLower(strings.TrimSpace(lastname))

	// Remove special characters
	firstname = regexp.MustCompile(`[^a-z0-9]`).ReplaceAllString(firstname, "")
	lastname = regexp.MustCompile(`[^a-z0-9]`).ReplaceAllString(lastname, "")

	if lastname == "" {
		return fmt.Sprintf("%s@company.com", firstname)
	}
	return fmt.Sprintf("%s.%s@company.com", firstname, lastname)
}

// AdminLeaveRequest represents a leave request created by admin
// Supports multipart/form-data (for file uploads)
type AdminLeaveRequest struct {
	EmployeeID  uint   `form:"employee_id" binding:"required" example:"1"`
	LeaveTypeID uint   `form:"leave_type_id" binding:"required" example:"1"`
	StartDate   string `form:"start_date" binding:"required" example:"2025-12-01"`
	EndDate     string `form:"end_date" binding:"required" example:"2025-12-05"`
	Reason      string `form:"reason" example:"Admin created leave"`
	Status      string `form:"status" example:"Approved"` // Optional: defaults to Approved
}

// UpdateLeaveRequest represents an update to a leave record
type UpdateLeaveRequest struct {
	StartDate       string `json:"start_date,omitempty" example:"2025-12-01"`
	EndDate         string `json:"end_date,omitempty" example:"2025-12-05"`
	Reason          string `json:"reason,omitempty" example:"Updated reason"`
	Status          string `json:"status,omitempty" example:"Approved"`
	RejectionReason string `json:"rejection_reason,omitempty" example:"Rejection reason"`
}

// CreateLeaveForEmployee creates a leave record for an employee (Admin only)
// @Summary Create leave for employee
// @Description Admin creates a leave record for any employee (Admin only)
// @Tags HR - Leave Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body AdminLeaveRequest true "Leave data"
// @Success 201 {object} models.Leave
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/hr/leaves [post]
func CreateLeaveForEmployee(c *gin.Context) {
	var req AdminLeaveRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form data: " + err.Error()})
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
	if err := utils.ValidateLeaveDates(startDate, endDate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check for overlapping leaves (excluding cancelled/rejected)
	hasOverlap, err := utils.CheckOverlappingLeaves(req.EmployeeID, startDate, endDate, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check overlapping leaves"})
		return
	}
	if hasOverlap {
		c.JSON(http.StatusConflict, gin.H{"error": utils.ErrOverlappingLeave.Error()})
		return
	}

	// Determine status (default to Approved for admin-created leaves)
	status := models.StatusApproved
	if req.Status != "" {
		switch req.Status {
		case string(models.StatusPending):
			status = models.StatusPending
		case string(models.StatusApproved):
			status = models.StatusApproved
		case string(models.StatusRejected):
			status = models.StatusRejected
		case string(models.StatusCancelled):
			status = models.StatusCancelled
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status. Use: Pending, Approved, Rejected, or Cancelled"})
			return
		}
	}

	// Get current user (admin)
	userID, _ := c.Get("user_id")
	adminID := userID.(uint)

	// If status is Approved and leave type uses balance, check balance and set approver; record-only types skip balance
	var approvedBy *uint
	var approvedAt *time.Time
	if status == models.StatusApproved {
		if leaveType.UsesBalance {
			utils.EnsureAccrualsUpToDate(req.EmployeeID, req.LeaveTypeID)

			balance, err := utils.GetCurrentLeaveBalance(req.EmployeeID, req.LeaveTypeID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate leave balance"})
				return
			}

			leaveDuration := float64(int(endDate.Sub(startDate).Hours()/24) + 1)
			if leaveDuration > balance {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":           "Insufficient leave balance",
					"current_balance": balance,
					"requested_days":  leaveDuration,
					"message":         fmt.Sprintf("Insufficient leave balance. Available: %.2f days, Requested: %.2f days.", balance, leaveDuration),
				})
				return
			}

			if leaveType.AllowCarryOver {
				if err := utils.UpdateCarryOverUsage(req.EmployeeID, req.LeaveTypeID, leaveDuration); err != nil {
					// Log error but don't fail the creation
				}
			}
		}
		approvedBy = &adminID
		now := time.Now()
		approvedAt = &now
	}

	// Handle leave form file upload - REQUIRED
	var formFileName, formFilePath, formMimeType *string
	var formFileSize *int64
	
	file, err := c.FormFile("leave_form")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Leave form attachment is required. Please upload a PNG or PDF file."})
		return
	}

	// Validate file extension (PNG/PDF only)
	if err := utils.ValidateLeaveFormFileExtension(file.Filename); err != nil {
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
	if err := utils.ValidateLeaveFormMimeType(mimeType); err != nil {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": err.Error()})
		return
	}

	// Generate secure filename
	secureFilename, err := utils.GenerateSecureFileName(file.Filename, req.EmployeeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate filename"})
		return
	}

	// Save file to storage (we'll update leave ID after creation)
	relativePath, fileSize, err := utils.SaveLeaveFormFile(src, secureFilename, req.EmployeeID, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file: " + err.Error()})
		return
	}

	formFileName = &file.Filename
	formFilePath = &relativePath
	formMimeType = &mimeType
	formFileSize = &fileSize

	// Create leave record
	leave := models.Leave{
		EmployeeID:  req.EmployeeID,
		LeaveTypeID: req.LeaveTypeID,
		StartDate:   startDate,
		EndDate:     endDate,
		Reason:      req.Reason,
		Status:      status,
		ApprovedBy:  approvedBy,
		ApprovedAt:  approvedAt,
		FormFileName: formFileName,
		FormFilePath: formFilePath,
		FormFileSize: formFileSize,
		FormMimeType: formMimeType,
	}

	if err := database.DB.Create(&leave).Error; err != nil {
		// Clean up file if database save fails
		if formFilePath != nil {
			utils.DeleteLeaveFormFile(*formFilePath)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create leave record"})
		return
	}

	// If file was uploaded, update the file path with leave ID (for better organization)
	if formFilePath != nil {
		// The file is already saved, we can optionally rename it to include leave ID
		// For now, we'll keep the current structure
	}

	if status == models.StatusApproved && leaveType.UsesBalance {
		if err := utils.EnsureAccrualsUpToDate(req.EmployeeID, req.LeaveTypeID); err != nil {
			// Log error but don't fail the creation
		}
	}

	// Create audit log
	createAuditLog(models.AuditEntityEmployee, req.EmployeeID, models.AuditActionCreate, adminID, c,
		nil, map[string]interface{}{
			"leave_id":      leave.ID,
			"leave_type_id": req.LeaveTypeID,
			"start_date":    startDate.Format("2006-01-02"),
			"end_date":      endDate.Format("2006-01-02"),
			"status":        string(status),
			"reason":        req.Reason,
		})

	// Load associations
	database.DB.Preload("LeaveType").Preload("Employee").Preload("Approver").First(&leave, leave.ID)

	c.JSON(http.StatusCreated, leave)
}

// DownloadLeaveForm downloads the leave form attachment
// @Summary Download leave form attachment
// @Description Download the leave form file (PNG/PDF) attached to a leave record
// @Tags HR - Leave Management
// @Produce application/octet-stream
// @Security BearerAuth
// @Param id path int true "Leave ID"
// @Success 200 {file} file
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/hr/leaves/:id/form [get]
func DownloadLeaveForm(c *gin.Context) {
	leaveID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid leave ID"})
		return
	}

	var leave models.Leave
	if err := database.DB.First(&leave, uint(leaveID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Leave not found"})
		return
	}

	// Check if leave has a form attachment
	if leave.FormFilePath == nil || *leave.FormFilePath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "No leave form attachment found for this leave"})
		return
	}

	// Get full file path
	fullPath := utils.GetLeaveFormFilePath(*leave.FormFilePath)

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Leave form file not found on server"})
		return
	}

	// Set appropriate headers
	if leave.FormFileName != nil {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", *leave.FormFileName))
	} else {
		c.Header("Content-Disposition", "attachment; filename=\"leave_form\"")
	}

	if leave.FormMimeType != nil {
		c.Header("Content-Type", *leave.FormMimeType)
	} else {
		c.Header("Content-Type", "application/octet-stream")
	}

	// Serve the file
	c.File(fullPath)
}

// UpdateLeaveForEmployee updates a leave record (Admin only)
// @Summary Update leave for employee
// @Description Admin updates a leave record for any employee (Admin only)
// @Tags HR - Leave Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave ID"
// @Param request body UpdateLeaveRequest true "Leave update data"
// @Success 200 {object} models.Leave
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/hr/leaves/{id} [put]
func UpdateLeaveForEmployee(c *gin.Context) {
	leaveID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid leave ID"})
		return
	}

	var req UpdateLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get leave record
	var leave models.Leave
	if err := database.DB.Preload("Employee").Preload("LeaveType").First(&leave, uint(leaveID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Leave not found"})
		return
	}

	// Get current user (admin)
	userID, _ := c.Get("user_id")
	adminID := userID.(uint)

	oldStatus := string(leave.Status)
	oldStartDate := leave.StartDate
	oldEndDate := leave.EndDate

	// Update dates if provided
	if req.StartDate != "" {
		startDate, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format. Use YYYY-MM-DD"})
			return
		}
		leave.StartDate = startDate
	}

	if req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format. Use YYYY-MM-DD"})
			return
		}
		leave.EndDate = endDate
	}

	// Validate dates
	if err := utils.ValidateLeaveDates(leave.StartDate, leave.EndDate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check for overlapping leaves (excluding this leave)
	hasOverlap, err := utils.CheckOverlappingLeaves(leave.EmployeeID, leave.StartDate, leave.EndDate, &leave.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check overlapping leaves"})
		return
	}
	if hasOverlap {
		c.JSON(http.StatusConflict, gin.H{"error": utils.ErrOverlappingLeave.Error()})
		return
	}

	// Update reason if provided
	if req.Reason != "" {
		leave.Reason = req.Reason
	}

	// Update status if provided
	if req.Status != "" {
		switch req.Status {
		case string(models.StatusPending):
			leave.Status = models.StatusPending
			leave.ApprovedBy = nil
			leave.ApprovedAt = nil
			leave.RejectionReason = ""
		case string(models.StatusApproved):
			if oldStatus != string(models.StatusApproved) {
				if leave.LeaveType.UsesBalance {
					utils.EnsureAccrualsUpToDate(leave.EmployeeID, leave.LeaveTypeID)

					balance, err := utils.GetAvailableLeaveBalance(leave.EmployeeID, leave.LeaveTypeID, &leave.ID, &leave.StartDate)
					if err != nil {
						balance, err = utils.GetCurrentLeaveBalance(leave.EmployeeID, leave.LeaveTypeID)
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate leave balance"})
							return
						}
					}

					leaveDuration := float64(leave.GetDuration())
					if leaveDuration > balance {
						c.JSON(http.StatusBadRequest, gin.H{
							"error":           "Insufficient leave balance",
							"current_balance": balance,
							"requested_days":  leaveDuration,
							"message":         fmt.Sprintf("Insufficient leave balance. Available: %.2f days, Requested: %.2f days.", balance, leaveDuration),
						})
						return
					}

					if leave.LeaveType.AllowCarryOver {
						if err := utils.UpdateCarryOverUsage(leave.EmployeeID, leave.LeaveTypeID, leaveDuration); err != nil {
							// Log error but don't fail the update
						}
					}
				}
				leave.ApprovedBy = &adminID
				now := time.Now()
				leave.ApprovedAt = &now
				leave.RejectionReason = ""
			}
			leave.Status = models.StatusApproved
		case string(models.StatusRejected):
			leave.Status = models.StatusRejected
			if req.RejectionReason != "" {
				leave.RejectionReason = req.RejectionReason
			}
			leave.ApprovedBy = &adminID
			now := time.Now()
			leave.ApprovedAt = &now
		case string(models.StatusCancelled):
			leave.Status = models.StatusCancelled
			leave.RejectionReason = ""
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status. Use: Pending, Approved, Rejected, or Cancelled"})
			return
		}
	}

	// Update rejection reason if provided and status is rejected
	if req.RejectionReason != "" && leave.Status == models.StatusRejected {
		leave.RejectionReason = req.RejectionReason
	}

	if err := database.DB.Save(&leave).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update leave record"})
		return
	}

	if leave.LeaveType.UsesBalance && (leave.Status == models.StatusApproved || oldStatus == string(models.StatusApproved)) {
		if err := utils.EnsureAccrualsUpToDate(leave.EmployeeID, leave.LeaveTypeID); err != nil {
			// Log error but don't fail the update
		}
	}

	// Create audit log
	createAuditLog(models.AuditEntityEmployee, leave.EmployeeID, models.AuditActionUpdate, adminID, c,
		map[string]interface{}{
			"leave_id":   leave.ID,
			"status":     oldStatus,
			"start_date": oldStartDate.Format("2006-01-02"),
			"end_date":   oldEndDate.Format("2006-01-02"),
		},
		map[string]interface{}{
			"leave_id":   leave.ID,
			"status":     string(leave.Status),
			"start_date": leave.StartDate.Format("2006-01-02"),
			"end_date":   leave.EndDate.Format("2006-01-02"),
			"reason":     leave.Reason,
		})

	// Load associations
	database.DB.Preload("LeaveType").Preload("Employee").Preload("Approver").First(&leave, leave.ID)

	c.JSON(http.StatusOK, leave)
}

// DeleteLeaveForEmployee deletes a leave record (Admin only)
// @Summary Delete leave for employee
// @Description Admin deletes a leave record for any employee (Admin only)
// @Tags HR - Leave Management
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave ID"
// @Success 200 {object} MessageResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/hr/leaves/{id} [delete]
func DeleteLeaveForEmployee(c *gin.Context) {
	leaveID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid leave ID"})
		return
	}

	// Get leave record
	var leave models.Leave
	if err := database.DB.Preload("Employee").Preload("LeaveType").First(&leave, uint(leaveID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Leave not found"})
		return
	}

	// Get current user (admin)
	userID, _ := c.Get("user_id")
	adminID := userID.(uint)

	// Create audit log before deletion
	createAuditLog(models.AuditEntityEmployee, leave.EmployeeID, models.AuditActionDelete, adminID, c,
		map[string]interface{}{
			"leave_id":   leave.ID,
			"status":     string(leave.Status),
			"start_date": leave.StartDate.Format("2006-01-02"),
			"end_date":   leave.EndDate.Format("2006-01-02"),
			"reason":     leave.Reason,
		}, nil)

	// Delete leave record (soft delete)
	if err := database.DB.Delete(&leave).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete leave record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Leave record deleted successfully"})
}

// GetEmployeeLeaves gets all leave records for an employee (Admin only)
// @Summary Get employee leaves
// @Description Admin gets all leave records for any employee (Admin only)
// @Tags HR - Leave Management
// @Produce json
// @Security BearerAuth
// @Param id path int true "Employee ID"
// @Param status query string false "Filter by status (Pending, Approved, Rejected, Cancelled)"
// @Param leave_type_id query int false "Filter by leave type ID"
// @Param start_date query string false "Filter by start date (YYYY-MM-DD)"
// @Param end_date query string false "Filter by end date (YYYY-MM-DD)"
// @Success 200 {array} models.Leave
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/hr/employees/{id}/leaves [get]
func GetEmployeeLeaves(c *gin.Context) {
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
		Preload("LeaveType").
		Preload("Employee").
		Preload("Approver").
		Order("start_date DESC, created_at DESC")

	// Apply filters
	status := c.Query("status")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	leaveTypeIDStr := c.Query("leave_type_id")
	if leaveTypeIDStr != "" {
		leaveTypeID, err := strconv.ParseUint(leaveTypeIDStr, 10, 32)
		if err == nil {
			query = query.Where("leave_type_id = ?", leaveTypeID)
		}
	}

	startDateStr := c.Query("start_date")
	if startDateStr != "" {
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err == nil {
			query = query.Where("start_date >= ?", startDate)
		}
	}

	endDateStr := c.Query("end_date")
	if endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err == nil {
			query = query.Where("end_date <= ?", endDate)
		}
	}

	var leaves []models.Leave
	if err := query.Find(&leaves).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaves"})
		return
	}

	c.JSON(http.StatusOK, leaves)
}
