package routes

import (
	"hrms-api/handlers"
	"hrms-api/middleware"
	"hrms-api/models"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes() *gin.Engine {
	r := gin.Default()

	// CORS configuration
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Public routes
	auth := r.Group("/auth")
	{
		auth.POST("/login", handlers.Login)            // Employee/Manager login with NRC
		auth.POST("/admin/login", handlers.AdminLogin) // Admin login with username
		auth.POST("/register", handlers.Register)
	}

	// Protected routes
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		// Employee routes (all authenticated users)
		leaves := api.Group("/leaves")
		{
			leaves.POST("", handlers.ApplyLeave)
			leaves.GET("", handlers.GetMyLeaves)
			leaves.GET("/balance", handlers.GetLeaveBalance)
			leaves.PUT("/:id/cancel", handlers.CancelLeave) // Employees can cancel their own leaves
		}

		// Leave types - GET is available to all, other operations require admin
		api.GET("/leave-types", handlers.GetLeaveTypes)

		// Manager routes
		manager := api.Group("")
		manager.Use(middleware.RequireRole(models.RoleManager, models.RoleAdmin))
		{
			manager.GET("/leaves/pending", handlers.GetPendingLeaves)
			manager.PUT("/leaves/:id/approve", handlers.ApproveLeave)
			manager.PUT("/leaves/:id/reject", handlers.RejectLeave)
			manager.GET("/leaves/:id/audit", handlers.GetLeaveAudit) // View audit trail
		}

		// Admin routes
		admin := api.Group("")
		admin.Use(middleware.RequireRole(models.RoleAdmin))
		{
			// Leave types management (create, update, delete)
			admin.POST("/leave-types", handlers.CreateLeaveType)
			admin.PUT("/leave-types/:id", handlers.UpdateLeaveType)
			admin.DELETE("/leave-types/:id", handlers.DeleteLeaveType)

			// Employee management
			admin.GET("/employees", handlers.GetEmployees)
			admin.POST("/employees", handlers.CreateEmployee)                   // For employees/managers (NRC)
			admin.POST("/admins", handlers.CreateAdmin)                         // For admins (username)
			admin.GET("/employees/template", handlers.DownloadEmployeeTemplate) // CSV template
			admin.POST("/employees/bulk", handlers.BulkUploadEmployees)         // Bulk upload
			admin.GET("/employees/:id", handlers.GetEmployee)
			admin.PUT("/employees/:id", handlers.UpdateEmployee)
			admin.DELETE("/employees/:id", handlers.DeleteEmployee)
		}
	}

	return r
}
