package utils

import (
	"fmt"
	"hrms-api/database"
	"hrms-api/models"
	"time"
)

// GetCarryOverBalance calculates the total available carry-over balance for an employee
// This includes all non-expired carry-over days
func GetCarryOverBalance(employeeID uint, leaveTypeID uint) (float64, error) {
	now := time.Now()
	var carryOvers []models.LeaveCarryOver

	// Get all non-expired carry-overs
	query := database.DB.Where("employee_id = ? AND leave_type_id = ? AND is_expired = ?",
		employeeID, leaveTypeID, false)

	// Check expiry dates
	query = query.Where("(expiry_date IS NULL OR expiry_date >= ?)", now)

	if err := query.Find(&carryOvers).Error; err != nil {
		return 0, err
	}

	var totalBalance float64
	for _, co := range carryOvers {
		totalBalance += co.DaysRemaining
	}

	return totalBalance, nil
}

// ProcessYearEndCarryOver processes carry-over for an employee at year-end
// This should be called at the end of each year to carry over unused leave
func ProcessYearEndCarryOver(employeeID uint, leaveTypeID uint, fromYear int, processedBy *uint) (*models.LeaveCarryOver, error) {
	// Get leave type to check carry-over settings
	var leaveType models.LeaveType
	if err := database.DB.First(&leaveType, leaveTypeID).Error; err != nil {
		return nil, fmt.Errorf("leave type not found")
	}

	// Check if carry-over is allowed
	if !leaveType.AllowCarryOver {
		return nil, fmt.Errorf("carry-over is not allowed for this leave type")
	}

	// Check if carry-over already processed for this year
	var existing models.LeaveCarryOver
	if err := database.DB.Where("employee_id = ? AND leave_type_id = ? AND from_year = ?",
		employeeID, leaveTypeID, fromYear).First(&existing).Error; err == nil {
		return &existing, nil // Already processed
	}

	// Calculate year-end balance using ONLY current year's accrual (not including previous carry-over)
	yearEnd := time.Date(fromYear, 12, 31, 23, 59, 59, 0, time.UTC)
	yearStart := time.Date(fromYear, 1, 1, 0, 0, 0, 0, time.UTC)

	// Get accruals for the current year only (to calculate current year's accrued balance)
	// Handle both accrual_month and year/month schemas
	var currentYearAccruals []models.LeaveAccrual
	database.DB.Where("employee_id = ? AND leave_type_id = ? AND (accrual_month >= ? AND accrual_month <= ? OR (year = ? AND accrual_month IS NULL))",
		employeeID, leaveTypeID, yearStart, yearEnd, yearStart.Year()).
		Order("COALESCE(accrual_month, MAKE_DATE(year, month, 1)) ASC").
		Find(&currentYearAccruals)

	// Calculate days accrued in the current year only
	var yearAccrued float64
	for _, acc := range currentYearAccruals {
		yearAccrued += acc.DaysAccrued
	}

	// Cap current year accrual at 24 days (annual entitlement)
	if yearAccrued > 24.0 {
		yearAccrued = 24.0
	}

	// Calculate days used in the year
	var leaves []models.Leave
	database.DB.Where("employee_id = ? AND leave_type_id = ? AND status = ? AND start_date >= ? AND start_date <= ?",
		employeeID, leaveTypeID, models.StatusApproved, yearStart, yearEnd).Find(&leaves)

	var daysUsed float64
	for _, leave := range leaves {
		// Only count days within the year
		leaveStart := leave.StartDate
		if leaveStart.Before(yearStart) {
			leaveStart = yearStart
		}
		leaveEnd := leave.EndDate
		if leaveEnd.After(yearEnd) {
			leaveEnd = yearEnd
		}
		if !leaveStart.After(leaveEnd) {
			duration := leaveEnd.Sub(leaveStart)
			daysUsed += duration.Hours()/24 + 1
		}
	}

	// Calculate unused balance from current year only (this is what can be carried over)
	// This is current year's accrued minus current year's used (excluding previous carry-over)
	unusedBalance := yearAccrued - daysUsed
	if unusedBalance < 0 {
		unusedBalance = 0
	}

	// Apply max carry-over limit if set
	daysToCarryOver := unusedBalance
	if leaveType.MaxCarryOverDays != nil && *leaveType.MaxCarryOverDays < daysToCarryOver {
		daysToCarryOver = *leaveType.MaxCarryOverDays
	}

	// If nothing to carry over, return nil
	if daysToCarryOver <= 0 {
		return nil, nil
	}

	// Calculate expiry date
	var expiryDate *time.Time
	toYear := fromYear + 1
	if leaveType.CarryOverExpiryMonths != nil {
		// Expiry is X months after year-end
		expiry := yearEnd.AddDate(0, *leaveType.CarryOverExpiryMonths, 0)
		expiryDate = &expiry
	} else if leaveType.CarryOverExpiryDate != nil {
		// Fixed expiry date (e.g., end of Q1)
		expiry := time.Date(toYear, leaveType.CarryOverExpiryDate.Month(), leaveType.CarryOverExpiryDate.Day(), 0, 0, 0, 0, time.UTC)
		expiryDate = &expiry
	}

	// Create carry-over record
	now := time.Now()
	carryOver := models.LeaveCarryOver{
		EmployeeID:      employeeID,
		LeaveTypeID:     leaveTypeID,
		FromYear:        fromYear,
		ToYear:          toYear,
		DaysCarriedOver: daysToCarryOver,
		DaysUsed:        0,
		DaysRemaining:   daysToCarryOver,
		ExpiryDate:      expiryDate,
		IsExpired:       false,
		ProcessedAt:     now,
		ProcessedBy:     processedBy,
	}

	if err := database.DB.Create(&carryOver).Error; err != nil {
		return nil, fmt.Errorf("failed to create carry-over record: %w", err)
	}

	return &carryOver, nil
}

