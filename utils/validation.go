package utils

import (
	"hrms-api/database"
	"hrms-api/models"
	"time"
)

// ValidateLeaveDates checks if start date is before end date
func ValidateLeaveDates(startDate, endDate time.Time) error {
	if startDate.After(endDate) {
		return ErrInvalidDateRange
	}
	if startDate.Before(time.Now().Truncate(24 * time.Hour)) {
		return ErrPastDate
	}
	return nil
}

// CheckOverlappingLeaves checks if the employee has any overlapping approved or pending leaves
func CheckOverlappingLeaves(employeeID uint, startDate, endDate time.Time, excludeLeaveID *uint) (bool, error) {
	var count int64
	query := database.DB.Model(&models.Leave{}).
		Where("employee_id = ?", employeeID).
		Where("status IN ?", []models.LeaveStatus{models.StatusPending, models.StatusApproved}).
		Where("(start_date <= ? AND end_date >= ?) OR (start_date <= ? AND end_date >= ?) OR (start_date >= ? AND end_date <= ?)",
			endDate, startDate, startDate, endDate, startDate, endDate)

	if excludeLeaveID != nil {
		query = query.Where("id != ?", *excludeLeaveID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// CalculateLeaveBalance calculates the remaining leave balance for an employee
func CalculateLeaveBalance(employeeID uint, leaveTypeID uint) (int, error) {
	var leaveType models.LeaveType
	if err := database.DB.First(&leaveType, leaveTypeID).Error; err != nil {
		return 0, err
	}

	var usedDays int
	var leaves []models.Leave
	if err := database.DB.Where("employee_id = ? AND leave_type_id = ? AND status = ?",
		employeeID, leaveTypeID, models.StatusApproved).Find(&leaves).Error; err != nil {
		return 0, err
	}

	for _, leave := range leaves {
		usedDays += leave.GetDuration()
	}

	balance := leaveType.MaxDays - usedDays
	if balance < 0 {
		balance = 0
	}

	return balance, nil
}

