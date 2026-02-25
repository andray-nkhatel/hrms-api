package utils

import (
	"fmt"
	"hrms-api/database"
	"hrms-api/models"
	"strings"
	"time"
)

// CalculateProjectedAnnualLeaveBalance calculates the projected annual leave balance
// at a future date, accounting for monthly accruals between now and the target date
// This uses the same calculation approach as GetCurrentLeaveBalance for consistency
func CalculateProjectedAnnualLeaveBalance(employeeID uint, leaveTypeID uint, targetDate time.Time) (float64, error) {
	now := time.Now()
	if targetDate.Before(now) || targetDate.Equal(now) {
		// If target date is today or in the past, return current balance
		return GetCurrentLeaveBalance(employeeID, leaveTypeID)
	}

	// Ensure accruals are up to date first (for consistency with GetCurrentLeaveBalance)
	var leaveType models.LeaveType
	if err := database.DB.First(&leaveType, leaveTypeID).Error; err != nil {
		return 0, err
	}

	// For leave types that use balance (e.g. Annual), use accrual-based calculation
	if leaveType.UsesBalance {
		// Calculate projected accrued by target date
		projectedAccrued, err := CalculateAnnualLeaveAccrued(employeeID, leaveTypeID, targetDate)
		if err != nil {
			return 0, err
		}

		// Get total used (only approved leaves - don't assume pending will be approved)
		var usedDays float64
		var leaves []models.Leave
		database.DB.Where("employee_id = ? AND leave_type_id = ? AND status = ?",
			employeeID, leaveTypeID, models.StatusApproved).Find(&leaves)

		for _, leave := range leaves {
			// Only count leaves that are before or on the target date
			if leave.StartDate.Before(targetDate) || leave.StartDate.Equal(targetDate) {
				usedDays += float64(leave.GetDuration())
			}
		}

		// Projected balance = projected accrued - used days
		// Allow negative balances (overdrawn) to be visible
		projectedBalance := projectedAccrued - usedDays

		// Add carry-over balance if carry-over is enabled (consistent with GetCurrentLeaveBalance)
		if leaveType.AllowCarryOver {
			carryOverBalance, err := GetCarryOverBalance(employeeID, leaveTypeID)
			if err == nil {
				projectedBalance += carryOverBalance
			}
		}

		return projectedBalance, nil
	}

	// For other leave types, use current balance (no projection needed)
	return GetCurrentLeaveBalance(employeeID, leaveTypeID)
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
// This calculates cumulative accrual across all years (not capped at 12 months)
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

	// No cap - employees accrue 2 days per month indefinitely
	// Each year they get 24 days (12 months * 2 days/month)
	return float64(months) * AnnualLeaveDaysPerMonth
}

