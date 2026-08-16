package fastag

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Config holds connection settings for the FASTag aggregator API.
type Config struct {
	Endpoint string
	APIKey   string
	Enabled  bool
}

// Balance represents the wallet balance linked to a FASTag.
type Balance struct {
	VehicleNumber string    `json:"vehicle_number"`
	TagID         string    `json:"tag_id"`
	Balance       float64   `json:"balance"`
	Currency      string    `json:"currency"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// DeductTollRequest carries toll deduction inputs.
type DeductTollRequest struct {
	VehicleNumber string  `json:"vehicle_number"`
	TagID         string  `json:"tag_id"`
	PlazaID       string  `json:"plaza_id"`
	PlazaName     string  `json:"plaza_name"`
	Amount        float64 `json:"amount"`
	TripID        string  `json:"trip_id"`
}

// TollTransaction represents a toll deduction record.
type TollTransaction struct {
	TransactionID string    `json:"transaction_id"`
	VehicleNumber string    `json:"vehicle_number"`
	TagID         string    `json:"tag_id"`
	PlazaID       string    `json:"plaza_id"`
	PlazaName     string    `json:"plaza_name"`
	Amount        float64   `json:"amount"`
	Timestamp     time.Time `json:"timestamp"`
	Status        string    `json:"status"`
}

// Client defines operations supported by the FASTag aggregator API.
type Client interface {
	GetBalance(ctx context.Context, vehicleNumber, tagID string) (Balance, error)
	DeductToll(ctx context.Context, req DeductTollRequest) (TollTransaction, error)
	ListTransactions(ctx context.Context, vehicleNumber string, limit int) ([]TollTransaction, error)
}

type stubClient struct {
	cfg Config
}

// NewClient returns a stub FASTag client that logs calls and returns fake data.
func NewClient(cfg Config) Client {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://api.fastag.org"
	}
	return &stubClient{cfg: cfg}
}

func (c *stubClient) GetBalance(ctx context.Context, vehicleNumber, tagID string) (Balance, error) {
	slog.Default().Info("[fastag] GetBalance called", "endpoint", c.cfg.Endpoint, "enabled", c.cfg.Enabled, "vehicle", vehicleNumber, "tag", tagID)
	if !c.cfg.Enabled {
		return Balance{}, fmt.Errorf("fastag integration disabled")
	}
	return Balance{
		VehicleNumber: vehicleNumber,
		TagID:         tagID,
		Balance:       2475.50,
		Currency:      "INR",
		UpdatedAt:     time.Now(),
	}, nil
}

func (c *stubClient) DeductToll(ctx context.Context, req DeductTollRequest) (TollTransaction, error) {
	slog.Default().Info("[fastag] DeductToll called", "endpoint", c.cfg.Endpoint, "enabled", c.cfg.Enabled, "vehicle", req.VehicleNumber, "plaza", req.PlazaName, "amount", req.Amount)
	if !c.cfg.Enabled {
		return TollTransaction{}, fmt.Errorf("fastag integration disabled")
	}
	return TollTransaction{
		TransactionID: uuid.New().String(),
		VehicleNumber: req.VehicleNumber,
		TagID:         req.TagID,
		PlazaID:       req.PlazaID,
		PlazaName:     req.PlazaName,
		Amount:        req.Amount,
		Timestamp:     time.Now(),
		Status:        "SUCCESS",
	}, nil
}

func (c *stubClient) ListTransactions(ctx context.Context, vehicleNumber string, limit int) ([]TollTransaction, error) {
	slog.Default().Info("[fastag] ListTransactions called", "endpoint", c.cfg.Endpoint, "enabled", c.cfg.Enabled, "vehicle", vehicleNumber, "limit", limit)
	if !c.cfg.Enabled {
		return nil, fmt.Errorf("fastag integration disabled")
	}
	if limit <= 0 {
		limit = 10
	}
	now := time.Now()
	txs := make([]TollTransaction, limit)
	for i := 0; i < limit; i++ {
		txs[i] = TollTransaction{
			TransactionID: uuid.New().String(),
			VehicleNumber: vehicleNumber,
			TagID:         "TAG" + vehicleNumber,
			PlazaID:       fmt.Sprintf("PLZ%03d", i+1),
			PlazaName:     fmt.Sprintf("Toll Plaza %d", i+1),
			Amount:        85.00 + float64(i)*5,
			Timestamp:     now.Add(-time.Duration(i) * time.Hour),
			Status:        "SUCCESS",
		}
	}
	return txs, nil
}
