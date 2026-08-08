package razorpay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	rzp "github.com/razorpay/razorpay-go"
)

type RazorpayClient struct {
	client    *rzp.Client
	keyID     string
	keySecret string
}

type Order struct {
	ID       string `json:"id"`
	Entity   string `json:"entity"`
	Amount   int64  `json:"amount"` // in paise
	Currency string `json:"currency"`
	Receipt  string `json:"receipt"`
	Status   string `json:"status"`
}

func NewRazorpayClient(keyID, keySecret string) *RazorpayClient {
	var client *rzp.Client
	if keyID != "" && keySecret != "" {
		client = rzp.NewClient(keyID, keySecret)
	}
	return &RazorpayClient{
		client:    client,
		keyID:     keyID,
		keySecret: keySecret,
	}
}

// CreateOrder generates a Razorpay Order ID for checkout
func (c *RazorpayClient) CreateOrder(invoiceID string, amountINR float64, currency string) (*Order, error) {
	if c.client == nil {
		return nil, errors.New("razorpay credentials not configured")
	}

	amountInPaise := int64(amountINR * 100)
	if currency == "" {
		currency = "INR"
	}

	data := map[string]interface{}{
		"amount":   amountInPaise,
		"currency": currency,
		"receipt":  fmt.Sprintf("rcpt_%s", invoiceID),
		"notes": map[string]interface{}{
			"invoice_id": invoiceID,
		},
	}

	body, err := c.client.Order.Create(data, nil)
	if err != nil {
		return nil, err
	}

	orderID, _ := body["id"].(string)
	status, _ := body["status"].(string)

	return &Order{
		ID:       orderID,
		Amount:   amountInPaise,
		Currency: currency,
		Receipt:  fmt.Sprintf("rcpt_%s", invoiceID),
		Status:   status,
	}, nil
}

// VerifyPaymentSignature validates the cryptographic signature returned by Razorpay Checkout
func (c *RazorpayClient) VerifyPaymentSignature(orderID, paymentID, signature string) bool {
	if c.keySecret == "" {
		return false
	}

	data := orderID + "|" + paymentID
	h := hmac.New(sha256.New, []byte(c.keySecret))
	h.Write([]byte(data))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	return expectedSignature == signature
}
