package main

import (
	"hrms-api/config"
	"hrms-api/database"
	_ "hrms-api/docs"
	"hrms-api/routes"
	"log"

	"github.com/gin-gonic/gin"
)

// @title HRMS Leave Management API
// @version 1.0
// @description REST API for Leave Management System - part of HRMS
// @description Allows employees to apply for leaves, track balances, and managers to approve/reject requests.

// @contact.name API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8070
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter "Bearer {token}" (without quotes)

func main() {
	// Load configuration
	if err := config.LoadConfig(); err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Set Gin mode
	gin.SetMode(config.AppConfig.GinMode)

	// Connect to database
	if err := database.Connect(); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Run migrations
	if err := database.Migrate(); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed initial data
	if err := database.SeedData(); err != nil {
		log.Fatal("Failed to seed database:", err)
	}

	// Setup routes
	r := routes.SetupRoutes()

	// Start server - bind to all interfaces (0.0.0.0) to allow network access
	address := "0.0.0.0:" + config.AppConfig.Port
	log.Printf("Server starting on %s", address)
	if err := r.Run(address); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
