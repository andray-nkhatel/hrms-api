package utils

import (
	"fmt"
	"hrms-api/database"
	"hrms-api/models"
	"time"
)

// CalculateProjectedAnnualLeaveBalance calculates the projected annual leave balance
// at a future date, accounting for monthly accruals between now and the target date
func CalculateProjectedAnnualLeaveBalance(employeeID uint, leaveTypeID uint, targetDate time.Time) (float64, error) {
	// Get current balance
	currentBalance, err := GetCurrentLeaveBalance(employeeID, leaveTypeID)
	if err != nil {
		return 0, err
	}

	// Calculate how many additional days will be accrued between now and target date
	now := time.Now()
	if targetDate.Before(now) || targetDate.Equal(now) {
		// If target date is today or in the past, return current balance
		return currentBalance, nil
	}

	// Calculate total projected accrued by target date
	// This will include all accruals up to and including the target date
	projectedAccrued, err := CalculateAnnualLeaveAccrued(employeeID, leaveTypeID, targetDate)
	if err != nil {
		return 0, err
	}

	// Get total used (all approved leaves, including pending ones that will be approved)
	var usedDays float64
	var leaves []models.Leave
	// Count approved leaves and pending leaves (assuming they'll be approved)
	database.DB.Where("employee_id = ? AND leave_type_id = ? AND status IN (?)",
		employeeID, leaveTypeID, []models.LeaveStatus{models.StatusApproved, models.StatusPending}).Find(&leaves)

	for _, leave := range leaves {
		// Only count leaves that are before or on the target date
		if leave.StartDate.Before(targetDate) || leave.StartDate.Equal(targetDate) {
			usedDays += float64(leave.GetDuration())
		}
	}

	// Projected balance = projected accrued - used days
	projectedBalance := projectedAccrued - usedDays
	if projectedBalance < 0 {
		projectedBalance = 0
	}

	return projectedBalance, nil
}

const (
	AnnualLeaveDaysPerYear  = 24.0
	AnnualLeaveDaysPerMonth = 2.0
)

// CalculateAnnualLeaveAccrued calculates how many days of annual leave an employee has accrued
// based on their employment start date and the current date
func CalculateAnnualLeaveAccrued(employeeID uint, leaveTypeID uint, asOfDate time.Time) (float64, error) {
	// Get employee's employment details to find start date
	var employment models.EmploymentDetails
	if err := database.DB.Where("employee_id = ?", employeeID).First(&employment).Error; err != nil {
		// If no employment details, try to get from employee created_at
		var employee models.Employee
		if err := database.DB.First(&employee, employeeID).Error; err != nil {
			return 0, fmt.Errorf("employee not found")
		}
		// Use employee creation date as fallback
		return calculateAccruedFromDate(employee.CreatedAt, asOfDate), nil
	}

	// Use hire date or start date
	var startDate time.Time
	if employment.HireDate != nil {
		startDate = *employment.HireDate
	} else if employment.StartDate != nil {
		startDate = *employment.StartDate
	} else {
		// Fallback to employee creation date
		var employee models.Employee
		if err := database.DB.First(&employee, employeeID).Error; err != nil {
			return 0, fmt.Errorf("employee not found")
		}
		startDate = employee.CreatedAt
	}

	return calculateAccruedFromDate(startDate, asOfDate), nil
}

