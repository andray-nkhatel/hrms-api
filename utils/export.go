package utils

import (
	"bytes"
	"fmt"
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
	headers := []string{"Employee ID", "Employee Name", "Department", "Total Accrued", "Total Used", "Current Balance", "Pending Leaves", "Upcoming Leaves"}
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
	f.SetColWidth(sheetName, "D", "H", 15)

	// Write data
	for i, balance := range balances {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), balance.EmployeeID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), balance.EmployeeName)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), balance.Department)
		f.SetCellFloat(sheetName, fmt.Sprintf("D%d", row), balance.TotalAccrued, 2, 64)
		f.SetCellFloat(sheetName, fmt.Sprintf("E%d", row), balance.TotalUsed, 2, 64)
		f.SetCellFloat(sheetName, fmt.Sprintf("F%d", row), balance.CurrentBalance, 2, 64)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), balance.PendingLeaves)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), balance.UpcomingLeaves)
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
	f.SetCellStyle(sheetName, fmt.Sprintf("B%d", summaryRow), fmt.Sprintf("H%d", summaryRow), summaryStyle)

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
	headers := []string{"ID", "Employee Name", "Department", "Accrued", "Used", "Balance", "Pending", "Upcoming"}
	colWidths := []float64{15, 50, 40, 25, 25, 25, 25, 25}

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
		pdf.CellFormat(colWidths[6], 7, fmt.Sprintf("%d", balance.PendingLeaves), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[7], 7, fmt.Sprintf("%d", balance.UpcomingLeaves), "1", 0, "C", false, 0, "")
		pdf.Ln(7)
	}

	// Add summary
	pdf.Ln(5)
	pdf.SetFont("Arial", "B", 10)
	var totalAccrued, totalUsed, totalBalance float64
	for _, balance := range balances {
		totalAccrued += balance.TotalAccrued
		totalUsed += balance.TotalUsed
		totalBalance += balance.CurrentBalance
	}
	pdf.Cell(40, 8, fmt.Sprintf("Total Employees: %d", len(balances)))
	pdf.Ln(5)
	pdf.Cell(40, 8, fmt.Sprintf("Total Accrued: %.1f days", totalAccrued))
	pdf.Ln(5)
	pdf.Cell(40, 8, fmt.Sprintf("Total Used: %.1f days", totalUsed))
	pdf.Ln(5)
	pdf.Cell(40, 8, fmt.Sprintf("Total Balance: %.1f days", totalBalance))
	pdf.Ln(5)
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
