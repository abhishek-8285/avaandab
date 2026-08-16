package ewaybill

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Config holds connection settings for the NIC E-way bill API.
type Config struct {
	Endpoint string
	APIKey   string
	Enabled  bool
}

// GenerateRequest carries the inputs needed to create an E-way bill.
type GenerateRequest struct {
	DocumentNumber string  `json:"document_number"`
	FromGSTIN      string  `json:"from_gstin"`
	ToGSTIN        string  `json:"to_gstin"`
	TransporterID  string  `json:"transporter_id"`
	VehicleNumber  string  `json:"vehicle_number"`
	Distance       int     `json:"distance"`
	TaxAmount      float64 `json:"tax_amount"`
	TotalAmount    float64 `json:"total_amount"`
}

// EWayBill represents a generated or retrieved E-way bill.
type EWayBill struct {
	EwbNumber   string    `json:"ewb_number"`
	Status      string    `json:"status"`
	GeneratedAt time.Time `json:"generated_at"`
	ValidUpto   time.Time `json:"valid_upto"`
	QRCode      string    `json:"qr_code"`
	DocumentRef string    `json:"document_ref"`
}

// Cancellation represents the result of cancelling an E-way bill.
type Cancellation struct {
	EwbNumber   string    `json:"ewb_number"`
	CancelledAt time.Time `json:"cancelled_at"`
	Reason      string    `json:"reason"`
	Status      string    `json:"status"`
}

// Client defines operations supported by the NIC E-way bill API.
type Client interface {
	Generate(ctx context.Context, req GenerateRequest) (EWayBill, error)
	Get(ctx context.Context, ewbNumber string) (EWayBill, error)
	Cancel(ctx context.Context, ewbNumber, reason string) (Cancellation, error)
}

type stubClient struct {
	cfg Config
}

// NewClient returns a stub E-way bill client that logs calls and returns fake data.
func NewClient(cfg Config) Client {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://ewaybill.nic.in/api"
	}
	return &stubClient{cfg: cfg}
}

func (c *stubClient) Generate(ctx context.Context, req GenerateRequest) (EWayBill, error) {
	slog.Default().Info("[ewaybill] Generate called", "endpoint", c.cfg.Endpoint, "enabled", c.cfg.Enabled, "document", req.DocumentNumber)
	if !c.cfg.Enabled {
		return EWayBill{}, fmt.Errorf("ewaybill integration disabled")
	}
	now := time.Now()
	return EWayBill{
		EwbNumber:   "EWB" + uuid.New().String()[:12],
		Status:      "ACTIVE",
		GeneratedAt: now,
		ValidUpto:   now.Add(24 * time.Hour),
		QRCode:      "data:image/png;base64,stubqrcode",
		DocumentRef: req.DocumentNumber,
	}, nil
}

func (c *stubClient) Get(ctx context.Context, ewbNumber string) (EWayBill, error) {
	slog.Default().Info("[ewaybill] Get called", "endpoint", c.cfg.Endpoint, "enabled", c.cfg.Enabled, "ewb_number", ewbNumber)
	if !c.cfg.Enabled {
		return EWayBill{}, fmt.Errorf("ewaybill integration disabled")
	}
	return EWayBill{
		EwbNumber:   ewbNumber,
		Status:      "ACTIVE",
		GeneratedAt: time.Now().Add(-2 * time.Hour),
		ValidUpto:   time.Now().Add(22 * time.Hour),
		QRCode:      "data:image/png;base64,stubqrcode",
		DocumentRef: "DOC/2026/0001",
	}, nil
}

func (c *stubClient) Cancel(ctx context.Context, ewbNumber, reason string) (Cancellation, error) {
	slog.Default().Info("[ewaybill] Cancel called", "endpoint", c.cfg.Endpoint, "enabled", c.cfg.Enabled, "ewb_number", ewbNumber, "reason", reason)
	if !c.cfg.Enabled {
		return Cancellation{}, fmt.Errorf("ewaybill integration disabled")
	}
	return Cancellation{
		EwbNumber:   ewbNumber,
		CancelledAt: time.Now(),
		Reason:      reason,
		Status:      "CANCELLED",
	}, nil
}
