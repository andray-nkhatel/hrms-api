package handlers

import (
	"fmt"
	"hrms-api/database"
	"hrms-api/models"
	"hrms-api/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ApplyLeaveRequest represents a leave application
type ApplyLeaveRequest struct {
	LeaveTypeID uint      `json:"leave_type_id" binding:"required" example:"1"`
	StartDate   time.Time `json:"start_date" binding:"required" time_format:"2006-01-02" example:"2025-12-01"`
	EndDate     time.Time `json:"end_date" binding:"required" time_format:"2006-01-02" example:"2025-12-05"`
	Reason      string    `json:"reason" example:"Family vacation"`
}

// LeaveBalanceResponse represents leave balance for a leave type
type LeaveBalanceResponse struct {
	LeaveTypeID   uint   `json:"leave_type_id" example:"1"`
	LeaveTypeName string `json:"leave_type_name" example:"Annual"`
	MaxDays       int    `json:"max_days" example:"20"`
	UsedDays      int    `json:"used_days" example:"5"`
	Balance       int    `json:"balance" example:"15"`
}

// ApplyLeave creates a new leave request
// @Summary Apply for leave
// @Description Submit a new leave request
// @Tags Leaves
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body ApplyLeaveRequest true "Leave application data"
// @Success 201 {object} models.Leave
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse "Overlapping leave exists"
// @Router /api/leaves [post]
func ApplyLeave(c *gin.Context) {
	userID, _ := c.Get("user_id")
	employeeID := userID.(uint)

	var req ApplyLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate dates
	if err := utils.ValidateLeaveDates(req.StartDate, req.EndDate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if leave type exists
	var leaveType models.LeaveType
	if err := database.DB.First(&leaveType, req.LeaveTypeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Leave type not found"})
		return
	}

	// Check for overlapping leaves
	hasOverlap, err := utils.CheckOverlappingLeaves(employeeID, req.StartDate, req.EndDate, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check overlapping leaves"})
		return
	}
	if hasOverlap {
		c.JSON(http.StatusConflict, gin.H{"error": utils.ErrOverlappingLeave.Error()})
		return
	}

	// Ensure accruals are up to date for annual leave
	var leaveTypeCheck models.LeaveType
	if err := database.DB.First(&leaveTypeCheck, req.LeaveTypeID).Error; err == nil {
		if leaveTypeCheck.Name == "Annual" || leaveTypeCheck.MaxDays == 24 {
			utils.EnsureAccrualsUpToDate(employeeID, req.LeaveTypeID)
		}
	}

	// Check leave balance
	// For annual leave with future start dates, calculate projected balance at start date
	var balance int
	if (leaveType.Name == "Annual" || leaveType.MaxDays == 24) && req.StartDate.After(time.Now()) {
		// Calculate projected balance at the start date of the leave
		projectedBalance, err := utils.CalculateProjectedAnnualLeaveBalance(employeeID, req.LeaveTypeID, req.StartDate)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate projected leave balance"})
			return
		}
		balance = int(projectedBalance)
	} else {
		// Use current balance for immediate leaves or non-annual leave types
		var err error
		balance, err = utils.CalculateLeaveBalance(employeeID, req.LeaveTypeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate leave balance"})
			return
		}
	}

	leaveDuration := int(req.EndDate.Sub(req.StartDate).Hours()/24) + 1
	if leaveDuration > balance {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           utils.ErrInsufficientBalance.Error(),
			"current_balance": balance,
			"requested_days":  leaveDuration,
			"message":         fmt.Sprintf("Insufficient leave balance. You have %d days available, but requested %d days.", balance, leaveDuration),
		})
		return
	}

	// Create leave request
	leave := models.Leave{
		EmployeeID:  employeeID,
		LeaveTypeID: req.LeaveTypeID,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Reason:      req.Reason,
		Status:      models.StatusPending,
	}

	if err := database.DB.Create(&leave).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create leave request"})
		return
	}

	// Create audit record
	createAuditRecord(leave.ID, models.AuditActionCreate, employeeID, "", string(leave.Status), req.Reason, c.ClientIP())

	// Load associations
	database.DB.Preload("LeaveType").Preload("Employee").First(&leave, leave.ID)

	c.JSON(http.StatusCreated, leave)
}

