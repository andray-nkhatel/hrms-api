package routes

import (
	"hrms-api/handlers"
	"hrms-api/middleware"
	"hrms-api/models"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes() *gin.Engine {
	r := gin.Default()

	// CORS configuration
	// Check if we're in development mode for more permissive CORS
	ginMode := os.Getenv("GIN_MODE")
	isDevelopment := ginMode == "" || ginMode == "debug" || ginMode == "test"
	// Allow explicit override via environment variable
	corsAllowAll := os.Getenv("CORS_ALLOW_ALL") == "true" || os.Getenv("CORS_ALLOW_ALL") == "1"

	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// Allow empty origin (same-origin requests, mobile apps, etc.)
			if origin == "" {
				return true
			}

			// Always allow requests from Swagger UI (same origin)
			// Swagger is served from the same server, so same-origin requests have empty origin
			// But we also allow explicit same-origin requests

			// In development mode or if CORS_ALLOW_ALL is set, allow all http/https origins
			if isDevelopment || corsAllowAll {
				// Check if it's a valid http/https URL (must start with http:// or https://)
				if strings.HasPrefix(origin, "http://") || strings.HasPrefix(origin, "https://") {
					return true
				}
			}

			// Production: More restrictive origin checking
			// Allow if origin ends with :8070 (any IP or hostname)
			// This handles http://192.168.1.100:8070, http://localhost:8070, etc.
			if len(origin) >= 6 {
				suffix := origin[len(origin)-5:]
				if suffix == ":8070" {
					return true
				}
			}

			// Allow common development ports from any host
			allowedDevPorts := []string{":5173", ":3000", ":8080", ":5174", ":5175", ":4200", ":5176"}
			for _, port := range allowedDevPorts {
				if len(origin) >= len(port) {
					suffix := origin[len(origin)-len(port):]
					if suffix == port {
						return true
					}
				}
			}

			// Allow localhost variants (with or without port)
			if origin == "http://localhost" || origin == "https://localhost" ||
				origin == "http://127.0.0.1" || origin == "https://127.0.0.1" {
				return true
			}

			// Allow any localhost with port
			if len(origin) > 16 && (origin[:16] == "http://localhost:" || origin[:17] == "https://localhost:") {
				return true
			}

			// Allow any 127.0.0.1 with port
			if len(origin) > 17 && (origin[:17] == "http://127.0.0.1:" || origin[:18] == "https://127.0.0.1:") {
				return true
			}

			return false
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * 3600, // 12 hours
	}))

	// Swagger documentation
	// Configure Swagger with CORS support
	swaggerHandler := ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.DeepLinking(true), ginSwagger.DefaultModelsExpandDepth(-1))
	r.GET("/swagger/*any", swaggerHandler)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Serve static files from static directory (built Vue app)
	staticDir := "./static"
	if _, err := filepath.Abs(staticDir); err == nil {
		// Check if static directory exists
		if _, err := os.Stat(staticDir); err == nil {
			// Serve static assets
			r.Static("/assets", filepath.Join(staticDir, "assets"))
			r.StaticFile("/favicon.ico", filepath.Join(staticDir, "favicon.ico"))

			// Serve index.html for root path
			r.GET("/", func(c *gin.Context) {
				indexPath := filepath.Join(staticDir, "index.html")
				if _, err := os.Stat(indexPath); err == nil {
					c.File(indexPath)
				} else {
					c.JSON(http.StatusNotFound, gin.H{"error": "Frontend not built. Please build the client first."})
				}
			})

			// Serve index.html for all non-API routes (SPA routing)
			r.NoRoute(func(c *gin.Context) {
				path := c.Request.URL.Path
				// Don't serve index.html for API routes
				if !strings.HasPrefix(path, "/api") &&
					!strings.HasPrefix(path, "/auth") &&
					!strings.HasPrefix(path, "/swagger") &&
					path != "/health" &&
					path != "/" {
					indexPath := filepath.Join(staticDir, "index.html")
					if _, err := os.Stat(indexPath); err == nil {
						c.File(indexPath)
					} else {
						c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
					}
				} else {
					c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
				}
			})
		}
	}

	// Public routes
	auth := r.Group("/auth")
	{
		// GET handler for /auth/login to prevent 404 when browser navigates to the route
		auth.GET("/login", func(c *gin.Context) {
			c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Use POST method to login"})
		})
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

		// HR Leave Management routes (Manager/Admin only)
		hr := api.Group("/hr")
		hr.Use(middleware.RequireRole(models.RoleManager, models.RoleAdmin))
		{
			// View endpoints
			hr.GET("/employees/annual-leave-balances", handlers.GetAllEmployeesLeaveBalances)
			hr.GET("/employees/annual-leave-balances/export", handlers.ExportAnnualLeaveBalances)
			hr.GET("/employees/:id/annual-leave-balance", handlers.GetAnnualLeaveBalance)
			hr.GET("/leaves/calendar", handlers.GetLeaveCalendar)
			hr.GET("/leaves/department-report", handlers.GetDepartmentLeaveReport)
			hr.GET("/leaves/upcoming", handlers.GetUpcomingLeaves)

			// Management endpoints
			hr.POST("/employees/:id/annual-leave-balance/adjust", handlers.AdjustLeaveBalance)
			hr.POST("/employees/:id/annual-leave-balance/accrual", handlers.AddManualAccrual)
			hr.POST("/leaves/process-accruals", handlers.ProcessMonthlyAccruals)
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

		// Core HR routes - Identity Information
		api.GET("/employees/:id/identity", handlers.GetIdentityInformation)
		api.POST("/employees/:id/identity", handlers.CreateOrUpdateIdentityInformation)

		// Core HR routes - Employment Details
		api.GET("/employees/:id/employment", handlers.GetEmploymentDetails)
		api.POST("/employees/:id/employment", handlers.CreateOrUpdateEmploymentDetails)
		api.GET("/employees/:id/employment/history", handlers.GetEmploymentHistory)

		// Core HR routes - Positions
		api.GET("/positions", handlers.GetPositions)
		api.GET("/positions/:id", handlers.GetPosition)
		managerAdmin := api.Group("")
		managerAdmin.Use(middleware.RequireRole(models.RoleManager, models.RoleAdmin))
		{
			managerAdmin.POST("/positions", handlers.CreatePosition)
			managerAdmin.PUT("/positions/:id", handlers.UpdatePosition)
			managerAdmin.POST("/employees/:id/positions", handlers.AssignPosition)
		}

		// Core HR routes - Documents
		api.GET("/employees/:id/documents", handlers.GetDocuments)
		api.POST("/employees/:id/documents", handlers.CreateDocument)
		api.GET("/employees/:id/documents/:doc_id/download", handlers.DownloadDocument)
		api.DELETE("/employees/:id/documents/:doc_id", handlers.DeleteDocument)

		// Core HR routes - Work Lifecycle
		api.GET("/employees/:id/lifecycle", handlers.GetLifecycleEvents)
		managerAdmin.POST("/employees/:id/lifecycle", handlers.CreateLifecycleEvent)

		// Core HR routes - Onboarding
		api.GET("/employees/:id/onboarding", handlers.GetOnboardingProcess)
		managerAdmin.POST("/employees/:id/onboarding", handlers.CreateOnboardingProcess)

		// Core HR routes - Offboarding
		api.GET("/employees/:id/offboarding", handlers.GetOffboardingProcess)
		managerAdmin.POST("/employees/:id/offboarding", handlers.CreateOffboardingProcess)

		// Core HR routes - Compliance
		api.GET("/compliance/requirements", handlers.GetComplianceRequirements)
		api.GET("/employees/:id/compliance", handlers.GetComplianceRecords)
		managerAdmin.POST("/compliance/requirements", handlers.CreateComplianceRequirement)
		managerAdmin.POST("/employees/:id/compliance", handlers.CreateComplianceRecord)

		// Core HR routes - Audit Logs
		api.GET("/audit-logs", handlers.GetAuditLogs)
		api.GET("/employees/:id/audit-logs", handlers.GetEmployeeAuditLogs)
	}

	return r
}
