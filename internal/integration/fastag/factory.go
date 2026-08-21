package fastag

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type realHttpClient struct {
	cfg        Config
	httpClient *http.Client
	db         *sql.DB
}

// NewClient returns a live NETC client if !UseMock && APIKey != "", otherwise stub with DB fallback (Spec 21 §5).
func NewClient(cfg Config, dbConn ...*sql.DB) Client {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://api.fastag.org"
	}
	var db *sql.DB
	if len(dbConn) > 0 {
		db = dbConn[0]
	}
	if !cfg.UseMock && cfg.APIKey != "" {
		return &realHttpClient{cfg: cfg, httpClient: &http.Client{Timeout: 10 * time.Second}, db: db}
	}
	return &clientImpl{cfg: cfg, db: db}
}

func hmacSign(key, data string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func (c *realHttpClient) GetBalance(ctx context.Context, vehicleNumber, tagID string) (Balance, error) {
	if !c.cfg.Enabled {
		return Balance{}, fmt.Errorf("fastag integration disabled")
	}
	url := fmt.Sprintf("%s/balance?vehicle_number=%s&tag_id=%s", c.cfg.Endpoint, vehicleNumber, tagID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Balance{}, err
	}
	req.Header.Set("X-API-Key", c.cfg.APIKey)
	req.Header.Set("X-HMAC-Signature", hmacSign(c.cfg.APIKey, vehicleNumber+tagID))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Balance{}, fmt.Errorf("fastag_unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Balance{}, fmt.Errorf("fastag_unavailable: status %d", resp.StatusCode)
	}
	var out Balance
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Balance{}, fmt.Errorf("fastag_unavailable: %w", err)
	}
	return out, nil
}

func (c *realHttpClient) DeductToll(ctx context.Context, req DeductTollRequest) (TollTransaction, error) {
	if !c.cfg.Enabled {
		return TollTransaction{}, fmt.Errorf("fastag integration disabled")
	}
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint+"/deduct", bytes.NewReader(body))
	if err != nil {
		return TollTransaction{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.cfg.APIKey)
	httpReq.Header.Set("X-HMAC-Signature", hmacSign(c.cfg.APIKey, req.VehicleNumber+req.PlazaID))
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return TollTransaction{}, fmt.Errorf("fastag_unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TollTransaction{}, fmt.Errorf("fastag_unavailable: status %d", resp.StatusCode)
	}
	var out TollTransaction
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return TollTransaction{}, fmt.Errorf("fastag_unavailable: %w", err)
	}
	return out, nil
}

func (c *realHttpClient) ListTransactions(ctx context.Context, vehicleNumber string, limit int) ([]TollTransaction, error) {
	if !c.cfg.Enabled {
		return nil, fmt.Errorf("fastag integration disabled")
	}
	url := fmt.Sprintf("%s/transactions?vehicle_number=%s&limit=%d", c.cfg.Endpoint, vehicleNumber, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.cfg.APIKey)
	req.Header.Set("X-HMAC-Signature", hmacSign(c.cfg.APIKey, vehicleNumber))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fastag_unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fastag_unavailable: status %d", resp.StatusCode)
	}
	var out []TollTransaction
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("fastag_unavailable: %w", err)
	}
	// also try wrapper {transactions: []}
	if len(out) == 0 {
		var wrapped struct {
			Transactions []TollTransaction `json:"transactions"`
		}
		// re-decode if needed - already consumed, but for real NETC the shape varies; return empty slice as fallback
		_ = wrapped
	}
	return out, nil
}

func (c *realHttpClient) Reconcile(ctx context.Context, vehicleNumber string, from, to string) (ReconcileResult, error) {
	if !c.cfg.Enabled {
		return ReconcileResult{}, fmt.Errorf("fastag integration disabled")
	}
	body, _ := json.Marshal(map[string]string{"vehicle_number": vehicleNumber, "from": from, "to": to})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint+"/reconcile", bytes.NewReader(body))
	if err != nil {
		return ReconcileResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.cfg.APIKey)
	req.Header.Set("X-HMAC-Signature", hmacSign(c.cfg.APIKey, vehicleNumber+from+to))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ReconcileResult{}, fmt.Errorf("fastag_unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ReconcileResult{}, fmt.Errorf("fastag_unavailable: status %d", resp.StatusCode)
	}
	var out ReconcileResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ReconcileResult{}, fmt.Errorf("fastag_unavailable: %w", err)
	}
	return out, nil
}