// GetMyLeaves returns the leave history for the authenticated employee
// @Summary Get my leave history
// @Description Get all leave requests for the authenticated employee
// @Tags Leaves
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Leave
// @Failure 401 {object} ErrorResponse
// @Router /api/leaves [get]
func GetMyLeaves(c *gin.Context) {
	userID, _ := c.Get("user_id")
	employeeID := userID.(uint)

	var leaves []models.Leave
	if err := database.DB.Where("employee_id = ?", employeeID).
		Preload("LeaveType").
		Order("created_at DESC").
		Find(&leaves).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaves"})
		return
	}

	c.JSON(http.StatusOK, leaves)
}

// GetLeaveBalance returns the leave balance for all leave types
// @Summary Get leave balance
// @Description Get remaining leave balance for all leave types
// @Tags Leaves
// @Produce json
// @Security BearerAuth
// @Success 200 {array} LeaveBalanceResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/leaves/balance [get]
func GetLeaveBalance(c *gin.Context) {
	userID, _ := c.Get("user_id")
	employeeID := userID.(uint)

	var leaveTypes []models.LeaveType
	if err := database.DB.Find(&leaveTypes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leave types"})
		return
	}

	// Ensure accruals are up to date for annual leave
	var annualLeaveType models.LeaveType
	if err := database.DB.Where("name = ? OR max_days = ?", "Annual", 24).First(&annualLeaveType).Error; err == nil {
		utils.EnsureAccrualsUpToDate(employeeID, annualLeaveType.ID)
	}

	var balances []LeaveBalanceResponse
	for _, lt := range leaveTypes {
		// Ensure accruals are up to date for annual leave
		if lt.Name == "Annual" || lt.MaxDays == 24 {
			utils.EnsureAccrualsUpToDate(employeeID, lt.ID)
		}

		balance, err := utils.CalculateLeaveBalance(employeeID, lt.ID)
		if err != nil {
			continue
		}

		// Calculate used days
		var usedDays int
		var leaves []models.Leave
		database.DB.Where("employee_id = ? AND leave_type_id = ? AND status = ?",
			employeeID, lt.ID, models.StatusApproved).Find(&leaves)
		for _, leave := range leaves {
			usedDays += leave.GetDuration()
		}

		balances = append(balances, LeaveBalanceResponse{
			LeaveTypeID:   lt.ID,
			LeaveTypeName: lt.Name,
			MaxDays:       lt.MaxDays,
			UsedDays:      usedDays,
			Balance:       balance,
		})
	}

	c.JSON(http.StatusOK, balances)
}

// GetPendingLeaves returns all pending leave requests
// @Summary Get pending leaves
// @Description Get all pending leave requests (Manager/Admin only)
// @Tags Manager
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Leave
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /api/leaves/pending [get]
func GetPendingLeaves(c *gin.Context) {
	var leaves []models.Leave
	if err := database.DB.Where("status = ?", models.StatusPending).
		Preload("Employee").
		Preload("LeaveType").
		Order("created_at ASC").
		Find(&leaves).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch pending leaves"})
		return
	}

	c.JSON(http.StatusOK, leaves)
}

// ApproveLeave approves a leave request
// @Summary Approve leave
// @Description Approve a pending leave request (Manager/Admin only)
// @Tags Manager
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave ID"
// @Success 200 {object} models.Leave
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/leaves/{id}/approve [put]
func ApproveLeave(c *gin.Context) {
	leaveID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid leave ID"})
		return
	}

	userID, _ := c.Get("user_id")
	approverID := userID.(uint)

	var leave models.Leave
	if err := database.DB.Preload("Employee").Preload("LeaveType").First(&leave, uint(leaveID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Leave not found"})
		return
	}

	if leave.Status != models.StatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Leave is not in pending status"})
		return
	}

	// Ensure accruals are up to date for annual leave
	if leave.LeaveType.Name == "Annual" || leave.LeaveType.MaxDays == 24 {
		utils.EnsureAccrualsUpToDate(leave.EmployeeID, leave.LeaveTypeID)
	}

	// Check balance again before approving
	balance, err := utils.CalculateLeaveBalance(leave.EmployeeID, leave.LeaveTypeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate leave balance"})
		return
	}

	leaveDuration := leave.GetDuration()
	if leaveDuration > balance {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient leave balance"})
		return
	}

	oldStatus := string(leave.Status)
	now := time.Now()
	leave.Status = models.StatusApproved
	leave.ApprovedBy = &approverID
	leave.ApprovedAt = &now

	if err := database.DB.Save(&leave).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve leave"})
		return
	}

	// Create audit record
	createAuditRecord(leave.ID, models.AuditActionApprove, approverID, oldStatus, string(leave.Status), "Approved", c.ClientIP())

	c.JSON(http.StatusOK, leave)
}

