package utils

import (
	"bytes"
	"fmt"
	"hrms-api/database"
	"hrms-api/models"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

// AnnualLeaveBalanceExport represents data for export
type AnnualLeaveBalanceExport struct {
	EmployeeID     uint
	EmployeeName   string
	Department     string
	TotalAccrued   float64
	TotalUsed      float64
	CurrentBalance float64
	PendingLeaves  int
	UpcomingLeaves int
}

// EmployeeBalanceData represents employee balance data for export
type EmployeeBalanceData struct {
	EmployeeID     uint
	EmployeeName   string
	Department     string
	TotalAccrued   float64
	TotalUsed      float64
	CurrentBalance float64
	PendingLeaves  int
	UpcomingLeaves int
}

// ExportAnnualLeaveBalancesToExcel exports annual leave balances to Excel format
func ExportAnnualLeaveBalancesToExcel(balances []AnnualLeaveBalanceExport) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Annual Leave Balances"
	f.NewSheet(sheetName)
	f.DeleteSheet("Sheet1")

	// Set column headers
	headers := []string{"Employee ID", "Employee Name", "Department", "Total Accrued", "Total Used", "Current Balance"}
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 12,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#4472C4"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// Set column widths
	f.SetColWidth(sheetName, "A", "A", 12)
	f.SetColWidth(sheetName, "B", "B", 25)
	f.SetColWidth(sheetName, "C", "C", 20)
	f.SetColWidth(sheetName, "D", "F", 15)

	// Write data
	for i, balance := range balances {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), balance.EmployeeID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), balance.EmployeeName)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), balance.Department)
		f.SetCellFloat(sheetName, fmt.Sprintf("D%d", row), balance.TotalAccrued, 2, 64)
		f.SetCellFloat(sheetName, fmt.Sprintf("E%d", row), balance.TotalUsed, 2, 64)
		f.SetCellFloat(sheetName, fmt.Sprintf("F%d", row), balance.CurrentBalance, 2, 64)
	}

	// Add summary row
	summaryRow := len(balances) + 3
	summaryStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})
	f.SetCellValue(sheetName, fmt.Sprintf("B%d", summaryRow), "TOTAL")
	f.SetCellFormula(sheetName, fmt.Sprintf("D%d", summaryRow), fmt.Sprintf("SUM(D2:D%d)", len(balances)+1))
	f.SetCellFormula(sheetName, fmt.Sprintf("E%d", summaryRow), fmt.Sprintf("SUM(E2:E%d)", len(balances)+1))
	f.SetCellFormula(sheetName, fmt.Sprintf("F%d", summaryRow), fmt.Sprintf("SUM(F2:F%d)", len(balances)+1))
	f.SetCellStyle(sheetName, fmt.Sprintf("B%d", summaryRow), fmt.Sprintf("F%d", summaryRow), summaryStyle)

	// Add timestamp
	timestampRow := summaryRow + 2
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", timestampRow), fmt.Sprintf("Generated: %s", time.Now().Format("2006-01-02 15:04:05")))

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ExportAnnualLeaveBalancesToPDF exports annual leave balances to PDF format
func ExportAnnualLeaveBalancesToPDF(balances []AnnualLeaveBalanceExport) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Annual Leave Balance Report")
	pdf.Ln(12)

	// Set font for table
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(200, 200, 200)

	// Table headers
	headers := []string{"ID", "Employee Name", "Department", "Accrued", "Used", "Balance"}
	colWidths := []float64{15, 50, 40, 25, 25, 25}

	// Draw header row
	for i, header := range headers {
		pdf.CellFormat(colWidths[i], 8, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(8)

	// Draw data rows
	pdf.SetFont("Arial", "", 9)
	pdf.SetFillColor(255, 255, 255)
	for i, balance := range balances {
		if i%20 == 0 && i > 0 {
			pdf.AddPage()
			// Redraw headers on new page
			pdf.SetFont("Arial", "B", 10)
			pdf.SetFillColor(200, 200, 200)
			for j, header := range headers {
				pdf.CellFormat(colWidths[j], 8, header, "1", 0, "C", true, 0, "")
			}
			pdf.Ln(8)
			pdf.SetFont("Arial", "", 9)
			pdf.SetFillColor(255, 255, 255)
		}

		pdf.CellFormat(colWidths[0], 7, fmt.Sprintf("%d", balance.EmployeeID), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[1], 7, balance.EmployeeName, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[2], 7, balance.Department, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[3], 7, fmt.Sprintf("%.1f", balance.TotalAccrued), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colWidths[4], 7, fmt.Sprintf("%.1f", balance.TotalUsed), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colWidths[5], 7, fmt.Sprintf("%.1f", balance.CurrentBalance), "1", 0, "R", false, 0, "")
		pdf.Ln(7)
	}

	// Add summary
	pdf.Ln(5)
	pdf.SetFont("Arial", "B", 10)
	var totalAccrued, totalUsed, totalBalance, calculatedBalance float64
	for _, balance := range balances {
		totalAccrued += balance.TotalAccrued
		totalUsed += balance.TotalUsed
		totalBalance += balance.CurrentBalance
	}
	// Calculate expected balance from accruals (without carry-over)
	calculatedBalance = totalAccrued - totalUsed
	pdf.Cell(40, 8, fmt.Sprintf("Total Employees: %d", len(balances)))
	pdf.Ln(5)
	pdf.Cell(40, 8, fmt.Sprintf("Total Accrued: %.1f days", totalAccrued))
	pdf.Ln(5)
	pdf.Cell(40, 8, fmt.Sprintf("Total Used: %.1f days", totalUsed))
	pdf.Ln(5)
	pdf.Cell(40, 8, fmt.Sprintf("Balance (Accrued - Used): %.1f days", calculatedBalance))
	pdf.Ln(5)
	pdf.Cell(40, 8, fmt.Sprintf("Total Current Balance: %.1f days", totalBalance))
	pdf.Ln(5)
	if totalBalance != calculatedBalance {
		carryOverDiff := totalBalance - calculatedBalance
		pdf.SetFont("Arial", "", 9)
		pdf.Cell(40, 6, fmt.Sprintf("(Difference: %.1f days - includes carry-over and manual adjustments)", carryOverDiff))
		pdf.Ln(5)
		pdf.SetFont("Arial", "B", 10)
	}
	pdf.SetFont("Arial", "", 8)
	pdf.Cell(40, 6, fmt.Sprintf("Generated: %s", time.Now().Format("2006-01-02 15:04:05")))

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// PrepareBalancesForExport converts balance data to export format
func PrepareBalancesForExport(employeeBalances []EmployeeBalanceData) []AnnualLeaveBalanceExport {
	exports := make([]AnnualLeaveBalanceExport, 0, len(employeeBalances))

	for _, balance := range employeeBalances {
		exports = append(exports, AnnualLeaveBalanceExport{
			EmployeeID:     balance.EmployeeID,
			EmployeeName:   balance.EmployeeName,
			Department:     balance.Department,
			TotalAccrued:   balance.TotalAccrued,
			TotalUsed:      balance.TotalUsed,
			CurrentBalance: balance.CurrentBalance,
			PendingLeaves:  balance.PendingLeaves,
			UpcomingLeaves: balance.UpcomingLeaves,
		})
	}

	return exports
}

// EmployeeAnnualLeaveReport represents single employee annual leave report data
type EmployeeAnnualLeaveReport struct {
	EmployeeID        uint
	EmployeeName      string
	Department        string
	TotalAccrued      float64
	TotalUsed         float64
	CurrentBalance    float64
	CarryOverBalance  float64
	AllTimeNetBalance float64
	Accruals          []AccrualExport
	ApprovedLeaves    []LeaveExport
}

// AccrualExport represents accrual data for export
type AccrualExport struct {
	Month       string
	DaysAccrued float64
	DaysUsed    float64
	DaysBalance float64
	IsProcessed bool
	ProcessedAt string
}

// LeaveExport represents leave data for export
type LeaveExport struct {
	StartDate string
	EndDate   string
	Duration  float64
	Reason    string
}

// ExportEmployeeAnnualLeaveToExcel exports single employee annual leave report to Excel
func ExportEmployeeAnnualLeaveToExcel(report EmployeeAnnualLeaveReport) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Annual Leave Report"
	f.NewSheet(sheetName)
	f.DeleteSheet("Sheet1")

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	subHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})

	// Employee Information
	row := 1
	f.SetCellValue(sheetName, "A1", "Employee Annual Leave Report")
	f.MergeCell(sheetName, "A1", "F1")
	f.SetCellStyle(sheetName, "A1", "F1", headerStyle)

	row = 3
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Employee Name:")
	f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), report.EmployeeName)
	row++
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Employee ID:")
	f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), report.EmployeeID)
	row++
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Department:")
	f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), report.Department)
	row++

	// Summary Section
	row++
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Summary")
	f.MergeCell(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row))
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), subHeaderStyle)
	row++
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Total Accrued:")
	f.SetCellFloat(sheetName, fmt.Sprintf("B%d", row), report.TotalAccrued, 2, 64)
	row++
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Total Used:")
	f.SetCellFloat(sheetName, fmt.Sprintf("B%d", row), report.TotalUsed, 2, 64)
	row++
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Current Balance:")
	f.SetCellFloat(sheetName, fmt.Sprintf("B%d", row), report.CurrentBalance, 2, 64)
	if report.CarryOverBalance > 0 {
		row++
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Carry-Over Balance:")
		f.SetCellFloat(sheetName, fmt.Sprintf("B%d", row), report.CarryOverBalance, 2, 64)
	}
	row++
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "All-Time Net Balance:")
	f.SetCellFloat(sheetName, fmt.Sprintf("B%d", row), report.AllTimeNetBalance, 2, 64)

	// Monthly Accrual History
	row += 2
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Monthly Accrual History")
	f.MergeCell(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("F%d", row))
	f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("F%d", row), subHeaderStyle)

	row++
	accrualHeaders := []string{"Month", "Days Accrued", "Days Used", "Days Balance", "Processed", "Processed At"}
	accrualHeaderStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#D0D0D0"}, Pattern: 1},
	})
	for i, header := range accrualHeaders {
		cell := fmt.Sprintf("%c%d", 'A'+i, row)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, accrualHeaderStyle)
	}

	// Set column widths for accrual table
	f.SetColWidth(sheetName, "A", "A", 15)
	f.SetColWidth(sheetName, "B", "F", 15)

	for _, acc := range report.Accruals {
		row++
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), acc.Month)
		f.SetCellFloat(sheetName, fmt.Sprintf("B%d", row), acc.DaysAccrued, 2, 64)
		f.SetCellFloat(sheetName, fmt.Sprintf("C%d", row), acc.DaysUsed, 2, 64)
		f.SetCellFloat(sheetName, fmt.Sprintf("D%d", row), acc.DaysBalance, 2, 64)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), map[bool]string{true: "Yes", false: "No"}[acc.IsProcessed])
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), acc.ProcessedAt)
	}

	// Approved Leaves History
	if len(report.ApprovedLeaves) > 0 {
		row += 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), "Approved Leaves History")
		f.MergeCell(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row))
		f.SetCellStyle(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("D%d", row), subHeaderStyle)

		row++
		leaveHeaders := []string{"Start Date", "End Date", "Duration (Days)", "Reason"}
		for i, header := range leaveHeaders {
			cell := fmt.Sprintf("%c%d", 'A'+i, row)
			f.SetCellValue(sheetName, cell, header)
			f.SetCellStyle(sheetName, cell, cell, accrualHeaderStyle)
		}

		for _, leave := range report.ApprovedLeaves {
			row++
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), leave.StartDate)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), leave.EndDate)
			f.SetCellFloat(sheetName, fmt.Sprintf("C%d", row), leave.Duration, 2, 64)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), leave.Reason)
		}
	}

	// Timestamp
	row += 2
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), fmt.Sprintf("Generated: %s", time.Now().Format("2006-01-02 15:04:05")))

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ExportEmployeeAnnualLeaveToPDF exports single employee annual leave report to PDF
func ExportEmployeeAnnualLeaveToPDF(report EmployeeAnnualLeaveReport) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Title
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "Employee Annual Leave Report")
	pdf.Ln(12)

	// Employee Information
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(40, 8, "Employee Information")
	pdf.Ln(8)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(40, 6, fmt.Sprintf("Name: %s", report.EmployeeName))
	pdf.Ln(6)
	pdf.Cell(40, 6, fmt.Sprintf("ID: %d", report.EmployeeID))
	pdf.Ln(6)
	pdf.Cell(40, 6, fmt.Sprintf("Department: %s", report.Department))
	pdf.Ln(10)

	// Summary
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(40, 8, "Summary")
	pdf.Ln(8)
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(40, 6, fmt.Sprintf("Total Accrued: %.1f days", report.TotalAccrued))
	pdf.Ln(6)
	pdf.Cell(40, 6, fmt.Sprintf("Total Used: %.1f days", report.TotalUsed))
	pdf.Ln(6)
	pdf.Cell(40, 6, fmt.Sprintf("Current Balance: %.1f days", report.CurrentBalance))
	pdf.Ln(6)
	if report.CarryOverBalance > 0 {
		pdf.Cell(40, 6, fmt.Sprintf("Carry-Over Balance: %.1f days", report.CarryOverBalance))
		pdf.Ln(6)
	}
	pdf.Cell(40, 6, fmt.Sprintf("All-Time Net Balance: %.1f days", report.AllTimeNetBalance))
	pdf.Ln(10)

	// Monthly Accrual History
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(40, 8, "Monthly Accrual History")
	pdf.Ln(8)

	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(200, 200, 200)
	headers := []string{"Month", "Accrued", "Used", "Balance", "Processed"}
	colWidths := []float64{35, 25, 25, 25, 30}

	for i, header := range headers {
		pdf.CellFormat(colWidths[i], 7, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(7)

	pdf.SetFont("Arial", "", 8)
	pdf.SetFillColor(255, 255, 255)
	for _, acc := range report.Accruals {
		processed := "Yes"
		if !acc.IsProcessed {
			processed = "No"
		}
		pdf.CellFormat(colWidths[0], 6, acc.Month, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[1], 6, fmt.Sprintf("%.1f", acc.DaysAccrued), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colWidths[2], 6, fmt.Sprintf("%.1f", acc.DaysUsed), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colWidths[3], 6, fmt.Sprintf("%.1f", acc.DaysBalance), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colWidths[4], 6, processed, "1", 0, "C", false, 0, "")
		pdf.Ln(6)
	}

	// Approved Leaves History
	if len(report.ApprovedLeaves) > 0 {
		pdf.Ln(5)
		pdf.SetFont("Arial", "B", 12)
		pdf.Cell(40, 8, "Approved Leaves History")
		pdf.Ln(8)

		pdf.SetFont("Arial", "B", 9)
		pdf.SetFillColor(200, 200, 200)
		leaveHeaders := []string{"Start Date", "End Date", "Duration", "Reason"}
		leaveColWidths := []float64{40, 40, 30, 80}

		for i, header := range leaveHeaders {
			pdf.CellFormat(leaveColWidths[i], 7, header, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(7)

		pdf.SetFont("Arial", "", 8)
		pdf.SetFillColor(255, 255, 255)
		for _, leave := range report.ApprovedLeaves {
			reason := leave.Reason
			if len(reason) > 30 {
				reason = reason[:27] + "..."
			}
			pdf.CellFormat(leaveColWidths[0], 6, leave.StartDate, "1", 0, "L", false, 0, "")
			pdf.CellFormat(leaveColWidths[1], 6, leave.EndDate, "1", 0, "L", false, 0, "")
			pdf.CellFormat(leaveColWidths[2], 6, fmt.Sprintf("%.1f", leave.Duration), "1", 0, "R", false, 0, "")
			pdf.CellFormat(leaveColWidths[3], 6, reason, "1", 0, "L", false, 0, "")
			pdf.Ln(6)
		}
	}

	// Timestamp
	pdf.Ln(10)
	pdf.SetFont("Arial", "", 8)
	pdf.Cell(40, 6, fmt.Sprintf("Generated: %s", time.Now().Format("2006-01-02 15:04:05")))

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MonthlyLeaveReportData represents monthly leave report data matching CSV format
type MonthlyLeaveReportData struct {
	Number     int     // Row number
	Name       string  // Employee name
	Position   string  // Employee position
	Opening    float64 // Opening balance (previous month's NET)
	DaysEarned float64 // Days earned this month (2.0)
	Total      float64 // Opening + Days Earned
	DaysTaken  float64 // Days taken this month
	Net        float64 // Total - Days Taken (final balance)
}

// GetMonthlyLeaveReport generates monthly leave report data for a specific month
func GetMonthlyLeaveReport(month time.Time, annualLeaveTypeID uint) ([]MonthlyLeaveReportData, error) {
	monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	prevMonthStart := monthStart.AddDate(0, -1, 0)

	// Get all employees (excluding admins)
	var employees []models.Employee
	database.DB.Where("role != ?", models.RoleAdmin).Order("firstname, lastname").Find(&employees)

	reportData := make([]MonthlyLeaveReportData, 0, len(employees))

	for i, emp := range employees {
		// Get accrual for the specified month
		var currentAccrual models.LeaveAccrual
		err := database.DB.Where("employee_id = ? AND leave_type_id = ? AND accrual_month = ?",
			emp.ID, annualLeaveTypeID, monthStart).First(&currentAccrual).Error

		var opening, daysEarned, daysTaken, total, net float64

		if err == nil {
			// Accrual exists for this month
			daysEarned = currentAccrual.DaysAccrued
			daysTaken = currentAccrual.DaysUsed
			net = currentAccrual.DaysBalance

			// Get previous month's balance for opening
			var prevAccrual models.LeaveAccrual
			prevErr := database.DB.Where("employee_id = ? AND leave_type_id = ? AND accrual_month = ?",
				emp.ID, annualLeaveTypeID, prevMonthStart).First(&prevAccrual).Error
			if prevErr == nil {
				opening = prevAccrual.DaysBalance
			} else {
				// No previous accrual, opening is 0
				opening = 0
			}
			total = opening + daysEarned
		} else {
			// No accrual for this month - calculate from scratch
			// Get previous month's balance
			var prevAccrual models.LeaveAccrual
			prevErr := database.DB.Where("employee_id = ? AND leave_type_id = ? AND accrual_month = ?",
				emp.ID, annualLeaveTypeID, prevMonthStart).First(&prevAccrual).Error
			if prevErr == nil {
				opening = prevAccrual.DaysBalance
			} else {
				opening = 0
			}

			// Calculate days earned (should be 2.0 for annual leave)
			daysEarned = AnnualLeaveDaysPerMonth

			// Calculate days taken in this month
			daysTaken = CalculateDaysUsedInMonth(emp.ID, annualLeaveTypeID, monthStart)

			total = opening + daysEarned
			net = total - daysTaken
		}

		// Get employee position
		position := "N/A"
		if emp.PositionID != nil {
			var pos models.Position
			if err := database.DB.First(&pos, *emp.PositionID).Error; err == nil {
				position = pos.Title
			}
		}
		if position == "N/A" && emp.Department != "" {
			position = emp.Department
		}

		reportData = append(reportData, MonthlyLeaveReportData{
			Number:     i + 1,
			Name:       emp.Firstname + " " + emp.Lastname,
			Position:   position,
			Opening:    opening,
			DaysEarned: daysEarned,
			Total:      total,
			DaysTaken:  daysTaken,
			Net:        net,
		})
	}

	return reportData, nil
}

// ExportMonthlyLeaveReportToExcel exports monthly leave report to Excel format matching CSV structure
func ExportMonthlyLeaveReportToExcel(reportData []MonthlyLeaveReportData, month time.Time, organizationName string) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Monthly Leave Report"
	f.NewSheet(sheetName)
	f.DeleteSheet("Sheet1")

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 12,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#4472C4"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	// Title rows (matching CSV format)
	monthName := month.Format("January 2006")
	f.SetCellValue(sheetName, "C2", fmt.Sprintf("%s STAFF LEAVE DAYS", organizationName))
	f.SetCellValue(sheetName, "C3", fmt.Sprintf("FOR THE MONTH OF %s", monthName))

	// Column headers (row 4, matching CSV)
	headers := []string{"", "NAME", "POSITION", "OPENING", "DAYS EARNED", "TOTAL", "DAYS TAKEN", "NET"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c4", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
		if i > 0 { // Skip first empty column
			f.SetCellStyle(sheetName, cell, cell, headerStyle)
		}
	}

	// Set column widths
	f.SetColWidth(sheetName, "A", "A", 5)  // Number
	f.SetColWidth(sheetName, "B", "B", 30) // Name
	f.SetColWidth(sheetName, "C", "C", 25) // Position
	f.SetColWidth(sheetName, "D", "H", 15) // Data columns

	// Write data (starting from row 5)
	for i, data := range reportData {
		row := i + 5
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), data.Number)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), data.Name)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), data.Position)

		// Format opening balance (can be negative)
		if data.Opening == 0 {
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), " -  ")
		} else {
			f.SetCellFloat(sheetName, fmt.Sprintf("D%d", row), data.Opening, 1, 64)
		}

		f.SetCellFloat(sheetName, fmt.Sprintf("E%d", row), data.DaysEarned, 1, 64)

		// Format total (can be negative or zero)
		if data.Total == 0 {
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), " -  ")
		} else {
			f.SetCellFloat(sheetName, fmt.Sprintf("F%d", row), data.Total, 1, 64)
		}

		// Days taken - leave empty if 0 (matching CSV format)
		if data.DaysTaken > 0 {
			f.SetCellFloat(sheetName, fmt.Sprintf("G%d", row), data.DaysTaken, 1, 64)
		}

		// Format net balance (can be negative or zero)
		if data.Net == 0 {
			f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), " -  ")
		} else {
			f.SetCellFloat(sheetName, fmt.Sprintf("H%d", row), data.Net, 1, 64)
		}
	}

	// Add timestamp
	timestampRow := len(reportData) + 6
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", timestampRow), fmt.Sprintf("Generated: %s", time.Now().Format("2006-01-02 15:04:05")))

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EmployeeDataExport represents employee data for export
type EmployeeDataExport struct {
	ID                        uint
	EmployeeNumber            string
	NRC                       string
	Username                  string
	Firstname                 string
	Lastname                  string
	Email                     string
	Department                string
	Role                      string
	Phone                     string
	Mobile                    string
	Address                   string
	City                      string
	PostalCode                string
	DateOfBirth               string
	Gender                    string
	JobTitle                  string
	EmploymentStatus          string
	StartDate                 string
	Tenure                    string
	EmergencyContactName      string
	EmergencyContactPhone     string
	EmergencyContactRelationship string
	BankName                  string
	BankAccountNumber         string
	TaxID                     string
	Notes                     string
}

