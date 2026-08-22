package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"

	"transport-app/internal/domain/invoice"
)

// GenerateInvoicePDF generates a high-performance, open-source (MIT) professional PDF document for an invoice.
func GenerateInvoicePDF(inv invoice.Invoice, companyName string) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()

	// ── Company Branding Header ──────────────────────────────────────
	pdf.SetFont("Arial", "B", 20)
	pdf.SetTextColor(25, 51, 128) // Corporate Blue
	pdf.CellFormat(0, 10, companyName, "", 1, "L", false, 0, "")

	pdf.SetFont("Arial", "B", 12)
	pdf.SetTextColor(100, 100, 100)
	pdf.CellFormat(0, 6, "INVOICE STATEMENT", "", 1, "L", false, 0, "")
	pdf.Ln(4)

	// Divider line
	pdf.SetDrawColor(220, 220, 220)
	pdf.SetLineWidth(0.5)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(6)

	// ── Invoice Metadata Section ─────────────────────────────────────
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(50, 50, 50)
	pdf.CellFormat(90, 6, fmt.Sprintf("Invoice Number: %s", inv.InvoiceNumber), "", 0, "L", false, 0, "")
	pdf.CellFormat(90, 6, fmt.Sprintf("Status: %s", inv.Status), "", 1, "R", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(90, 6, fmt.Sprintf("Date Created: %s", inv.CreatedAt.Format("2006-01-02 15:04")), "", 0, "L", false, 0, "")
	pdf.CellFormat(90, 6, fmt.Sprintf("Customer ID: %s", string(inv.CustomerID)), "", 1, "R", false, 0, "")
	pdf.Ln(8)

	// ── Financial Breakdown Table ─────────────────────────────────────
	// Header row
	pdf.SetFillColor(240, 243, 246)
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(30, 30, 30)
	pdf.CellFormat(120, 8, "Description", "1", 0, "L", true, 0, "")
	pdf.CellFormat(60, 8, "Amount (Rs.)", "1", 1, "R", true, 0, "")

	// Items
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(50, 50, 50)

	pdf.CellFormat(120, 7, "Subtotal", "1", 0, "L", false, 0, "")
	pdf.CellFormat(60, 7, fmt.Sprintf("%.2f", inv.Subtotal), "1", 1, "R", false, 0, "")

	pdf.CellFormat(120, 7, "Tax", "1", 0, "L", false, 0, "")
	pdf.CellFormat(60, 7, fmt.Sprintf("%.2f", inv.Tax), "1", 1, "R", false, 0, "")

	pdf.CellFormat(120, 7, "Discount", "1", 0, "L", false, 0, "")
	pdf.CellFormat(60, 7, fmt.Sprintf("-%.2f", inv.Discount), "1", 1, "R", false, 0, "")

	// Total Row
	pdf.SetFont("Arial", "B", 11)
	pdf.SetFillColor(245, 247, 250)
	pdf.CellFormat(120, 8, "Total Amount", "1", 0, "L", true, 0, "")
	pdf.SetTextColor(0, 128, 25)
	pdf.CellFormat(60, 8, fmt.Sprintf("%.2f", inv.Total), "1", 1, "R", true, 0, "")

	// Paid Amount Row
	pdf.SetFont("Arial", "", 10)
	pdf.SetTextColor(50, 50, 50)
	pdf.CellFormat(120, 7, "Paid Amount", "1", 0, "L", false, 0, "")
	pdf.CellFormat(60, 7, fmt.Sprintf("%.2f", inv.PaidAmount), "1", 1, "R", false, 0, "")

	// Outstanding Balance Row
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(120, 8, "Outstanding Balance Due", "1", 0, "L", true, 0, "")

	bal := inv.OutstandingBalance()
	if bal > 0 {
		pdf.SetTextColor(200, 30, 30) // Red
	} else {
		pdf.SetTextColor(0, 128, 25) // Green
	}
	pdf.CellFormat(60, 8, fmt.Sprintf("%.2f", bal), "1", 1, "R", true, 0, "")

	pdf.Ln(15)

	// ── Footer ─────────────────────────────────────────────────────────
	pdf.SetFont("Arial", "I", 9)
	pdf.SetTextColor(128, 128, 128)
	pdf.CellFormat(0, 6, fmt.Sprintf("Generated automatically on %s | Thank you for your business!", time.Now().Format("2006-01-02 15:04:05")), "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
