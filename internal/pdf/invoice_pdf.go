package pdf

import (
	"bytes"
	"fmt"
	"time"

	"github.com/unidoc/unipdf/v3/creator"

	"transport-app/internal/domain/invoice"
)

// GenerateInvoicePDF creates a professional PDF document for an invoice using UniPDF.
func GenerateInvoicePDF(inv invoice.Invoice, companyName string) ([]byte, error) {
	c := creator.New()
	c.SetPageMargins(40, 40, 40, 40)

	// Main Title / Header
	p := c.NewParagraph(fmt.Sprintf("%s - INVOICE", companyName))
	p.SetFontSize(24)
	p.SetColor(creator.ColorRGBFrom8bit(25, 51, 128))
	p.SetMargins(0, 0, 0, 20)
	if err := c.Draw(p); err != nil {
		return nil, err
	}

	// Invoice Info Meta Table
	table := c.NewTable(2)
	table.SetMargins(0, 0, 0, 20)

	cell1 := table.NewCell()
	cellText := c.NewParagraph(fmt.Sprintf("Invoice No: %s\nDate: %s\nStatus: %s",
		inv.InvoiceNumber,
		inv.CreatedAt.Format("2006-01-02"),
		inv.Status,
	))
	cellText.SetFontSize(11)
	_ = cell1.SetContent(cellText)

	cell2 := table.NewCell()
	_ = cell2.SetContent(c.NewParagraph(""))

	if err := c.Draw(table); err != nil {
		return nil, err
	}

	// Summary details
	subP := c.NewParagraph(fmt.Sprintf("Subtotal: $%.2f", inv.Subtotal))
	taxP := c.NewParagraph(fmt.Sprintf("Tax: $%.2f", inv.Tax))
	discP := c.NewParagraph(fmt.Sprintf("Discount: $%.2f", inv.Discount))
	totP := c.NewParagraph(fmt.Sprintf("Total: $%.2f", inv.Total))
	totP.SetFontSize(14)
	totP.SetColor(creator.ColorRGBFrom8bit(0, 128, 25))

	balP := c.NewParagraph(fmt.Sprintf("Outstanding Balance: $%.2f", inv.OutstandingBalance()))
	balP.SetFontSize(12)

	_ = c.Draw(subP)
	_ = c.Draw(taxP)
	_ = c.Draw(discP)
	_ = c.Draw(totP)
	_ = c.Draw(balP)

	// Footer timestamp
	footer := c.NewParagraph(fmt.Sprintf("Generated automatically on %s", time.Now().Format("2006-01-02 15:04:05")))
	footer.SetFontSize(9)
	footer.SetMargins(0, 0, 40, 0)
	_ = c.Draw(footer)

	var buf bytes.Buffer
	if err := c.Write(&buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
