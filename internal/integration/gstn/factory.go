package gstn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// realHttpClient performs live NIC/GSTN calls when !UseMock && APIKey != "".
type realHttpClient struct {
	cfg        Config
	httpClient *http.Client
}

// NewClient returns a live HTTP client if !UseMock && APIKey != "", otherwise a stub.
// Factory keeps mock for CI and real for prod (Spec 21 §5).
func NewClient(cfg Config) Client {
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://api.gstn.org"
	}
	if !cfg.UseMock && cfg.APIKey != "" {
		return &realHttpClient{cfg: cfg, httpClient: &http.Client{Timeout: 10 * time.Second}}
	}
	return &stubClient{cfg: cfg, einvoice: NewMockEInvoiceClient(cfg)}
}

func (c *realHttpClient) ValidateGSTIN(ctx context.Context, gstin string) (GSTINDetails, error) {
	if !c.cfg.Enabled {
		return GSTINDetails{}, fmt.Errorf("gstn integration disabled")
	}
	body, _ := json.Marshal(map[string]string{"gstin": gstin})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint+"/gstn/validate", bytes.NewReader(body))
	if err != nil {
		return GSTINDetails{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.cfg.APIKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return GSTINDetails{}, fmt.Errorf("gstn_unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GSTINDetails{}, fmt.Errorf("gstn_unavailable: status %d", resp.StatusCode)
	}
	var out GSTINDetails
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return GSTINDetails{}, fmt.Errorf("gstn_unavailable: %w", err)
	}
	return out, nil
}

func (c *realHttpClient) FetchGSTR1Summary(ctx context.Context, gstin, period string) (GSTR1Summary, error) {
	if !c.cfg.Enabled {
		return GSTR1Summary{}, fmt.Errorf("gstn integration disabled")
	}
	url := fmt.Sprintf("%s/gstn/gstr1?gstin=%s&period=%s", c.cfg.Endpoint, gstin, period)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return GSTR1Summary{}, err
	}
	req.Header.Set("X-API-Key", c.cfg.APIKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return GSTR1Summary{}, fmt.Errorf("gstn_unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GSTR1Summary{}, fmt.Errorf("gstn_unavailable: status %d", resp.StatusCode)
	}
	var out GSTR1Summary
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return GSTR1Summary{}, fmt.Errorf("gstn_unavailable: %w", err)
	}
	return out, nil
}

func (c *realHttpClient) FetchGSTR3BSummary(ctx context.Context, gstin, period string) (GSTR3BSummary, error) {
	if !c.cfg.Enabled {
		return GSTR3BSummary{}, fmt.Errorf("gstn integration disabled")
	}
	url := fmt.Sprintf("%s/gstn/gstr3b?gstin=%s&period=%s", c.cfg.Endpoint, gstin, period)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return GSTR3BSummary{}, err
	}
	req.Header.Set("X-API-Key", c.cfg.APIKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return GSTR3BSummary{}, fmt.Errorf("gstn_unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GSTR3BSummary{}, fmt.Errorf("gstn_unavailable: status %d", resp.StatusCode)
	}
	var out GSTR3BSummary
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return GSTR3BSummary{}, fmt.Errorf("gstn_unavailable: %w", err)
	}
	return out, nil
}

func (c *realHttpClient) GenerateIRN(ctx context.Context, inv InvoiceView) (*IRNResponse, error) {
	if !c.cfg.Enabled {
		return nil, fmt.Errorf("gstn integration disabled")
	}
	body, err := json.Marshal(inv)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint+"/einvoice/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.cfg.APIKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gstn_unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gstn_unavailable: status %d", resp.StatusCode)
	}
	var out IRNResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("gstn_unavailable: %w", err)
	}
	return &out, nil
}

func (c *realHttpClient) PushEInvoice(ctx context.Context, invoiceID, irn string) (*PushResponse, error) {
	if !c.cfg.Enabled {
		return nil, fmt.Errorf("gstn integration disabled")
	}
	body, _ := json.Marshal(map[string]string{"invoice_id": invoiceID, "irn": irn})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.Endpoint+"/einvoice/push", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.cfg.APIKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gstn_unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gstn_unavailable: status %d", resp.StatusCode)
	}
	var out PushResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("gstn_unavailable: %w", err)
	}
	return &out, nil
}
