package handlers

import (
	"encoding/csv"
	"fmt"
	"hrms-api/database"
	"hrms-api/models"
	"hrms-api/utils"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// BulkCreateLeavesRequest represents a request to create multiple leaves from CSV
type BulkCreateLeavesRequest struct {
	SkipInvalidRows bool `json:"skip_invalid_rows" example:"true"` // Skip rows with errors instead of failing entire import
}

// BulkCreateLeavesResponse represents the response from bulk leave creation
type BulkCreateLeavesResponse struct {
	Total   int                     `json:"total"`
	Success int                     `json:"success"`
	Failed  int                     `json:"failed"`
	Results []BulkLeaveCreateResult `json:"results"`
}

// BulkLeaveCreateResult represents the result of creating a single leave
type BulkLeaveCreateResult struct {
	RowNumber     int    `json:"row_number"`
	EmployeeName  string `json:"employee_name"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	LeaveTypeName string `json:"leave_type_name"`
	Success       bool   `json:"success"`
	LeaveID       *uint  `json:"leave_id,omitempty"`
	Error         string `json:"error,omitempty"`
}

// BulkCreateLeaves creates multiple leave records from CSV file
// @Summary Bulk create leaves from CSV
// @Description Import multiple leave records from CSV file. CSV format: Employee Name, Leave Type, Start Date (YYYY-MM-DD), End Date (YYYY-MM-DD), Reason (optional)
// @Tags HR - Leave Management
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "CSV file with leave data"
// @Param skip_invalid_rows formData bool false "Skip invalid rows instead of failing entire import"
// @Success 200 {object} BulkCreateLeavesResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/hr/leaves/bulk-import [post]
func BulkCreateLeaves(c *gin.Context) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	skipInvalid := c.PostForm("skip_invalid_rows") == "true"

	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	var results []BulkLeaveCreateResult
	total := 0
	success := 0
	failed := 0

	// Get current user (admin)
	userID, _ := c.Get("user_id")
	adminID := userID.(uint)

	// Get Annual leave type (default for bulk import)
	var annualLeaveType models.LeaveType
	if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Annual leave type not found"})
		return
	}

	// Read header row
	header, err := reader.Read()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read CSV file: " + err.Error()})
		return
	}

	// Validate header (expect: Employee Name, Start Date, End Date, Reason - Leave Type is optional, defaults to Annual)
	headerMap := make(map[string]int)
	for i, h := range header {
		headerMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	// Check if required headers exist (Leave Type is optional, will default to Annual)
	requiredHeaders := []string{"employee name", "start date", "end date"}
	for _, expected := range requiredHeaders {
		if _, exists := headerMap[expected]; !exists {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Missing required column: %s. Expected columns: Employee Name, Start Date, End Date, Reason (optional). Leave Type is optional and defaults to Annual.", expected),
			})
			return
		}
	}

	// Read data rows
	rowNum := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			if skipInvalid {
				failed++
				results = append(results, BulkLeaveCreateResult{
					RowNumber: rowNum,
					Success:   false,
					Error:     "Failed to parse row: " + err.Error(),
				})
				rowNum++
				continue
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to read row %d: %v", rowNum, err)})
			return
		}

		rowNum++
		total++

		// Skip empty rows
		if len(record) == 0 || (len(record) == 1 && strings.TrimSpace(record[0]) == "") {
			continue
		}

		// Extract data from CSV row
		employeeName := strings.TrimSpace(record[headerMap["employee name"]])
		startDateStr := strings.TrimSpace(record[headerMap["start date"]])
		endDateStr := strings.TrimSpace(record[headerMap["end date"]])
		reason := ""
		if reasonIdx, exists := headerMap["reason"]; exists && reasonIdx < len(record) {
			reason = strings.TrimSpace(record[reasonIdx])
		}

		// Leave type is optional - default to Annual
		leaveTypeName := "Annual"
		if leaveTypeIdx, exists := headerMap["leave type"]; exists && leaveTypeIdx < len(record) {
			parsedType := strings.TrimSpace(record[leaveTypeIdx])
			if parsedType != "" {
				leaveTypeName = parsedType
			}
		}

		// Validate required fields
		if employeeName == "" || startDateStr == "" || endDateStr == "" {
			if skipInvalid {
				failed++
				results = append(results, BulkLeaveCreateResult{
					RowNumber:    rowNum - 1,
					EmployeeName: employeeName,
					Success:      false,
					Error:        "Missing required fields (Employee Name, Start Date, End Date)",
				})
				continue
			}
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Row %d: Missing required fields", rowNum-1),
			})
			return
		}

		// Find employee by name
		var employee models.Employee
		nameParts := strings.Fields(employeeName)
		if len(nameParts) >= 2 {
			// Try firstname + lastname
			firstname := nameParts[0]
			lastname := strings.Join(nameParts[1:], " ")
			if err := database.DB.Where("LOWER(firstname) = LOWER(?) AND LOWER(lastname) = LOWER(?)", firstname, lastname).First(&employee).Error; err != nil {
				// Try alternative: first word as firstname, rest as lastname
				if skipInvalid {
					failed++
					results = append(results, BulkLeaveCreateResult{
						RowNumber:    rowNum - 1,
						EmployeeName: employeeName,
						Success:      false,
						Error:        "Employee not found: " + employeeName,
					})
					continue
				}
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("Row %d: Employee not found: %s", rowNum-1, employeeName),
				})
				return
			}
		} else {
			if skipInvalid {
				failed++
				results = append(results, BulkLeaveCreateResult{
					RowNumber:    rowNum - 1,
					EmployeeName: employeeName,
					Success:      false,
					Error:        "Invalid employee name format (need firstname and lastname)",
				})
				continue
			}
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Row %d: Invalid employee name format", rowNum-1),
			})
			return
		}

		// Use Annual leave type (default for bulk import)
		leaveType := annualLeaveType

		// Parse dates
		startDate, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			if skipInvalid {
				failed++
				results = append(results, BulkLeaveCreateResult{
					RowNumber:    rowNum - 1,
					EmployeeName: employeeName,
					Success:      false,
					Error:        "Invalid start date format: " + startDateStr + " (expected YYYY-MM-DD)",
				})
				continue
			}
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Row %d: Invalid start date format: %s", rowNum-1, startDateStr),
			})
			return
		}

		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			if skipInvalid {
				failed++
				results = append(results, BulkLeaveCreateResult{
					RowNumber:    rowNum - 1,
					EmployeeName: employeeName,
					Success:      false,
					Error:        "Invalid end date format: " + endDateStr + " (expected YYYY-MM-DD)",
				})
				continue
			}
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Row %d: Invalid end date format: %s", rowNum-1, endDateStr),
			})
			return
		}

		// Validate dates
		if startDate.After(endDate) {
			if skipInvalid {
				failed++
				results = append(results, BulkLeaveCreateResult{
					RowNumber:    rowNum - 1,
					EmployeeName: employeeName,
					Success:      false,
					Error:        "Start date must be before or equal to end date",
				})
				continue
			}
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Row %d: Start date must be before or equal to end date", rowNum-1),
			})
			return
		}

		// Check for overlapping leaves
		hasOverlap, err := utils.CheckOverlappingLeaves(employee.ID, startDate, endDate, nil)
		if err != nil {
			if skipInvalid {
				failed++
				results = append(results, BulkLeaveCreateResult{
					RowNumber:    rowNum - 1,
					EmployeeName: employeeName,
					Success:      false,
					Error:        "Failed to check overlapping leaves: " + err.Error(),
				})
				continue
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Row %d: Failed to check overlapping leaves", rowNum-1),
			})
			return
		}
		if hasOverlap {
			if skipInvalid {
				failed++
				results = append(results, BulkLeaveCreateResult{
					RowNumber:    rowNum - 1,
					EmployeeName: employeeName,
					Success:      false,
					Error:        "Overlapping leave exists",
				})
				continue
			}
			c.JSON(http.StatusConflict, gin.H{
				"error": fmt.Sprintf("Row %d: Overlapping leave exists for %s", rowNum-1, employeeName),
			})
			return
		}

		var leaveDuration float64
		if leaveType.UsesBalance {
			utils.EnsureAccrualsUpToDate(employee.ID, leaveType.ID)

			balance, err := utils.GetCurrentLeaveBalance(employee.ID, leaveType.ID)
			if err != nil {
				if skipInvalid {
					failed++
					results = append(results, BulkLeaveCreateResult{
						RowNumber:    rowNum - 1,
						EmployeeName: employeeName,
						Success:      false,
						Error:        "Failed to calculate leave balance: " + err.Error(),
					})
					continue
				}
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("Row %d: Failed to calculate leave balance", rowNum-1),
				})
				return
			}

			leaveDuration = float64(int(endDate.Sub(startDate).Hours()/24) + 1)
			if leaveDuration > balance {
				if skipInvalid {
					failed++
					results = append(results, BulkLeaveCreateResult{
						RowNumber:    rowNum - 1,
						EmployeeName: employeeName,
						Success:      false,
						Error:        fmt.Sprintf("Insufficient balance: %.2f available, %.2f requested", balance, leaveDuration),
					})
					continue
				}
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("Row %d: Insufficient leave balance for %s", rowNum-1, employeeName),
				})
				return
			}
		} else {
			leaveDuration = float64(int(endDate.Sub(startDate).Hours()/24) + 1)
		}

		// Create leave record (default to Approved for admin-created leaves)
		now := time.Now()
		leave := models.Leave{
			EmployeeID:  employee.ID,
			LeaveTypeID: leaveType.ID,
			StartDate:   startDate,
			EndDate:     endDate,
			Reason:      reason,
			Status:      models.StatusApproved,
			ApprovedBy:  &adminID,
			ApprovedAt:  &now,
		}

		if err := database.DB.Create(&leave).Error; err != nil {
			if skipInvalid {
				failed++
				results = append(results, BulkLeaveCreateResult{
					RowNumber:    rowNum - 1,
					EmployeeName: employeeName,
					Success:      false,
					Error:        "Failed to create leave: " + err.Error(),
				})
				continue
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Row %d: Failed to create leave", rowNum-1),
			})
			return
		}

		if leaveType.UsesBalance {
			if leaveType.AllowCarryOver {
				utils.UpdateCarryOverUsage(employee.ID, leaveType.ID, leaveDuration)
			}
			if leave.Status == models.StatusApproved {
				if err := utils.EnsureAccrualsUpToDate(employee.ID, leaveType.ID); err != nil {
					// Log error but don't fail the creation
				}
			}
		}

		// Create audit record
		createAuditRecord(leave.ID, models.AuditActionCreate, adminID, "", string(leave.Status), fmt.Sprintf("Bulk imported: %s", reason), c.ClientIP())

		success++
		results = append(results, BulkLeaveCreateResult{
			RowNumber:     rowNum - 1,
			EmployeeName:  employeeName,
			StartDate:     startDateStr,
			EndDate:       endDateStr,
			LeaveTypeName: leaveTypeName,
			Success:       true,
			LeaveID:       &leave.ID,
		})
	}

	c.JSON(http.StatusOK, BulkCreateLeavesResponse{
		Total:   total,
		Success: success,
		Failed:  failed,
		Results: results,
	})
}

// BulkCreateLeavesFromTemplate creates leaves for multiple employees using a template
// @Summary Bulk create leaves from template
// @Description Create the same leave for multiple employees (e.g., public holiday for all)
// @Tags HR - Leave Management
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body BulkCreateLeavesTemplateRequest true "Template data"
// @Success 200 {object} BulkCreateLeavesResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/hr/leaves/bulk-template [post]
func BulkCreateLeavesFromTemplate(c *gin.Context) {
	var req BulkCreateLeavesTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current user (admin)
	userID, _ := c.Get("user_id")
	adminID := userID.(uint)

	// Verify leave type exists
	var leaveType models.LeaveType
	if err := database.DB.First(&leaveType, req.LeaveTypeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Leave type not found"})
		return
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format. Use YYYY-MM-DD"})
		return
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format. Use YYYY-MM-DD"})
		return
	}

	if startDate.After(endDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Start date must be before or equal to end date"})
		return
	}

	var results []BulkLeaveCreateResult
	success := 0
	failed := 0

	// Process each employee
	for i, employeeID := range req.EmployeeIDs {
		var employee models.Employee
		if err := database.DB.First(&employee, employeeID).Error; err != nil {
			failed++
			results = append(results, BulkLeaveCreateResult{
				RowNumber:    i + 1,
				EmployeeName: fmt.Sprintf("Employee ID %d", employeeID),
				Success:      false,
				Error:        "Employee not found",
			})
			continue
		}

		// Check for overlapping leaves
		hasOverlap, err := utils.CheckOverlappingLeaves(employeeID, startDate, endDate, nil)
		if err != nil {
			failed++
			results = append(results, BulkLeaveCreateResult{
				RowNumber:    i + 1,
				EmployeeName: employee.Firstname + " " + employee.Lastname,
				Success:      false,
				Error:        "Failed to check overlapping leaves",
			})
			continue
		}
		if hasOverlap {
			failed++
			results = append(results, BulkLeaveCreateResult{
				RowNumber:    i + 1,
				EmployeeName: employee.Firstname + " " + employee.Lastname,
				Success:      false,
				Error:        "Overlapping leave exists",
			})
			continue
		}

		if leaveType.UsesBalance {
			utils.EnsureAccrualsUpToDate(employeeID, leaveType.ID)

			balance, err := utils.GetCurrentLeaveBalance(employeeID, leaveType.ID)
			if err != nil {
				failed++
				results = append(results, BulkLeaveCreateResult{
					RowNumber:    i + 1,
					EmployeeName: employee.Firstname + " " + employee.Lastname,
					Success:      false,
					Error:        "Failed to calculate balance",
				})
				continue
			}

			leaveDuration := float64(int(endDate.Sub(startDate).Hours()/24) + 1)
			if leaveDuration > balance {
				failed++
				results = append(results, BulkLeaveCreateResult{
					RowNumber:    i + 1,
					EmployeeName: employee.Firstname + " " + employee.Lastname,
					Success:      false,
					Error:        fmt.Sprintf("Insufficient balance: %.2f available", balance),
				})
				continue
			}
		}

		leaveDuration := float64(int(endDate.Sub(startDate).Hours()/24) + 1)

		// Create leave
		now := time.Now()
		leave := models.Leave{
			EmployeeID:  employeeID,
			LeaveTypeID: leaveType.ID,
			StartDate:   startDate,
			EndDate:     endDate,
			Reason:      req.Reason,
			Status:      models.StatusApproved,
			ApprovedBy:  &adminID,
			ApprovedAt:  &now,
		}

		if err := database.DB.Create(&leave).Error; err != nil {
			failed++
			results = append(results, BulkLeaveCreateResult{
				RowNumber:    i + 1,
				EmployeeName: employee.Firstname + " " + employee.Lastname,
				Success:      false,
				Error:        "Failed to create leave",
			})
			continue
		}

		if leaveType.UsesBalance && leaveType.AllowCarryOver {
			utils.UpdateCarryOverUsage(employeeID, leaveType.ID, leaveDuration)
		}

		// Create audit record
		createAuditRecord(leave.ID, models.AuditActionCreate, adminID, "", string(leave.Status), fmt.Sprintf("Bulk template: %s", req.Reason), c.ClientIP())

		success++
		results = append(results, BulkLeaveCreateResult{
			RowNumber:     i + 1,
			EmployeeName:  employee.Firstname + " " + employee.Lastname,
			StartDate:     req.StartDate,
			EndDate:       req.EndDate,
			LeaveTypeName: leaveType.Name,
			Success:       true,
			LeaveID:       &leave.ID,
		})
	}

	c.JSON(http.StatusOK, BulkCreateLeavesResponse{
		Total:   len(req.EmployeeIDs),
		Success: success,
		Failed:  failed,
		Results: results,
	})
}

// BulkCreateLeavesTemplateRequest represents a request to create leaves from template
type BulkCreateLeavesTemplateRequest struct {
	EmployeeIDs []uint `json:"employee_ids" binding:"required"`
	LeaveTypeID uint   `json:"leave_type_id" binding:"required"`
	StartDate   string `json:"start_date" binding:"required"`
	EndDate     string `json:"end_date" binding:"required"`
	Reason      string `json:"reason,omitempty"`
}