// calculateAccruedFromDate calculates accrued leave from start date to end date
func calculateAccruedFromDate(startDate, endDate time.Time) float64 {
	// Normalize dates to first of month for accurate month calculation
	start := time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(endDate.Year(), endDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Calculate months between dates
	months := 0
	current := start
	for current.Before(end) || current.Equal(end) {
		months++
		current = current.AddDate(0, 1, 0)
	}

	// Subtract 1 if we're counting from start month (employee earns in the month after starting)
	// For example: if employee started in January, they earn 2 days in February
	if months > 0 {
		months-- // First month doesn't count (accrual happens at end of first month)
	}

	// Cap at 12 months (24 days per year max)
	if months > 12 {
		months = 12
	}

	return float64(months) * AnnualLeaveDaysPerMonth
}

// ProcessMonthlyAccrual processes leave accrual for a specific month
func ProcessMonthlyAccrual(employeeID uint, leaveTypeID uint, accrualMonth time.Time) error {
	// Check if already processed
	monthStart := time.Date(accrualMonth.Year(), accrualMonth.Month(), 1, 0, 0, 0, 0, time.UTC)

	var existing models.LeaveAccrual
	err := database.DB.Where("employee_id = ? AND leave_type_id = ? AND accrual_month = ?",
		employeeID, leaveTypeID, monthStart).First(&existing).Error

	if err == nil && existing.IsProcessed {
		// Already processed
		return nil
	}

	// Get previous month's balance
	prevMonth := monthStart.AddDate(0, -1, 0)
	var prevAccrual models.LeaveAccrual
	prevBalance := 0.0
	database.DB.Where("employee_id = ? AND leave_type_id = ? AND accrual_month = ?",
		employeeID, leaveTypeID, prevMonth).First(&prevAccrual)
	if prevAccrual.ID > 0 {
		prevBalance = prevAccrual.DaysBalance
	}

	// Calculate days used in this month
	daysUsed := CalculateDaysUsedInMonth(employeeID, leaveTypeID, monthStart)

	// Calculate new balance
	newAccrued := AnnualLeaveDaysPerMonth
	newBalance := prevBalance + newAccrued - daysUsed

	// Create or update accrual record
	now := time.Now()
	if existing.ID > 0 {
		existing.DaysAccrued = newAccrued
		existing.DaysUsed = daysUsed
		existing.DaysBalance = newBalance
		existing.IsProcessed = true
		existing.ProcessedAt = &now
		return database.DB.Save(&existing).Error
	}

	accrual := models.LeaveAccrual{
		EmployeeID:   employeeID,
		LeaveTypeID:  leaveTypeID,
		AccrualMonth: monthStart,
		DaysAccrued:  newAccrued,
		DaysUsed:     daysUsed,
		DaysBalance:  newBalance,
		IsProcessed:  true,
		ProcessedAt:  &now,
	}

	return database.DB.Create(&accrual).Error
}

// CalculateDaysUsedInMonth calculates days used in a specific month
func CalculateDaysUsedInMonth(employeeID uint, leaveTypeID uint, monthStart time.Time) float64 {
	monthEnd := monthStart.AddDate(0, 1, 0).AddDate(0, 0, -1)

	var leaves []models.Leave
	database.DB.Where("employee_id = ? AND leave_type_id = ? AND status = ? AND start_date <= ? AND end_date >= ?",
		employeeID, leaveTypeID, models.StatusApproved, monthEnd, monthStart).Find(&leaves)

	var daysUsed float64
	for _, leave := range leaves {
		// Calculate overlap with the month
		overlapStart := leave.StartDate
		if overlapStart.Before(monthStart) {
			overlapStart = monthStart
		}
		overlapEnd := leave.EndDate
		if overlapEnd.After(monthEnd) {
			overlapEnd = monthEnd
		}

		if !overlapStart.After(overlapEnd) {
			duration := overlapEnd.Sub(overlapStart)
			daysUsed += duration.Hours()/24 + 1
		}
	}

	return daysUsed
}

// GetCurrentLeaveBalance calculates current leave balance including accruals
// For annual leave, it uses accrual records to ensure manual adjustments are reflected
func GetCurrentLeaveBalance(employeeID uint, leaveTypeID uint) (float64, error) {
	// Get leave type to check if it's annual leave
	var leaveType models.LeaveType
	if err := database.DB.First(&leaveType, leaveTypeID).Error; err != nil {
		return 0, err
	}

	// For annual leave, use accrual records to get the actual balance
	// This ensures manual adjustments are reflected
	if leaveType.Name == "Annual" || leaveType.MaxDays == 24 {
		// Ensure accruals are up to date first
		if err := EnsureAccrualsUpToDate(employeeID, leaveTypeID); err != nil {
			return 0, err
		}

		// Get the latest accrual record which contains the current balance
		var latestAccrual models.LeaveAccrual
		if err := database.DB.Where("employee_id = ? AND leave_type_id = ?", employeeID, leaveTypeID).
			Order("accrual_month DESC").
			First(&latestAccrual).Error; err == nil {
			// Use the balance from the latest accrual record (includes manual adjustments)
			return latestAccrual.DaysBalance, nil
		}

		// If no accrual records exist, calculate from scratch
		accrued, err := CalculateAnnualLeaveAccrued(employeeID, leaveTypeID, time.Now())
		if err != nil {
			return 0, err
		}

		// Get total used (all approved leaves)
		var usedDays float64
		var leaves []models.Leave
		database.DB.Where("employee_id = ? AND leave_type_id = ? AND status = ?",
			employeeID, leaveTypeID, models.StatusApproved).Find(&leaves)

		for _, leave := range leaves {
			usedDays += float64(leave.GetDuration())
		}

		balance := accrued - usedDays
		if balance < 0 {
			balance = 0
		}

		return balance, nil
	}

	// For other leave types, use the original calculation
	balance, err := CalculateLeaveBalance(employeeID, leaveTypeID)
	return float64(balance), err
}

// EnsureAccrualsUpToDate ensures all accruals are processed up to the current month
func EnsureAccrualsUpToDate(employeeID uint, leaveTypeID uint) error {
	// Get employee start date
	var employment models.EmploymentDetails
	startDate := time.Now()
	if err := database.DB.Where("employee_id = ?", employeeID).First(&employment).Error; err == nil {
		if employment.HireDate != nil {
			startDate = *employment.HireDate
		} else if employment.StartDate != nil {
			startDate = *employment.StartDate
		}
	}

	// Process accruals from start date to current month
	currentMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	processMonth := time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Start from the month after employment (first accrual happens at end of first month)
	processMonth = processMonth.AddDate(0, 1, 0)

	for !processMonth.After(currentMonth) {
		if err := ProcessMonthlyAccrual(employeeID, leaveTypeID, processMonth); err != nil {
			return err
		}
		processMonth = processMonth.AddDate(0, 1, 0)
	}

	return nil
}
