package accounting

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Config holds connection settings for the external accounting API.
type Config struct {
	Endpoint string
	APIKey   string
	Enabled  bool
}

// LineItem represents a single invoice line item.
type LineItem struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Amount      float64 `json:"amount"`
}

// ExportedInvoice represents an invoice to be pushed to the accounting system.
type ExportedInvoice struct {
	ExternalID    string     `json:"external_id"`
	InvoiceNumber string     `json:"invoice_number"`
	CustomerName  string     `json:"customer_name"`
	CustomerGSTIN string     `json:"customer_gstin"`
	Amount        float64    `json:"amount"`
	TaxAmount     float64    `json:"tax_amount"`
	TotalAmount   float64    `json:"total_amount"`
	InvoiceDate   time.Time  `json:"invoice_date"`
	DueDate       time.Time  `json:"due_date"`
	LineItems     []LineItem `json:"line_items"`
}

// ExportResult represents the response from exporting an invoice.
type ExportResult struct {
	SyncID     string `json:"sync_id"`
	Status     string `json:"status"`
	ExternalID string `json:"external_id"`
	Message    string `json:"message"`
}

// Contact represents a customer or vendor contact.
type Contact struct {
	ExternalID  string `json:"external_id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	GSTIN       string `json:"gstin"`
	Address     string `json:"address"`
	ContactType string `json:"contact_type"`
}

// SyncResult represents the response from syncing contacts.
type SyncResult struct {
	Synced  int      `json:"synced"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors"`
	Message string   `json:"message"`
}

// JournalLine represents a debit/credit line in a journal entry.
type JournalLine struct {
	Account string  `json:"account"`
	Debit   float64 `json:"debit"`
	Credit  float64 `json:"credit"`
}

// JournalEntry represents a journal entry to push to the accounting system.
type JournalEntry struct {
	EntryDate time.Time     `json:"entry_date"`
	Reference string        `json:"reference"`
	Narration string        `json:"narration"`
	Lines     []JournalLine `json:"lines"`
}

// JournalEntryResult represents the response from pushing a journal entry.
type JournalEntryResult struct {
	EntryID string `json:"entry_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Client defines operations supported by the external accounting API.
type Client interface {
	ExportInvoice(ctx context.Context, invoice ExportedInvoice) (ExportResult, error)
	SyncContacts(ctx context.Context, contacts []Contact) (SyncResult, error)
	PushJournalEntry(ctx context.Context, entry JournalEntry) (JournalEntryResult, error)
}

type stubClient struct {
	cfg Config
}

// NewClient returns a stub accounting client that logs calls and returns fake data.
func NewClient(cfg Config) Client {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://api.accounting.example.com"
	}
	return &stubClient{cfg: cfg}
}

func (c *stubClient) ExportInvoice(ctx context.Context, invoice ExportedInvoice) (ExportResult, error) {
	slog.Default().Info("[accounting] ExportInvoice called", "endpoint", c.cfg.Endpoint, "enabled", c.cfg.Enabled, "invoice", invoice.InvoiceNumber)
	if !c.cfg.Enabled {
		return ExportResult{}, fmt.Errorf("accounting integration disabled")
	}
	return ExportResult{
		SyncID:     uuid.New().String(),
		Status:     "SUCCESS",
		ExternalID: "EXT-" + invoice.InvoiceNumber,
		Message:    "Invoice exported successfully",
	}, nil
}

func (c *stubClient) SyncContacts(ctx context.Context, contacts []Contact) (SyncResult, error) {
	slog.Default().Info("[accounting] SyncContacts called", "endpoint", c.cfg.Endpoint, "enabled", c.cfg.Enabled, "count", len(contacts))
	if !c.cfg.Enabled {
		return SyncResult{}, fmt.Errorf("accounting integration disabled")
	}
	return SyncResult{
		Synced:  len(contacts),
		Failed:  0,
		Errors:  nil,
		Message: fmt.Sprintf("Synced %d contacts", len(contacts)),
	}, nil
}

func (c *stubClient) PushJournalEntry(ctx context.Context, entry JournalEntry) (JournalEntryResult, error) {
	slog.Default().Info("[accounting] PushJournalEntry called", "endpoint", c.cfg.Endpoint, "enabled", c.cfg.Enabled, "reference", entry.Reference)
	if !c.cfg.Enabled {
		return JournalEntryResult{}, fmt.Errorf("accounting integration disabled")
	}
	return JournalEntryResult{
		EntryID: uuid.New().String(),
		Status:  "SUCCESS",
		Message: "Journal entry pushed successfully",
	}, nil
}