// ProcessMonthlyAccrual processes leave accrual for a specific month
func ProcessMonthlyAccrual(employeeID uint, leaveTypeID uint, accrualMonth time.Time) error {
	// Check if already processed
	monthStart := time.Date(accrualMonth.Year(), accrualMonth.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Use Find() with Limit(1) instead of First() to avoid logging "record not found" errors
	var existingAccruals []models.LeaveAccrual
	database.DB.Where("employee_id = ? AND leave_type_id = ? AND accrual_month = ?",
		employeeID, leaveTypeID, monthStart).Limit(1).Find(&existingAccruals)
	
	var existing models.LeaveAccrual
	if len(existingAccruals) > 0 && existingAccruals[0].ID > 0 {
		existing = existingAccruals[0]
	}

	// Get previous month's balance
	prevMonth := monthStart.AddDate(0, -1, 0)
	prevBalance := 0.0
	
	// Check if there's an initial balance record for this month or earlier
	// If this month IS the initial balance month, we should NOT use previous month's balance
	// because it might be calculated from employment start date, not from the initial balance
	var initialBalanceForThisMonth []models.LeaveAccrual
	database.DB.Where("employee_id = ? AND leave_type_id = ?", employeeID, leaveTypeID).
		Where("notes IS NOT NULL AND notes != '' AND (notes LIKE '%Initial balance%' OR notes LIKE '%set-initial%' OR notes LIKE '%Set initial%')").
		Where("COALESCE(accrual_month, MAKE_DATE(year::integer, month::integer, 1)) = ?", monthStart).
		Limit(1).Find(&initialBalanceForThisMonth)
	
	// If this month has an initial balance record, don't use previous month's balance
	// The initial balance itself is the starting point
	if len(initialBalanceForThisMonth) > 0 {
		prevBalance = 0.0
	} else {
		// Not an initial balance month - use previous month's balance normally
		var prevAccruals []models.LeaveAccrual
		database.DB.Where("employee_id = ? AND leave_type_id = ? AND accrual_month = ?",
			employeeID, leaveTypeID, prevMonth).Limit(1).Find(&prevAccruals)
		if len(prevAccruals) > 0 && prevAccruals[0].ID > 0 {
			prevBalance = prevAccruals[0].DaysBalance
		}
	}

	// Calculate days used in this month from approved leaves
	// This MUST always be recalculated from actual leave records, even if accrual is already processed
	// This ensures DaysUsed stays accurate when new leaves are approved after manual adjustments
	daysUsedFromLeaves := CalculateDaysUsedInMonth(employeeID, leaveTypeID, monthStart)

	// Calculate new balance
	newAccrued := AnnualLeaveDaysPerMonth

	// Create or update accrual record
	now := time.Now()
	if existing.ID > 0 {
		// Always recalculate DaysUsed from actual leave records as the source of truth
		// This ensures DaysUsed matches actual approved leave records, even if accrual was manually processed
		daysUsed := daysUsedFromLeaves

		// Check if this is an initial balance record (set via SetInitialBalance)
		// Initial balance records should be treated specially - they set the starting balance
		// but subsequent months should still update the balance based on usage
		isInitialBalance := existing.Notes != nil && 
			(*existing.Notes != "" && (strings.Contains(*existing.Notes, "Initial balance") || 
			 strings.Contains(*existing.Notes, "set-initial") || 
			 strings.Contains(*existing.Notes, "Set initial")))

		// Calculate what the balance SHOULD be: prevBalance + newAccrued - daysUsed
		calculatedBalance := prevBalance + newAccrued - daysUsed

		// Check if DaysUsed has changed (new leaves were approved or cancelled)
		daysUsedChanged := false
		daysUsedDiff := daysUsed - existing.DaysUsed
		if daysUsedDiff > 0.01 || daysUsedDiff < -0.01 {
			daysUsedChanged = true
		}

		// Check if existing balance was manually adjusted
		// If the difference is significant (more than 0.01 to account for floating point), it was manually adjusted
		balanceDiff := existing.DaysBalance - calculatedBalance
		wasManuallyAdjusted := balanceDiff > 0.01 || balanceDiff < -0.01

		if isInitialBalance {
			// This is an initial balance record - it sets the starting balance
			// The balance should always be: originalInitialBalance - totalDaysUsedSinceInitialBalance
			// This ensures leaves are deducted from the initial balance, not recalculated from employment start
			
			// Extract the original initial balance from Notes
			// Format: "Initial balance set: X.XX days (was Y.YY). Reason: ..."
			originalInitialBalance := existing.DaysBalance
			if existing.Notes != nil && *existing.Notes != "" {
				note := *existing.Notes
				// Look for "Initial balance set: X.XX days" pattern
				if strings.Contains(note, "Initial balance set:") {
					parts := strings.Split(note, "Initial balance set:")
					if len(parts) > 1 {
						// Extract the number before "days"
						balancePart := strings.Split(parts[1], "days")[0]
						balancePart = strings.TrimSpace(balancePart)
						var parsedBalance float64
						if _, err := fmt.Sscanf(balancePart, "%f", &parsedBalance); err == nil {
							// Found the original initial balance in notes - use it
							originalInitialBalance = parsedBalance
						}
					}
				}
			}
			
			// For initial balance records, we need to calculate total days used since the initial balance was set
			// This is different from regular accruals which only track days used in that specific month
			// Get all approved leaves from the initial balance month onwards
			var totalDaysUsedSinceInitial float64
			var allApprovedLeaves []models.Leave
			database.DB.Where("employee_id = ? AND leave_type_id = ? AND status = ? AND start_date >= ?",
				employeeID, leaveTypeID, models.StatusApproved, monthStart).Find(&allApprovedLeaves)
			
			for _, leave := range allApprovedLeaves {
				totalDaysUsedSinceInitial += float64(leave.GetDuration())
			}
			
			// For initial balance records, ALWAYS calculate balance as: originalInitialBalance - totalDaysUsedSinceInitial
			// This ensures the balance is always correct, regardless of which month the leave was taken
			// The initial balance (e.g., 300) is the starting point, and we subtract all days used since then
			existing.DaysBalance = originalInitialBalance - totalDaysUsedSinceInitial
			existing.DaysAccrued = newAccrued
			existing.DaysUsed = totalDaysUsedSinceInitial // Store total days used since initial balance
		} else if daysUsedChanged {
			// DaysUsed has changed (new leaves approved/cancelled) - always recalculate balance
			// Recalculate balance from previous month's balance: prevBalance + newAccrued - daysUsed
			// This ensures the balance accurately reflects actual leave usage
			existing.DaysBalance = calculatedBalance
			existing.DaysAccrued = newAccrued
			existing.DaysUsed = daysUsed
		} else if wasManuallyAdjusted {
			// Balance was manually adjusted (not initial balance) and DaysUsed hasn't changed - preserve the balance
			// This maintains manual balance adjustments when no new leaves were processed
			existing.DaysAccrued = newAccrued
			existing.DaysUsed = daysUsed // Always use actual calculated days used from leave records
			// Keep existing.DaysBalance as is (it has manual adjustments)
		} else {
			// No manual adjustments detected and DaysUsed hasn't changed, recalculate normally
			existing.DaysAccrued = newAccrued
			existing.DaysUsed = daysUsed
			existing.DaysBalance = calculatedBalance
		}

		// Always mark as processed and update timestamp
		existing.IsProcessed = true
		existing.ProcessedAt = &now
		return database.DB.Save(&existing).Error
	}

	// New accrual - use calculated values
	daysUsed := daysUsedFromLeaves
	newBalance := prevBalance + newAccrued - daysUsed

	accrual := models.LeaveAccrual{
		EmployeeID:   employeeID,
		LeaveTypeID:  leaveTypeID,
		AccrualMonth: &monthStart,
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

// GetCurrentLeaveBalance calculates current leave balance including accruals and carry-over
// For annual leave, it uses accrual records to ensure manual adjustments are reflected
func GetCurrentLeaveBalance(employeeID uint, leaveTypeID uint) (float64, error) {
	// Get leave type to check if it's annual leave
	var leaveType models.LeaveType
	if err := database.DB.First(&leaveType, leaveTypeID).Error; err != nil {
		return 0, err
	}

	var baseBalance float64

	// For leave types that use balance (e.g. Annual), use accrual records
	if leaveType.UsesBalance {
		// Ensure accruals are up to date first
		if err := EnsureAccrualsUpToDate(employeeID, leaveTypeID); err != nil {
			return 0, err
		}

		// Get the latest accrual record which contains the current balance
		// Handle both accrual_month (full schema) and year/month (simplified schema)
		// Note: "record not found" errors are expected when no accruals exist yet
		var latestAccrual models.LeaveAccrual
		var err error

		// Try to find record with accrual_month first (full schema)
		err = database.DB.Where("employee_id = ? AND leave_type_id = ? AND accrual_month IS NOT NULL", employeeID, leaveTypeID).
			Order("accrual_month DESC").
			First(&latestAccrual).Error

		// If no record with accrual_month, try year/month schema (simplified)
		if err != nil {
			err = database.DB.Where("employee_id = ? AND leave_type_id = ? AND year > 0 AND month > 0", employeeID, leaveTypeID).
				Order("year DESC, month DESC").
				First(&latestAccrual).Error
		}

		if err == nil {
			// Use the balance from the latest accrual record (includes manual adjustments)
			baseBalance = latestAccrual.DaysBalance
		} else {
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

			// Allow negative balances (overdrawn) to be visible
			baseBalance = accrued - usedDays
		}
	} else {
		// For other leave types, use the original calculation
		balance, err := CalculateLeaveBalance(employeeID, leaveTypeID)
		if err != nil {
			return 0, err
		}
		baseBalance = float64(balance)
	}

	// Add carry-over balance if carry-over is enabled
	if leaveType.AllowCarryOver {
		carryOverBalance, err := GetCarryOverBalance(employeeID, leaveTypeID)
		if err != nil {
			// Log error but don't fail - carry-over is optional
			carryOverBalance = 0
		}
		baseBalance += carryOverBalance
	}

	return baseBalance, nil
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
	} else {
		// If no employment details, try to get from employee created_at
		var employee models.Employee
		if err := database.DB.First(&employee, employeeID).Error; err == nil {
			startDate = employee.CreatedAt
		}
	}

	// Check if there's an initial balance record - if so, use it as the starting point
	// This ensures we don't recalculate balances from employment start when an initial balance was set
	var initialBalanceRecord models.LeaveAccrual
	var initialBalanceMonth *time.Time
	var hasInitialBalance bool
	
	// Find the earliest initial balance record (identified by Notes containing "Initial balance")
	var allAccruals []models.LeaveAccrual
	database.DB.Where("employee_id = ? AND leave_type_id = ?", employeeID, leaveTypeID).
		Where("notes IS NOT NULL AND notes != '' AND (notes LIKE '%Initial balance%' OR notes LIKE '%set-initial%' OR notes LIKE '%Set initial%')").
		Order("COALESCE(accrual_month, MAKE_DATE(year::integer, month::integer, 1)) ASC").
		Find(&allAccruals)
	
	if len(allAccruals) > 0 {
		initialBalanceRecord = allAccruals[0]
		if initialBalanceRecord.AccrualMonth != nil {
			initialBalanceMonth = initialBalanceRecord.AccrualMonth
			hasInitialBalance = true
		} else if initialBalanceRecord.Year > 0 && initialBalanceRecord.Month > 0 {
			monthStart := time.Date(initialBalanceRecord.Year, time.Month(initialBalanceRecord.Month), 1, 0, 0, 0, 0, time.UTC)
			initialBalanceMonth = &monthStart
			hasInitialBalance = true
		}
	}

	// Process accruals from start date to current month
	currentMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	processMonth := time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Start from the month after employment (first accrual happens at end of first month)
	processMonth = processMonth.AddDate(0, 1, 0)

	// If there's an initial balance, only process months from the initial balance month forward
	// This prevents recalculating balances from employment start when an initial balance was set
	if hasInitialBalance && initialBalanceMonth != nil {
		// Start processing from the initial balance month (or the month after if it's in the past)
		if processMonth.Before(*initialBalanceMonth) {
			processMonth = *initialBalanceMonth
		}
	}

	for !processMonth.After(currentMonth) {
		if err := ProcessMonthlyAccrual(employeeID, leaveTypeID, processMonth); err != nil {
			return err
		}
		processMonth = processMonth.AddDate(0, 1, 0)
	}

	return nil
}

// GetAvailableLeaveBalance calculates the available leave balance accounting for pending leaves
// This is useful for approval checks to ensure we don't approve more than available
// For future-dated leaves, it uses projected balance; for current/past-dated, it uses current balance
func GetAvailableLeaveBalance(employeeID uint, leaveTypeID uint, excludeLeaveID *uint, targetDate *time.Time) (float64, error) {
	var balance float64
	var err error

	// For future-dated annual leave, use projected balance
	if targetDate != nil && targetDate.After(time.Now()) {
		// Get projected balance at target date
		// This already accounts for all pending leaves, so we need to add back the excluded leave
		projectedBalance, err := CalculateProjectedAnnualLeaveBalance(employeeID, leaveTypeID, *targetDate)
		if err != nil {
			return 0, err
		}

		// The projected balance calculation includes ALL pending leaves (including the excluded one)
		// So we need to add back the excluded leave's duration to get the balance available for it
		if excludeLeaveID != nil {
			var excludedLeave models.Leave
			if err := database.DB.First(&excludedLeave, *excludeLeaveID).Error; err == nil {
				// Add back the excluded leave's duration since it was already subtracted in projected balance
				projectedBalance += float64(excludedLeave.GetDuration())
			}
		}

		balance = projectedBalance
		// Don't subtract other pending leaves here - projected balance already accounts for them
	} else {
		// For current/past-dated leaves, use current balance and subtract other pending leaves
		balance, err = GetCurrentLeaveBalance(employeeID, leaveTypeID)
		if err != nil {
			return 0, err
		}

		// Subtract other pending leaves (excluding the one being approved if specified)
		var pendingLeaves []models.Leave
		query := database.DB.Where("employee_id = ? AND leave_type_id = ? AND status = ?",
			employeeID, leaveTypeID, models.StatusPending)

		if excludeLeaveID != nil {
			query = query.Where("id != ?", *excludeLeaveID)
		}

		query.Find(&pendingLeaves)

		// Subtract pending leave durations from available balance
		for _, leave := range pendingLeaves {
			balance -= float64(leave.GetDuration())
		}
	}

	// Allow negative balances (overdrawn) to be visible
	return balance, nil
}

// GetCurrentYearLeaveBalance calculates the current year's leave balance from the 24 days annual entitlement
// This shows only the balance from the current year, not cumulative all-time balance
func GetCurrentYearLeaveBalance(employeeID uint, leaveTypeID uint) (float64, error) {
	// Get leave type to check if it's annual leave
	var leaveType models.LeaveType
	if err := database.DB.First(&leaveType, leaveTypeID).Error; err != nil {
		return 0, err
	}

	// Only applicable to leave types that use balance (e.g. Annual)
	if !leaveType.UsesBalance {
		return 0, fmt.Errorf("this function is only for leave types that use balance")
	}

	// Ensure accruals are up to date
	if err := EnsureAccrualsUpToDate(employeeID, leaveTypeID); err != nil {
		return 0, err
	}

	// Get current year's start date
	now := time.Now()
	currentYearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)

	// Get accruals for the current year only
	// Handle both accrual_month and year/month schemas
	var currentYearAccruals []models.LeaveAccrual
	database.DB.Where("employee_id = ? AND leave_type_id = ? AND (accrual_month >= ? OR (year >= ? AND accrual_month IS NULL))",
		employeeID, leaveTypeID, currentYearStart, currentYearStart.Year()).
		Order("COALESCE(accrual_month, MAKE_DATE(year, month, 1)) ASC").
		Find(&currentYearAccruals)

	// Calculate days accrued and used in current year
	var yearAccrued, yearUsed float64
	for _, acc := range currentYearAccruals {
		yearAccrued += acc.DaysAccrued
		yearUsed += acc.DaysUsed
	}

	// Current year balance = Days Accrued This Year - Days Used This Year
	// Cap current year accrual at 24 days (annual entitlement)
	// Note: Balance can exceed 24 if carry-over is added, but current year accrual is capped
	if yearAccrued > AnnualLeaveDaysPerYear {
		yearAccrued = AnnualLeaveDaysPerYear
	}
	currentYearBalance := yearAccrued - yearUsed
	// Allow negative balances (overdrawn) to be visible

	// Add carry-over balance from previous years (if carry-over is enabled)
	var carryOverBalance float64
	if leaveType.AllowCarryOver {
		var err error
		carryOverBalance, err = GetCarryOverBalance(employeeID, leaveTypeID)
		if err != nil {
			// Log error but don't fail - carry-over is optional
			carryOverBalance = 0
		}
	}

	// Total balance = Current Year Balance + Carry-Over Balance
	// Allow negative balances (overdrawn) to be visible
	balance := currentYearBalance + carryOverBalance
	return balance, nil
}
