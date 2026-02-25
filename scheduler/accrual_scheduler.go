package scheduler

import (
	"fmt"
	"hrms-api/database"
	"hrms-api/models"
	"hrms-api/utils"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

var cronScheduler *cron.Cron

// StartAccrualScheduler starts the automatic monthly accrual processing scheduler
// It runs on the 1st of each month at 2:00 AM to process accruals for the previous month
func StartAccrualScheduler() {
	// Create a new cron scheduler with seconds precision
	cronScheduler = cron.New(cron.WithSeconds())

	// Schedule to run on the 1st of each month at 2:00 AM
	// Cron expression: "0 0 2 1 * *" means: second=0, minute=0, hour=2, day=1, month=*, weekday=*
	_, err := cronScheduler.AddFunc("0 0 2 1 * *", processMonthlyAccruals)
	if err != nil {
		log.Printf("Failed to schedule accrual processing: %v", err)
		return
	}

	// Start the scheduler
	cronScheduler.Start()
	log.Println("✅ Automatic accrual scheduler started - will process accruals on the 1st of each month at 2:00 AM")

	// Also check if we need to process the current month on startup
	// This handles cases where the server was down on the 1st
	go checkAndProcessPendingAccruals()
}

// StopAccrualScheduler stops the accrual scheduler
func StopAccrualScheduler() {
	if cronScheduler != nil {
		cronScheduler.Stop()
		log.Println("Accrual scheduler stopped")
	}
}

// processMonthlyAccruals processes accruals for the previous month
// This is called automatically on the 1st of each month
func processMonthlyAccruals() {
	log.Println("🔄 Starting automatic monthly accrual processing...")

	// Process accruals for the previous month
	now := time.Now()
	previousMonth := now.AddDate(0, -1, 0)
	processMonth := time.Date(previousMonth.Year(), previousMonth.Month(), 1, 0, 0, 0, 0, time.UTC)

	log.Printf("Processing accruals for month: %s", processMonth.Format("2006-01"))

	// Get all leave types that use balance (e.g. Annual)
	var balanceLeaveTypes []models.LeaveType
	if err := database.DB.Where("uses_balance = ?", true).Find(&balanceLeaveTypes).Error; err != nil || len(balanceLeaveTypes) == 0 {
		log.Printf("❌ No leave types with uses_balance=true found")
		return
	}

	// Get all active employees (exclude admins and inactive)
	var employees []models.Employee
	if err := database.DB.Where("role != ? AND status = ?", models.RoleAdmin, "active").Find(&employees).Error; err != nil {
		log.Printf("❌ Error fetching employees: %v", err)
		return
	}

	processed := 0
	errors := 0
	var errorDetails []string

	for _, leaveType := range balanceLeaveTypes {
		for _, emp := range employees {
			if err := utils.ProcessMonthlyAccrualSimple(emp.ID, leaveType.ID, previousMonth.Year(), int(previousMonth.Month())); err != nil {
				errors++
				errorDetails = append(errorDetails, fmt.Sprintf("Employee %d (%s %s) %s: %v", emp.ID, emp.Firstname, emp.Lastname, leaveType.Name, err))
				log.Printf("⚠️  Failed to process accrual for employee %d (%s %s) %s: %v", emp.ID, emp.Firstname, emp.Lastname, leaveType.Name, err)
				continue
			}
			processed++
		}
	}

	log.Printf("✅ Accrual processing completed: %d processed, %d errors", processed, errors)
	if len(errorDetails) > 0 {
		log.Printf("Error details: %v", errorDetails)
	}
}

// checkAndProcessPendingAccruals checks if there are any pending accruals that need to be processed
// This runs on server startup to catch up on any missed accruals
func checkAndProcessPendingAccruals() {
	// Wait a bit for the server to fully start
	time.Sleep(5 * time.Second)

	log.Println("🔍 Checking for pending accruals...")

	var balanceLeaveTypes []models.LeaveType
	if err := database.DB.Where("uses_balance = ?", true).Find(&balanceLeaveTypes).Error; err != nil || len(balanceLeaveTypes) == 0 {
		log.Printf("⚠️  No leave types with uses_balance=true found")
		return
	}

	var employees []models.Employee
	if err := database.DB.Where("role != ? AND status = ?", models.RoleAdmin, "active").Find(&employees).Error; err != nil {
		log.Printf("⚠️  Could not check pending accruals: Error fetching employees")
		return
	}

	now := time.Now()
	previousMonth := now.AddDate(0, -1, 0)
	prevMonthStart := time.Date(previousMonth.Year(), previousMonth.Month(), 1, 0, 0, 0, 0, time.UTC)

	needsProcessing := false
	for _, leaveType := range balanceLeaveTypes {
		for _, emp := range employees {
			var accruals []models.LeaveAccrual
			database.DB.Where("employee_id = ? AND leave_type_id = ? AND year = ? AND month = ?",
				emp.ID, leaveType.ID, previousMonth.Year(), int(previousMonth.Month())).Limit(1).Find(&accruals)
			if len(accruals) == 0 || accruals[0].ID == 0 {
				needsProcessing = true
				break
			}
		}
		if needsProcessing {
			break
		}
	}

	if needsProcessing {
		log.Printf("📅 Found pending accruals for %s - processing now...", prevMonthStart.Format("2006-01"))
		processed := 0
		for _, leaveType := range balanceLeaveTypes {
			for _, emp := range employees {
				if err := utils.ProcessMonthlyAccrualSimple(emp.ID, leaveType.ID, previousMonth.Year(), int(previousMonth.Month())); err != nil {
					log.Printf("⚠️  Failed to process accrual for employee %d: %v", emp.ID, err)
					continue
				}
				processed++
			}
		}
		log.Printf("✅ Processed %d pending accruals for %s", processed, prevMonthStart.Format("2006-01"))
	} else {
		log.Println("✅ No pending accruals found - all up to date")
	}
}