// ProcessCarryOverForAllEmployees processes carry-over for all employees at year-end
func ProcessCarryOverForAllEmployees(leaveTypeID uint, fromYear int, processedBy *uint) (int, int, []error) {
	var employees []models.Employee
	if err := database.DB.Find(&employees).Error; err != nil {
		return 0, 0, []error{err}
	}

	processed := 0
	skipped := 0
	var errors []error

	for _, emp := range employees {
		carryOver, err := ProcessYearEndCarryOver(emp.ID, leaveTypeID, fromYear, processedBy)
		if err != nil {
			errors = append(errors, fmt.Errorf("employee %d: %w", emp.ID, err))
			continue
		}
		if carryOver == nil {
			skipped++ // No balance to carry over
			continue
		}
		processed++
	}

	return processed, skipped, errors
}

// UpdateCarryOverUsage updates the usage of carry-over days when leave is taken
// This should be called when a leave is approved
func UpdateCarryOverUsage(employeeID uint, leaveTypeID uint, daysUsed float64) error {
	// Get all non-expired carry-overs, ordered by oldest first (FIFO)
	var carryOvers []models.LeaveCarryOver
	now := time.Now()
	if err := database.DB.Where("employee_id = ? AND leave_type_id = ? AND is_expired = ? AND days_remaining > 0",
		employeeID, leaveTypeID, false).
		Where("(expiry_date IS NULL OR expiry_date >= ?)", now).
		Order("from_year ASC, created_at ASC").
		Find(&carryOvers).Error; err != nil {
		return err
	}

	remainingDays := daysUsed
	for i := range carryOvers {
		if remainingDays <= 0 {
			break
		}

		available := carryOvers[i].DaysRemaining
		if available >= remainingDays {
			// Use all remaining days from this carry-over
			carryOvers[i].DaysUsed += remainingDays
			carryOvers[i].DaysRemaining -= remainingDays
			remainingDays = 0
		} else {
			// Use all available from this carry-over
			carryOvers[i].DaysUsed += available
			carryOvers[i].DaysRemaining = 0
			remainingDays -= available
		}

		if err := database.DB.Save(&carryOvers[i]).Error; err != nil {
			return fmt.Errorf("failed to update carry-over usage: %w", err)
		}
	}

	return nil
}

// ExpireCarryOvers marks expired carry-overs as expired
func ExpireCarryOvers() error {
	now := time.Now()
	result := database.DB.Model(&models.LeaveCarryOver{}).
		Where("is_expired = ? AND expiry_date IS NOT NULL AND expiry_date < ?", false, now).
		Update("is_expired", true)

	return result.Error
}

// GetCarryOverHistory returns carry-over history for an employee
func GetCarryOverHistory(employeeID uint, leaveTypeID uint) ([]models.LeaveCarryOver, error) {
	var carryOvers []models.LeaveCarryOver
	if err := database.DB.Where("employee_id = ? AND leave_type_id = ?", employeeID, leaveTypeID).
		Order("from_year DESC, created_at DESC").
		Find(&carryOvers).Error; err != nil {
		return nil, err
	}

	return carryOvers, nil
}
