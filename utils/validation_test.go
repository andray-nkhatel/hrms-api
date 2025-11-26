package utils

import (
	"hrms-api/config"
	"hrms-api/database"
	"hrms-api/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupValidationTestDB(t *testing.T) {
	config.LoadConfig()
	config.AppConfig.DBName = "hrms_test_db"

	if err := database.Connect(); err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := database.Migrate(); err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	database.DB.Exec("TRUNCATE TABLE leaves, employees, leave_types CASCADE")
}

func TestValidateLeaveDates(t *testing.T) {
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.AddDate(0, 0, 1)
	yesterday := today.AddDate(0, 0, -1)

	// Valid dates
	err := ValidateLeaveDates(tomorrow, tomorrow.AddDate(0, 0, 5))
	assert.NoError(t, err)

	// Invalid: start after end
	err = ValidateLeaveDates(tomorrow.AddDate(0, 0, 5), tomorrow)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidDateRange, err)

	// Invalid: past date
	err = ValidateLeaveDates(yesterday, tomorrow)
	assert.Error(t, err)
	assert.Equal(t, ErrPastDate, err)
}

func TestCheckOverlappingLeaves(t *testing.T) {
	setupValidationTestDB(t)
	defer database.DB.Exec("TRUNCATE TABLE leaves, employees, leave_types CASCADE")

	// Create test data
	employee := models.Employee{
		NRC:          "123456/78/9",
		Firstname:    "Test",
		Lastname:     "User",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Role:         models.RoleEmployee,
	}
	database.DB.Create(&employee)

	leaveType := models.LeaveType{Name: "Annual", MaxDays: 20}
	database.DB.Create(&leaveType)

	startDate := time.Now().AddDate(0, 0, 10).Truncate(24 * time.Hour)
	endDate := startDate.AddDate(0, 0, 5)

	// Create existing leave
	existingLeave := models.Leave{
		EmployeeID:  employee.ID,
		LeaveTypeID: leaveType.ID,
		StartDate:   startDate.AddDate(0, 0, 2),
		EndDate:     startDate.AddDate(0, 0, 7),
		Status:      models.StatusApproved,
	}
	database.DB.Create(&existingLeave)

	// Test overlapping leave
	hasOverlap, err := CheckOverlappingLeaves(employee.ID, startDate, endDate, nil)
	assert.NoError(t, err)
	assert.True(t, hasOverlap)

	// Test non-overlapping leave
	nonOverlapStart := startDate.AddDate(0, 0, 10)
	nonOverlapEnd := nonOverlapStart.AddDate(0, 0, 3)
	hasOverlap, err = CheckOverlappingLeaves(employee.ID, nonOverlapStart, nonOverlapEnd, nil)
	assert.NoError(t, err)
	assert.False(t, hasOverlap)

	// Test excluding current leave
	hasOverlap, err = CheckOverlappingLeaves(employee.ID, startDate, endDate, &existingLeave.ID)
	assert.NoError(t, err)
	assert.False(t, hasOverlap)
}

func TestCalculateLeaveBalance(t *testing.T) {
	setupValidationTestDB(t)
	defer database.DB.Exec("TRUNCATE TABLE leaves, employees, leave_types CASCADE")

	// Create test data
	employee := models.Employee{
		NRC:          "123456/78/9",
		Firstname:    "Test",
		Lastname:     "User",
		Email:        "test@example.com",
		PasswordHash: "hash",
		Role:         models.RoleEmployee,
	}
	database.DB.Create(&employee)

	leaveType := models.LeaveType{Name: "Annual", MaxDays: 20}
	database.DB.Create(&leaveType)

	// Create approved leaves
	startDate := time.Now().AddDate(0, 0, 10).Truncate(24 * time.Hour)
	leave1 := models.Leave{
		EmployeeID:  employee.ID,
		LeaveTypeID: leaveType.ID,
		StartDate:   startDate,
		EndDate:     startDate.AddDate(0, 0, 4), // 5 days
		Status:      models.StatusApproved,
	}
	database.DB.Create(&leave1)

	leave2 := models.Leave{
		EmployeeID:  employee.ID,
		LeaveTypeID: leaveType.ID,
		StartDate:   startDate.AddDate(0, 0, 10),
		EndDate:     startDate.AddDate(0, 0, 12), // 3 days
		Status:      models.StatusApproved,
	}
	database.DB.Create(&leave2)

	// Pending leave should not count
	leave3 := models.Leave{
		EmployeeID:  employee.ID,
		LeaveTypeID: leaveType.ID,
		StartDate:   startDate.AddDate(0, 0, 20),
		EndDate:     startDate.AddDate(0, 0, 22), // 3 days
		Status:      models.StatusPending,
	}
	database.DB.Create(&leave3)

	balance, err := CalculateLeaveBalance(employee.ID, leaveType.ID)
	assert.NoError(t, err)
	assert.Equal(t, 12, balance) // 20 - 5 - 3 = 12
}
