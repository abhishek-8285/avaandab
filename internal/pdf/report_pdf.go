package pdf

import (
	"bytes"
	"fmt"

	"github.com/go-pdf/fpdf"
)

// GenerateReportPDF creates a clean, tabular summary PDF report.
func GenerateReportPDF(title, companyName string, header []string, rows [][]string) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	// Company Name
	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(25, 51, 128)
	pdf.CellFormat(0, 10, companyName, "", 1, "L", false, 0, "")

	// Title
	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(70, 70, 70)
	pdf.CellFormat(0, 8, title, "", 1, "L", false, 0, "")
	pdf.Ln(4)

	if len(header) == 0 {
		var buf bytes.Buffer
		if err := pdf.Output(&buf); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	// Table Header
	pdf.SetFillColor(240, 243, 246)
	pdf.SetTextColor(30, 30, 30)
	pdf.SetFont("Arial", "B", 9)
	colWidth := 180.0 / float64(len(header))
	for _, h := range header {
		pdf.CellFormat(colWidth, 8, h, "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)

	// Table Rows
	pdf.SetTextColor(50, 50, 50)
	pdf.SetFont("Arial", "", 8)
	for _, row := range rows {
		for _, c := range row {
			val := c
			if len(val) > 40 {
				val = val[:37] + "..." // Truncate to prevent cell overflow
			}
			pdf.CellFormat(colWidth, 7, val, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf generation failed: %w", err)
	}
	return buf.Bytes(), nil
}
