package routes

import (
	"hrms-api/handlers"
	"hrms-api/middleware"
	"hrms-api/models"
	"net/http"
	"path/filepath"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRoutes() *gin.Engine {
	r := gin.Default()

	// CORS configuration
	// Allow requests from frontend on port 8070 (any IP/hostname) and common dev ports
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			// Allow empty origin (same-origin requests, mobile apps, etc.)
			if origin == "" {
				return true
			}

			// Allow if origin ends with :8070 (any IP or hostname)
			// This handles http://192.168.1.100:8070, http://localhost:8070, etc.
			if len(origin) >= 6 {
				suffix := origin[len(origin)-5:]
				if suffix == ":8070" {
					return true
				}
			}

			// Allow common development ports from any host
			allowedDevPorts := []string{":5173", ":3000", ":8080", ":5174", ":5175"}
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
	}))

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Serve static files from static directory (built Vue app)
	staticDir := "./static"
	if _, err := filepath.Abs(staticDir); err == nil {
		// Serve static assets
		r.Static("/assets", filepath.Join(staticDir, "assets"))
		r.StaticFile("/favicon.ico", filepath.Join(staticDir, "favicon.ico"))

		// Serve index.html for all non-API routes (SPA routing)
		r.NoRoute(func(c *gin.Context) {
			// Don't serve index.html for API routes
			if !filepath.HasPrefix(c.Request.URL.Path, "/api") &&
				!filepath.HasPrefix(c.Request.URL.Path, "/auth") &&
				!filepath.HasPrefix(c.Request.URL.Path, "/swagger") &&
				c.Request.URL.Path != "/health" {
				c.File(filepath.Join(staticDir, "index.html"))
			} else {
				c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			}
		})
	}

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
