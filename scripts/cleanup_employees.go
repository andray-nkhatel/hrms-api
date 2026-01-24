package main

import (
	"fmt"
	"hrms-api/config"
	"hrms-api/database"
	"hrms-api/models"
	"log"
	"os"
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

	// Confirm action
	fmt.Println("⚠️  WARNING: This will delete ALL employees except admins!")
	fmt.Println("This will also delete all related data (leaves, accruals, etc.) for non-admin employees.")
	fmt.Print("Type 'DELETE' to confirm: ")

	var confirmation string
	fmt.Scanln(&confirmation)

	if confirmation != "DELETE" {
		fmt.Println("Operation cancelled.")
		os.Exit(0)
	}

	// Get all non-admin employee IDs
	var nonAdminEmployeeIDs []uint
	if err := database.DB.Model(&models.Employee{}).
		Where("role != ?", models.RoleAdmin).
		Pluck("id", &nonAdminEmployeeIDs).Error; err != nil {
		log.Fatalf("Failed to get employee IDs: %v", err)
	}

	if len(nonAdminEmployeeIDs) == 0 {
		fmt.Println("No non-admin employees found. Nothing to delete.")
		os.Exit(0)
	}

	fmt.Printf("Found %d non-admin employees to delete.\n", len(nonAdminEmployeeIDs))

	// Start transaction
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Fatalf("Transaction rolled back due to error: %v", r)
		}
	}()

	// Delete related data for non-admin employees
	fmt.Println("Deleting related data...")

	// Delete leaves
	if err := tx.Where("employee_id IN ?", nonAdminEmployeeIDs).Delete(&models.Leave{}).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete leaves: %v", err)
	}
	fmt.Println("  ✓ Deleted leaves")

	// Delete leave accruals
	if err := tx.Where("employee_id IN ?", nonAdminEmployeeIDs).Delete(&models.LeaveAccrual{}).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete leave accruals: %v", err)
	}
	fmt.Println("  ✓ Deleted leave accruals")

	// Delete leave taken
	if err := tx.Where("employee_id IN ?", nonAdminEmployeeIDs).Delete(&models.LeaveTaken{}).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete leave taken: %v", err)
	}
	fmt.Println("  ✓ Deleted leave taken records")

	// Delete leave carry over
	if err := tx.Where("employee_id IN ?", nonAdminEmployeeIDs).Delete(&models.LeaveCarryOver{}).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete leave carry over: %v", err)
	}
	fmt.Println("  ✓ Deleted leave carry over records")

	// Delete identity information
	if err := tx.Where("employee_id IN ?", nonAdminEmployeeIDs).Delete(&models.IdentityInformation{}).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete identity information: %v", err)
	}
	fmt.Println("  ✓ Deleted identity information")

	// Delete employment details
	if err := tx.Where("employee_id IN ?", nonAdminEmployeeIDs).Delete(&models.EmploymentDetails{}).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete employment details: %v", err)
	}
	fmt.Println("  ✓ Deleted employment details")

	// Delete employment history
	if err := tx.Where("employee_id IN ?", nonAdminEmployeeIDs).Delete(&models.EmploymentHistory{}).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete employment history: %v", err)
	}
	fmt.Println("  ✓ Deleted employment history")

	// Delete position assignments
	if err := tx.Where("employee_id IN ?", nonAdminEmployeeIDs).Delete(&models.PositionAssignment{}).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete position assignments: %v", err)
	}
	fmt.Println("  ✓ Deleted position assignments")

	// Delete documents
	if err := tx.Where("employee_id IN ?", nonAdminEmployeeIDs).Delete(&models.Document{}).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete documents: %v", err)
	}
	fmt.Println("  ✓ Deleted documents")

	// Delete work lifecycle events
	if err := tx.Where("employee_id IN ?", nonAdminEmployeeIDs).Delete(&models.WorkLifecycleEvent{}).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete work lifecycle events: %v", err)
	}
	fmt.Println("  ✓ Deleted work lifecycle events")

	// Delete onboarding processes
	if err := tx.Where("employee_id IN ?", nonAdminEmployeeIDs).Delete(&models.OnboardingProcess{}).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete onboarding processes: %v", err)
	}
	fmt.Println("  ✓ Deleted onboarding processes")

	// Delete offboarding processes
	if err := tx.Where("employee_id IN ?", nonAdminEmployeeIDs).Delete(&models.OffboardingProcess{}).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete offboarding processes: %v", err)
	}
	fmt.Println("  ✓ Deleted offboarding processes")

	// Delete compliance records
	if err := tx.Where("employee_id IN ?", nonAdminEmployeeIDs).Delete(&models.ComplianceRecord{}).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete compliance records: %v", err)
	}
	fmt.Println("  ✓ Deleted compliance records")

	// Delete audit logs (where performed_by is a non-admin employee)
	if err := tx.Where("performed_by IN ?", nonAdminEmployeeIDs).Delete(&models.AuditLog{}).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete audit logs: %v", err)
	}
	fmt.Println("  ✓ Deleted audit logs")

	// Delete leave audits
	if err := tx.Where("employee_id IN ?", nonAdminEmployeeIDs).Delete(&models.LeaveAudit{}).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete leave audits: %v", err)
	}
	fmt.Println("  ✓ Deleted leave audits")

	// Finally, delete the employees themselves
	fmt.Println("Deleting employees...")
	if err := tx.Where("role != ?", models.RoleAdmin).Delete(&models.Employee{}).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete employees: %v", err)
	}
	fmt.Println("  ✓ Deleted employees")

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	// Verify admin still exists
	var adminCount int64
	if err := database.DB.Model(&models.Employee{}).
		Where("role = ?", models.RoleAdmin).
		Count(&adminCount).Error; err != nil {
		log.Fatalf("Failed to verify admin count: %v", err)
	}

	fmt.Printf("\n✅ Successfully deleted all non-admin employees and related data.\n")
	fmt.Printf("✅ %d admin user(s) remain in the database.\n", adminCount)
}
