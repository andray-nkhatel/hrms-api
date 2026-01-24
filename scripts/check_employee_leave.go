package main

import (
	"fmt"
	"hrms-api/config"
	"hrms-api/database"
	"hrms-api/models"
	"hrms-api/utils"
	"log"
	"strings"
	"time"
)

func main() {
	// Load configuration
	if err := config.LoadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	// Connect to database
	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Search for employee by name
	employeeName := "Randy Orton"
	nameParts := strings.Fields(employeeName)
	
	var employee models.Employee
	var err error
	
	if len(nameParts) >= 2 {
		// Try to find by firstname and lastname
		err = database.DB.Where("LOWER(firstname) = LOWER(?) AND LOWER(lastname) = LOWER(?)", 
			nameParts[0], strings.Join(nameParts[1:], " ")).First(&employee).Error
		if err != nil {
			// Try matching by full name
			err = database.DB.Where("LOWER(CONCAT(firstname, ' ', lastname)) = LOWER(?)", employeeName).
				First(&employee).Error
		}
		if err != nil {
			// Try partial match
			err = database.DB.Where("LOWER(firstname) LIKE LOWER(?) AND LOWER(lastname) LIKE LOWER(?)", 
				"%"+nameParts[0]+"%", "%"+strings.Join(nameParts[1:], " ")+"%").First(&employee).Error
		}
	} else {
		// Single name - try firstname or lastname
		err = database.DB.Where("LOWER(firstname) = LOWER(?) OR LOWER(lastname) = LOWER(?)", 
			employeeName, employeeName).First(&employee).Error
	}
	
	if err != nil {
		fmt.Printf("Employee '%s' not found. Listing all employees:\n\n", employeeName)
		var allEmployees []models.Employee
		database.DB.Where("role != ?", models.RoleAdmin).Find(&allEmployees)
		
		if len(allEmployees) == 0 {
			log.Fatalf("No employees found in database")
		}
		
		fmt.Printf("%-5s %-20s %-15s %-10s\n", "ID", "Name", "Department", "Role")
		fmt.Println(strings.Repeat("-", 60))
		for _, emp := range allEmployees {
			fmt.Printf("%-5d %-20s %-15s %-10s\n", 
				emp.ID, 
				emp.Firstname+" "+emp.Lastname,
				emp.Department,
				emp.Role)
		}
		fmt.Println(strings.Repeat("=", 60))
		log.Fatalf("\nPlease use one of the employee names above or provide the correct name")
	}

	fmt.Printf("Found Employee: %s %s (ID: %d)\n", employee.Firstname, employee.Lastname, employee.ID)
	fmt.Printf("Department: %s\n", employee.Department)
	fmt.Printf("Role: %s\n", employee.Role)
	fmt.Println(strings.Repeat("=", 60))

	// Get Annual leave type
	var annualLeaveType models.LeaveType
	if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
		log.Fatalf("Annual leave type not found: %v", err)
	}

	fmt.Printf("Annual Leave Type ID: %d\n", annualLeaveType.ID)
	fmt.Println(strings.Repeat("-", 60))

	// Ensure accruals are up to date
	if err := utils.EnsureAccrualsUpToDate(employee.ID, annualLeaveType.ID); err != nil {
		log.Printf("Warning: Failed to process accruals: %v", err)
	}

	// Get all accruals
	var accruals []models.LeaveAccrual
	database.DB.Where("employee_id = ? AND leave_type_id = ?", employee.ID, annualLeaveType.ID).
		Order("COALESCE(accrual_month, MAKE_DATE(year::integer, month::integer, 1)) DESC, year DESC, month DESC").
		Find(&accruals)

	// Get employee start date
	var employeeStartDate time.Time
	var employment models.EmploymentDetails
	if err := database.DB.Where("employee_id = ?", employee.ID).First(&employment).Error; err == nil {
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

	fmt.Printf("Employment Start Date: %s\n", employeeStartDate.Format("2006-01-02"))
	fmt.Println(strings.Repeat("-", 60))

	// Calculate totals
	var totalAccrued float64
	for _, acc := range accruals {
		var accrualMonth time.Time
		if acc.AccrualMonth != nil {
			accrualMonth = *acc.AccrualMonth
		} else if acc.Year > 0 && acc.Month > 0 {
			accrualMonth = time.Date(acc.Year, time.Month(acc.Month), 1, 0, 0, 0, 0, time.UTC)
		}

		// Skip regular accruals in the first month of employment
		isInitialBalance := acc.Notes != nil && 
			(*acc.Notes != "" && (strings.Contains(*acc.Notes, "Initial balance") || 
			 strings.Contains(*acc.Notes, "set-initial") || 
			 strings.Contains(*acc.Notes, "Set initial")))
		
		if !accrualMonth.IsZero() && accrualMonth.Equal(firstMonthStart) && !isInitialBalance {
			continue
		}

		totalAccrued += acc.DaysAccrued
	}

	// Calculate total used from approved leaves
	var totalUsed float64
	var approvedLeaves []models.Leave
	database.DB.Where("employee_id = ? AND leave_type_id = ? AND status = ?",
		employee.ID, annualLeaveType.ID, models.StatusApproved).Find(&approvedLeaves)
	for _, leave := range approvedLeaves {
		totalUsed += float64(leave.GetDuration())
	}

	// Get carry-over balance
	var carryOverBalance float64
	if annualLeaveType.AllowCarryOver {
		carryOverBalance, _ = utils.GetCarryOverBalance(employee.ID, annualLeaveType.ID)
	}

	// Get current balance
	currentBalance, _ := utils.GetCurrentLeaveBalance(employee.ID, annualLeaveType.ID)

	// Calculate all-time net balance
	allTimeNetBalance := totalAccrued - totalUsed

	fmt.Println("\n📊 ANNUAL LEAVE SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total Accrued:        %.2f days\n", totalAccrued)
	fmt.Printf("Total Used:           %.2f days\n", totalUsed)
	fmt.Printf("All-Time Net Balance: %.2f days\n", allTimeNetBalance)
	fmt.Printf("Current Balance:      %.2f days\n", currentBalance)
	if carryOverBalance > 0 {
		fmt.Printf("Carry-Over Balance:   %.2f days\n", carryOverBalance)
	}
	fmt.Println(strings.Repeat("=", 60))

	// Show recent accruals
	fmt.Println("\n📅 RECENT ACCRUAL RECORDS (Last 12 months)")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-12s %-12s %-12s %-12s %-12s\n", "Month", "Accrued", "Used", "Balance", "Notes")
	fmt.Println(strings.Repeat("-", 60))
	
	count := 0
	for _, acc := range accruals {
		if count >= 12 {
			break
		}
		var monthStr string
		if acc.AccrualMonth != nil {
			monthStr = acc.AccrualMonth.Format("2006-01")
		} else if acc.Year > 0 && acc.Month > 0 {
			monthStr = fmt.Sprintf("%d-%02d", acc.Year, acc.Month)
		} else {
			monthStr = "N/A"
		}
		
		notes := ""
		if acc.Notes != nil && *acc.Notes != "" {
			notes = *acc.Notes
			if len(notes) > 20 {
				notes = notes[:20] + "..."
			}
		}
		
		fmt.Printf("%-12s %-12.2f %-12.2f %-12.2f %-12s\n", 
			monthStr, acc.DaysAccrued, acc.DaysUsed, acc.DaysBalance, notes)
		count++
	}

	// Show approved leaves
	fmt.Println("\n📋 APPROVED LEAVES")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("%-12s %-12s %-12s %-12s\n", "Start Date", "End Date", "Duration", "Reason")
	fmt.Println(strings.Repeat("-", 60))
	
	if len(approvedLeaves) == 0 {
		fmt.Println("No approved leaves found")
	} else {
		for _, leave := range approvedLeaves {
			reason := leave.Reason
			if len(reason) > 20 {
				reason = reason[:20] + "..."
			}
			fmt.Printf("%-12s %-12s %-12d %-12s\n",
				leave.StartDate.Format("2006-01-02"),
				leave.EndDate.Format("2006-01-02"),
				leave.GetDuration(),
				reason)
		}
	}
	fmt.Println(strings.Repeat("=", 60))
}
