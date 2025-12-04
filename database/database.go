package database

import (
	"hrms-api/config"
	"hrms-api/models"
	"log"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() error {
	var err error

	dsn := config.AppConfig.GetDSN()
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return err
	}

	log.Println("Database connected successfully")
	return nil
}

func Migrate() error {
	err := DB.AutoMigrate(
		// Core models
		&models.Employee{},
		&models.LeaveType{},
		&models.Leave{},
		&models.LeaveAudit{},
		&models.LeaveAccrual{},
		// Core HR models
		&models.IdentityInformation{},
		&models.EmploymentDetails{},
		&models.EmploymentHistory{},
		&models.Position{},
		&models.PositionAssignment{},
		&models.Document{},
		&models.WorkLifecycleEvent{},
		&models.OnboardingProcess{},
		&models.OnboardingTask{},
		&models.OffboardingProcess{},
		&models.OffboardingTask{},
		&models.ComplianceRequirement{},
		&models.ComplianceRecord{},
		&models.AuditLog{},
	)

	if err != nil {
		return err
	}

	log.Println("Database migration completed")
	return nil
}

func SeedData() error {
	// Seed Leave Types (only if they don't exist)
	var leaveTypeCount int64
	DB.Model(&models.LeaveType{}).Count(&leaveTypeCount)
	if leaveTypeCount == 0 {
		leaveTypes := []models.LeaveType{
			{Name: "Sick", MaxDays: 3},
			{Name: "Compassionate", MaxDays: 7},
			{Name: "Annual", MaxDays: 24}, // 24 days/year, accrues 2 days/month
			{Name: "Maternity", MaxDays: 90},
			{Name: "Paternity", MaxDays: 7},
		}

		for _, lt := range leaveTypes {
			if err := DB.Create(&lt).Error; err != nil {
				return err
			}
		}
		log.Println("Leave types seeded")
	}

	// Default password for all test users: "password123"
	defaultPassword := "password123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Helper for string pointers
	strPtr := func(s string) *string { return &s }

	// Seed Test Employees (only if they don't exist)
	var employeeCount int64
	DB.Model(&models.Employee{}).Count(&employeeCount)
	if employeeCount == 0 {
		testEmployees := []models.Employee{
			{
				NRC:          strPtr("123456/78/9"),
				Firstname:    "John",
				Lastname:     "Doe",
				Email:        "john.doe@example.com",
				PasswordHash: string(hashedPassword),
				Department:   "IT",
				Role:         models.RoleEmployee,
			},
			{
				NRC:          strPtr("987654/32/1"),
				Firstname:    "Jane",
				Lastname:     "Manager",
				Email:        "jane.manager@example.com",
				PasswordHash: string(hashedPassword),
				Department:   "HR",
				Role:         models.RoleManager,
			},
		}

		for _, emp := range testEmployees {
			if err := DB.Create(&emp).Error; err != nil {
				return err
			}
		}

		log.Println("Test accounts created:")
		log.Println("  Employee: NRC=123456/78/9, Password=password123")
		log.Println("  Manager:  NRC=987654/32/1, Password=password123")
	}

	// Always ensure admin user exists with correct password
	var adminUser models.Employee
	adminUsername := "admin"
	if err := DB.Where("username = ? AND role = ?", adminUsername, models.RoleAdmin).First(&adminUser).Error; err != nil {
		// Admin doesn't exist, create it
		adminUser = models.Employee{
			Username:     strPtr(adminUsername),
			Firstname:    "Admin",
			Lastname:     "User",
			Email:        "admin@example.com",
			PasswordHash: string(hashedPassword),
			Department:   "Administration",
			Role:         models.RoleAdmin,
		}
		if err := DB.Create(&adminUser).Error; err != nil {
			return err
		}
		log.Println("Admin account created: Username=admin, Password=password123")
	} else {
		// Admin exists, ensure password is correct
		adminUser.PasswordHash = string(hashedPassword)
		if err := DB.Save(&adminUser).Error; err != nil {
			log.Printf("Warning: Failed to update admin password: %v", err)
		} else {
			log.Println("Admin password reset to: password123")
		}
	}

	log.Println("Seed data check completed")
	return nil
}