// RejectLeaveRequest represents a rejection request
type RejectLeaveRequest struct {
	Reason string `json:"reason" binding:"required" example:"Insufficient staffing during requested period"`
}

// RejectLeave rejects a leave request
// @Summary Reject leave
// @Description Reject a pending leave request with reason (Manager/Admin only)
// @Tags Manager
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave ID"
// @Param request body RejectLeaveRequest true "Rejection reason"
// @Success 200 {object} models.Leave
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/leaves/{id}/reject [put]
func RejectLeave(c *gin.Context) {
	leaveID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid leave ID"})
		return
	}

	var req RejectLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rejection reason is required"})
		return
	}

	userID, _ := c.Get("user_id")
	approverID := userID.(uint)

	var leave models.Leave
	if err := database.DB.Preload("Employee").Preload("LeaveType").First(&leave, uint(leaveID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Leave not found"})
		return
	}

	if leave.Status != models.StatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Leave is not in pending status"})
		return
	}

	oldStatus := string(leave.Status)
	now := time.Now()
	leave.Status = models.StatusRejected
	leave.RejectionReason = req.Reason
	leave.ApprovedBy = &approverID
	leave.ApprovedAt = &now

	if err := database.DB.Save(&leave).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject leave"})
		return
	}

	// Create audit record
	createAuditRecord(leave.ID, models.AuditActionReject, approverID, oldStatus, string(leave.Status), req.Reason, c.ClientIP())

	c.JSON(http.StatusOK, leave)
}

// CancelLeave cancels a leave request
// @Summary Cancel leave
// @Description Cancel own pending or approved leave request
// @Tags Leaves
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave ID"
// @Success 200 {object} models.Leave
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/leaves/{id}/cancel [put]
func CancelLeave(c *gin.Context) {
	leaveID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid leave ID"})
		return
	}

	userID, _ := c.Get("user_id")
	employeeID := userID.(uint)

	var leave models.Leave
	if err := database.DB.Preload("Employee").Preload("LeaveType").First(&leave, uint(leaveID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Leave not found"})
		return
	}

	// Ensure employee can only cancel their own leave
	if leave.EmployeeID != employeeID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only cancel your own leave requests"})
		return
	}

	// Can only cancel pending or approved (future) leaves
	if leave.Status != models.StatusPending && leave.Status != models.StatusApproved {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only pending or approved leaves can be cancelled"})
		return
	}

	// For approved leaves, can only cancel if start date is in the future
	if leave.Status == models.StatusApproved && !leave.StartDate.After(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot cancel leave that has already started"})
		return
	}

	oldStatus := string(leave.Status)
	leave.Status = models.StatusCancelled

	if err := database.DB.Save(&leave).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel leave"})
		return
	}

	// Create audit record
	createAuditRecord(leave.ID, models.AuditActionCancel, employeeID, oldStatus, string(leave.Status), "Cancelled by employee", c.ClientIP())

	c.JSON(http.StatusOK, leave)
}

// GetLeaveAudit returns audit trail for a leave request
// @Summary Get leave audit trail
// @Description Get audit history for a leave request (Manager/Admin only)
// @Tags Manager
// @Produce json
// @Security BearerAuth
// @Param id path int true "Leave ID"
// @Success 200 {array} models.LeaveAudit
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/leaves/{id}/audit [get]
func GetLeaveAudit(c *gin.Context) {
	leaveID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid leave ID"})
		return
	}

	var audits []models.LeaveAudit
	if err := database.DB.Where("leave_id = ?", uint(leaveID)).
		Preload("Performer").
		Order("created_at ASC").
		Find(&audits).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch audit records"})
		return
	}

	c.JSON(http.StatusOK, audits)
}

// Helper function to create audit records
func createAuditRecord(leaveID uint, action models.AuditAction, performedBy uint, oldStatus, newStatus, comment, ipAddress string) {
	audit := models.LeaveAudit{
		LeaveID:     leaveID,
		Action:      action,
		PerformedBy: performedBy,
		OldStatus:   oldStatus,
		NewStatus:   newStatus,
		Comment:     comment,
		IPAddress:   ipAddress,
	}
	database.DB.Create(&audit)
}
