package razorpay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	rzp "github.com/razorpay/razorpay-go"
)

// ErrNotConfigured is returned when Razorpay credentials are absent.
var ErrNotConfigured = errors.New("razorpay credentials not configured")

// orderAPI is the subset of the Razorpay SDK used to create orders. It is an
// interface so unit tests can fake the network call.
type orderAPI interface {
	Create(data map[string]interface{}, extraHeaders map[string]string) (map[string]interface{}, error)
}

// OrderCreator creates a Razorpay order for checkout. Satisfied by
// *RazorpayClient; faked in unit tests (Spec 11 §5.1).
type OrderCreator interface {
	CreateOrder(invoiceID string, amountINR float64, currency string) (*Order, error)
}

// SignatureVerifier validates the HMAC signature returned by Razorpay
// Checkout. Satisfied by *RazorpayClient; faked in unit tests.
type SignatureVerifier interface {
	VerifyPaymentSignature(orderID, paymentID, signature string) bool
}

// Client is the Razorpay payment-gateway gateway, and implements both
// OrderCreator and SignatureVerifier.
var _ OrderCreator = (*RazorpayClient)(nil)
var _ SignatureVerifier = (*RazorpayClient)(nil)

type RazorpayClient struct {
	orderAPI  orderAPI
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
	c := &RazorpayClient{
		keyID:     keyID,
		keySecret: keySecret,
	}
	if keyID != "" && keySecret != "" {
		client := rzp.NewClient(keyID, keySecret)
		c.orderAPI = client.Order
	}
	return c
}

// CreateOrder generates a Razorpay Order ID for checkout.
func (c *RazorpayClient) CreateOrder(invoiceID string, amountINR float64, currency string) (*Order, error) {
	if c.orderAPI == nil {
		return nil, ErrNotConfigured
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

	body, err := c.orderAPI.Create(data, nil)
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
	expectedSignature := h.Sum(nil)
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}

	return hmac.Equal(expectedSignature, sigBytes)
}