// ExportEmployeesToPDF exports all employees data to PDF
func ExportEmployeesToPDF(employees []EmployeeDataExport) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetTitle("Employee Directory", false)
	pdf.SetAuthor("HRMS System", false)
	pdf.SetCreator("HRMS API", false)

	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "Employee Directory")
	pdf.Ln(10)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(0, 6, fmt.Sprintf("Generated: %s", time.Now().Format("2006-01-02 15:04:05")))
	pdf.Ln(8)

	// Table headers
	pdf.SetFont("Arial", "B", 8)
	headers := []string{"Name", "NRC", "Email", "Department", "Role", "Phone", "Start Date"}
	colWidths := []float64{40, 30, 45, 30, 20, 30, 25}
	
	// Header row
	for i, header := range headers {
		pdf.CellFormat(colWidths[i], 7, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Data rows
	pdf.SetFont("Arial", "", 7)
	for _, emp := range employees {
		pdf.CellFormat(colWidths[0], 6, fmt.Sprintf("%s %s", emp.Firstname, emp.Lastname), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[1], 6, emp.NRC, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[2], 6, emp.Email, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[3], 6, emp.Department, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[4], 6, emp.Role, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[5], 6, emp.Mobile, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[6], 6, emp.StartDate, "1", 0, "L", false, 0, "")
		pdf.Ln(-1)
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ExportEmployeeToPDF exports single employee detailed data to PDF
func ExportEmployeeToPDF(emp EmployeeDataExport) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(fmt.Sprintf("Employee Details - %s %s", emp.Firstname, emp.Lastname), false)
	pdf.SetAuthor("HRMS System", false)
	pdf.SetCreator("HRMS API", false)

	pdf.AddPage()
	pdf.SetFont("Arial", "B", 18)
	pdf.Cell(0, 10, "Employee Details")
	pdf.Ln(12)

	// Basic Information
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 8, "Basic Information")
	pdf.Ln(6)
	pdf.SetFont("Arial", "", 10)
	
	basicInfo := [][]string{
		{"Name:", fmt.Sprintf("%s %s", emp.Firstname, emp.Lastname)},
		{"NRC/Username:", emp.NRC},
		{"Email:", emp.Email},
		{"Department:", emp.Department},
		{"Role:", emp.Role},
		{"Position:", emp.JobTitle},
		{"Employment Status:", emp.EmploymentStatus},
		{"Start Date:", emp.StartDate},
		{"Tenure:", emp.Tenure},
	}

	for _, row := range basicInfo {
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(50, 6, row[0])
		pdf.SetFont("Arial", "", 10)
		pdf.Cell(0, 6, row[1])
		pdf.Ln(5)
	}

	// Personal Information
	pdf.Ln(3)
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 8, "Personal Information")
	pdf.Ln(6)
	pdf.SetFont("Arial", "", 10)

	personalInfo := [][]string{
		{"Date of Birth:", emp.DateOfBirth},
		{"Gender:", emp.Gender},
		{"Address:", emp.Address},
		{"City:", emp.City},
		{"Postal Code:", emp.PostalCode},
		{"Phone:", emp.Phone},
		{"Mobile:", emp.Mobile},
	}

	for _, row := range personalInfo {
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(50, 6, row[0])
		pdf.SetFont("Arial", "", 10)
		pdf.Cell(0, 6, row[1])
		pdf.Ln(5)
	}

	// Emergency Contact
	pdf.Ln(3)
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 8, "Emergency Contact")
	pdf.Ln(6)
	pdf.SetFont("Arial", "", 10)

	emergencyInfo := [][]string{
		{"Contact Name:", emp.EmergencyContactName},
		{"Contact Phone:", emp.EmergencyContactPhone},
		{"Relationship:", emp.EmergencyContactRelationship},
	}

	for _, row := range emergencyInfo {
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(50, 6, row[0])
		pdf.SetFont("Arial", "", 10)
		pdf.Cell(0, 6, row[1])
		pdf.Ln(5)
	}

	// Financial Information
	pdf.Ln(3)
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 8, "Financial Information")
	pdf.Ln(6)
	pdf.SetFont("Arial", "", 10)

	financialInfo := [][]string{
		{"Bank Name:", emp.BankName},
		{"Bank Account Number:", emp.BankAccountNumber},
		{"Tax ID:", emp.TaxID},
	}

	for _, row := range financialInfo {
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(50, 6, row[0])
		pdf.SetFont("Arial", "", 10)
		pdf.Cell(0, 6, row[1])
		pdf.Ln(5)
	}

	// Notes
	if emp.Notes != "" && emp.Notes != "-" {
		pdf.Ln(3)
		pdf.SetFont("Arial", "B", 12)
		pdf.Cell(0, 8, "Additional Notes")
		pdf.Ln(6)
		pdf.SetFont("Arial", "", 10)
		pdf.MultiCell(0, 6, emp.Notes, "", "", false)
	}

	pdf.Ln(5)
	pdf.SetFont("Arial", "", 8)
	pdf.Cell(0, 6, fmt.Sprintf("Generated: %s", time.Now().Format("2006-01-02 15:04:05")))

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
