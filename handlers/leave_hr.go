package handlers

import (
	"fmt"
	"hrms-api/database"
	"hrms-api/models"
	"hrms-api/utils"
	"net/http"
	"strconv"
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
	EmployeeID     uint                   `json:"employee_id"`
	EmployeeName   string                 `json:"employee_name"`
	TotalAccrued   float64                `json:"total_accrued" example:"24.0"`
	TotalUsed      float64                `json:"total_used" example:"5.0"`
	CurrentBalance float64                `json:"current_balance" example:"19.0"`
	Accruals       []LeaveAccrualResponse `json:"accruals"`
	PendingLeaves  int                    `json:"pending_leaves"`
	UpcomingLeaves int                    `json:"upcoming_leaves"`
}

// LeaveCalendarResponse represents leave calendar data
type LeaveCalendarResponse struct {
	Date         string `json:"date" example:"2025-12-15"`
	EmployeeID   uint   `json:"employee_id"`
	EmployeeName string `json:"employee_name"`
	Department   string `json:"department"`
	LeaveType    string `json:"leave_type"`
	Status       string `json:"status"`
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
	var accruals []models.LeaveAccrual
	database.DB.Where("employee_id = ? AND leave_type_id = ?", employeeID, annualLeaveType.ID).
		Order("accrual_month DESC").
		Find(&accruals)

	// Calculate totals
	var totalAccrued, totalUsed, currentBalance float64
	accrualResponses := make([]LeaveAccrualResponse, 0, len(accruals))

	for _, acc := range accruals {
		totalAccrued += acc.DaysAccrued
		totalUsed += acc.DaysUsed
		currentBalance = acc.DaysBalance // Latest balance

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
		EmployeeID:     uint(employeeID),
		EmployeeName:   employee.Firstname + " " + employee.Lastname,
		TotalAccrued:   totalAccrued,
		TotalUsed:      totalUsed,
		CurrentBalance: currentBalance,
		Accruals:       accrualResponses,
		PendingLeaves:  int(pendingLeaves),
		UpcomingLeaves: int(upcomingLeaves),
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
	query := database.DB.Model(&models.Leave{}).
		Where("status = ? AND start_date <= ? AND end_date >= ?", models.StatusApproved, endDate, startDate).
		Preload("Employee").
		Preload("LeaveType")

	if department != "" {
		query = query.Where("employees.department = ?", department)
	}

	var leaves []models.Leave
	query.Find(&leaves)

	// Generate calendar entries for each day
	calendar := make([]LeaveCalendarResponse, 0)
	currentDate := startDate
	for !currentDate.After(endDate) {
		for _, leave := range leaves {
			if !currentDate.Before(leave.StartDate) && !currentDate.After(leave.EndDate) {
				calendar = append(calendar, LeaveCalendarResponse{
					Date:         currentDate.Format("2006-01-02"),
					EmployeeID:   leave.EmployeeID,
					EmployeeName: leave.Employee.Firstname + " " + leave.Employee.Lastname,
					Department:   leave.Employee.Department,
					LeaveType:    leave.LeaveType.Name,
					Status:       string(leave.Status),
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

// ProcessMonthlyAccruals processes leave accruals for all employees for a specific month
// @Summary Process monthly accruals
// @Description Process leave accruals for all employees for a specific month (Admin only)
// @Tags HR - Leave Management
// @Produce json
// @Security BearerAuth
// @Param month query string false "Month to process (YYYY-MM)" default:"current month"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/hr/leaves/process-accruals [post]
func ProcessMonthlyAccruals(c *gin.Context) {
	monthStr := c.Query("month")

	var processMonth time.Time
	var err error

	if monthStr == "" {
		// Default to previous month (accruals are processed at end of month)
		now := time.Now()
		processMonth = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		processMonth, err = time.Parse("2006-01", monthStr)
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

	// Get all active employees
	var employees []models.Employee
	database.DB.Find(&employees)

	processed := 0
	errors := 0

	for _, emp := range employees {
		if err := utils.ProcessMonthlyAccrual(emp.ID, annualLeaveType.ID, processMonth); err != nil {
			errors++
			continue
		}
		processed++
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Accrual processing completed",
		"month":     processMonth.Format("2006-01"),
		"processed": processed,
		"errors":    errors,
	})
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
	var latestAccrual models.LeaveAccrual
	if err := database.DB.Where("employee_id = ? AND leave_type_id = ?", employeeID, annualLeaveType.ID).
		Order("accrual_month DESC").
		First(&latestAccrual).Error; err != nil {
		// No accrual record exists, create one for current month
		now := time.Now()
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		latestAccrual = models.LeaveAccrual{
			EmployeeID:   uint(employeeID),
			LeaveTypeID:  annualLeaveType.ID,
			AccrualMonth: monthStart,
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
	if latestAccrual.DaysBalance < 0 {
		latestAccrual.DaysBalance = 0
	}

	// Add adjustment to notes
	notes := fmt.Sprintf("Manual adjustment: %+.2f days. Previous balance: %.2f, New balance: %.2f. Reason: %s",
		req.Days, oldBalance, latestAccrual.DaysBalance, req.Reason)
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
		AccrualMonth: monthStart,
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

	// Build query
	query := database.DB.Model(&models.Employee{})
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
		var accruals []models.LeaveAccrual
		database.DB.Where("employee_id = ? AND leave_type_id = ?", emp.ID, annualLeaveType.ID).
			Order("accrual_month DESC").
			Find(&accruals)

		var totalAccrued, totalUsed, currentBalance float64
		accrualResponses := make([]LeaveAccrualResponse, 0, len(accruals))

		for _, acc := range accruals {
			totalAccrued += acc.DaysAccrued
			totalUsed += acc.DaysUsed
			currentBalance = acc.DaysBalance

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
			EmployeeID:     emp.ID,
			EmployeeName:   emp.Firstname + " " + emp.Lastname,
			TotalAccrued:   totalAccrued,
			TotalUsed:      totalUsed,
			CurrentBalance: currentBalance,
			Accruals:       accrualResponses,
			PendingLeaves:  int(pendingLeaves),
			UpcomingLeaves: int(upcomingLeaves),
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

	// Build query (same logic as GetAllEmployeesLeaveBalances)
	query := database.DB.Model(&models.Employee{})
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

		var totalAccrued, totalUsed, currentBalance float64
		for _, acc := range accruals {
			totalAccrued += acc.DaysAccrued
			totalUsed += acc.DaysUsed
			currentBalance = acc.DaysBalance
		}

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
			EmployeeID:     emp.ID,
			EmployeeName:   emp.Firstname + " " + emp.Lastname,
			TotalAccrued:   totalAccrued,
			TotalUsed:      totalUsed,
			CurrentBalance: currentBalance,
			PendingLeaves:  int(pendingLeaves),
			UpcomingLeaves: int(upcomingLeaves),
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
