package utils

import (
	"fmt"
	"hrms-api/database"
	"hrms-api/models"
	"time"
)

// ProcessMonthlyAccrualSimple processes leave accrual for a specific month using simplified schema
// This creates a record with year, month, and days_accrued (2.0 days per month for Annual Leave)
func ProcessMonthlyAccrualSimple(employeeID uint, leaveTypeID uint, year int, month int) error {
	// Get leave type to check accrual rate
	var leaveType models.LeaveType
	if err := database.DB.First(&leaveType, leaveTypeID).Error; err != nil {
		return fmt.Errorf("leave type not found: %w", err)
	}

	// Get employee to check if they were active during the month
	var employee models.Employee
	if err := database.DB.First(&employee, employeeID).Error; err != nil {
		return fmt.Errorf("employee not found: %w", err)
	}

	// Check if employee was active during the month
	// If date_joined is set, check if they joined before or during the month
	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0).AddDate(0, 0, -1)

	// Check if employee was active during this month
	if employee.Status != "active" {
		// Check if status changed during the month (simplified - just check current status)
		// In a full implementation, you'd check employment history
		return fmt.Errorf("employee is not active")
	}

	// Check if employee joined during or before the month
	if employee.DateJoined != nil {
		if employee.DateJoined.After(monthEnd) {
			// Employee joined after this month, no accrual
			return nil
		}
		// Optional: Prorate for mid-month join
		// For now, we'll give full accrual if they joined during the month
	}

	// Check if accrual already exists
	// Use Find() with Limit(1) instead of First() to avoid logging "record not found" errors
	var existingAccruals []models.LeaveAccrual
	database.DB.Where("employee_id = ? AND leave_type_id = ? AND year = ? AND month = ?",
		employeeID, leaveTypeID, year, month).Limit(1).Find(&existingAccruals)
	if len(existingAccruals) > 0 && existingAccruals[0].ID > 0 {
		// Already exists, skip
		return nil
	}

	// Calculate days to accrue (use accrual_rate from leave type, default 2.0)
	daysToAccrue := leaveType.AccrualRate
	if daysToAccrue == 0 {
		daysToAccrue = 2.0 // Default to 2.0 days per month for Annual Leave
	}

	// Optional: Prorate for mid-month join
	if employee.DateJoined != nil && employee.DateJoined.After(monthStart) {
		// Employee joined mid-month, prorate
		daysInMonth := float64(monthEnd.Day())
		daysWorked := float64(monthEnd.Day() - employee.DateJoined.Day() + 1)
		daysToAccrue = (daysToAccrue / daysInMonth) * daysWorked
	}

	// Create accrual record
	accrual := models.LeaveAccrual{
		EmployeeID:  employeeID,
		LeaveTypeID: leaveTypeID,
		Year:        year,
		Month:       month,
		DaysAccrued: daysToAccrue,
	}

	if err := database.DB.Create(&accrual).Error; err != nil {
		return fmt.Errorf("failed to create accrual: %w", err)
	}

	return nil
}

// CalculateLeaveBalanceSimple calculates leave balance using simplified formula:
// Total Accrued - Total Taken = Balance
func CalculateLeaveBalanceSimple(employeeID uint, leaveTypeID uint) (float64, error) {
	// Get total accrued
	var totalAccrued float64
	database.DB.Model(&models.LeaveAccrual{}).
		Where("employee_id = ? AND leave_type_id = ?", employeeID, leaveTypeID).
		Select("COALESCE(SUM(days_accrued), 0)").
		Scan(&totalAccrued)

	// Get total taken
	var totalTaken float64
	database.DB.Model(&models.LeaveTaken{}).
		Where("employee_id = ? AND leave_type_id = ?", employeeID, leaveTypeID).
		Select("COALESCE(SUM(days_taken), 0)").
		Scan(&totalTaken)

	// Balance = Total Accrued - Total Taken
	balance := totalAccrued - totalTaken

	return balance, nil
}
