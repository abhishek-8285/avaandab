package razorpay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyPaymentSignature(t *testing.T) {
	keySecret := "test_secret_12345"
	orderID := "order_N8Kx2yX132s"
	paymentID := "pay_N8Kx8721ksx"

	// Generate expected HMAC-SHA256 signature
	data := orderID + "|" + paymentID
	h := hmac.New(sha256.New, []byte(keySecret))
	h.Write([]byte(data))
	validSignature := hex.EncodeToString(h.Sum(nil))

	client := NewRazorpayClient("key_id", keySecret)

	if !client.VerifyPaymentSignature(orderID, paymentID, validSignature) {
		t.Errorf("Expected signature verification to succeed")
	}

	if client.VerifyPaymentSignature(orderID, paymentID, "invalid_sig") {
		t.Errorf("Expected invalid signature verification to fail")
	}
}
