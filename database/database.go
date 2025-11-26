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
		&models.Employee{},
		&models.LeaveType{},
		&models.Leave{},
		&models.LeaveAudit{},
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
			{Name: "Sick", MaxDays: 10},
			{Name: "Casual", MaxDays: 12},
			{Name: "Annual", MaxDays: 20},
			{Name: "Maternity", MaxDays: 90},
			{Name: "Paternity", MaxDays: 14},
		}

		for _, lt := range leaveTypes {
			if err := DB.Create(&lt).Error; err != nil {
				return err
			}
		}
		log.Println("Leave types seeded")
	}

	// Seed Test Employees (only if they don't exist)
	var employeeCount int64
	DB.Model(&models.Employee{}).Count(&employeeCount)
	if employeeCount == 0 {
		// Default password for all test users: "password123"
		defaultPassword := "password123"
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		// Helper for string pointers
		strPtr := func(s string) *string { return &s }

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
			{
				Username:     strPtr("admin"),
				Firstname:    "Admin",
				Lastname:     "User",
				Email:        "admin@example.com",
				PasswordHash: string(hashedPassword),
				Department:   "Administration",
				Role:         models.RoleAdmin,
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
		log.Println("  Admin:    Username=admin, Password=password123")
	}

	log.Println("Seed data check completed")
	return nil
}
