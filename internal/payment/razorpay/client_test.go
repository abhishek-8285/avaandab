package razorpay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVerifyPaymentSignature_KnownVector proves the HMAC-SHA256 signature
// verification accepts a signature computed independently over "order|payment"
// (Spec 11 §5.1). A tampered signature must fail.
func TestVerifyPaymentSignature_KnownVector(t *testing.T) {
	keySecret := "1234567890abcdef"
	orderID := "order_1"
	paymentID := "pay_2"

	client := NewRazorpayClient("key_id", keySecret)

	expected := hmac.New(sha256.New, []byte(keySecret))
	expected.Write([]byte(orderID + "|" + paymentID))
	sig := hex.EncodeToString(expected.Sum(nil))

	assert.True(t, client.VerifyPaymentSignature(orderID, paymentID, sig), "valid signature must verify")
	assert.False(t, client.VerifyPaymentSignature(orderID, paymentID, sig+"00"), "tampered signature must fail")
	assert.False(t, client.VerifyPaymentSignature(orderID, paymentID, "zz-not-hex"), "non-hex signature must fail")

	unconfigured := NewRazorpayClient("", "")
	assert.False(t, unconfigured.VerifyPaymentSignature(orderID, paymentID, sig), "no key secret must fail closed")
}

// fakeOrderAPI records the payload sent to Razorpay and returns a canned order.
type fakeOrderAPI struct {
	got map[string]interface{}
}

func (f *fakeOrderAPI) Create(data map[string]interface{}, _ map[string]string) (map[string]interface{}, error) {
	f.got = data
	return map[string]interface{}{"id": "order_test_1", "status": "created"}, nil
}

// TestCreateOrder_SetsInvoiceNote proves the order payload carries
// notes.invoice_id and the correct paise amount — the field that lets the
// webhook attribute a payment to an invoice (Spec 11 §5.1).
func TestCreateOrder_SetsInvoiceNote(t *testing.T) {
	api := &fakeOrderAPI{}
	client := NewRazorpayClient("key_id", "key_secret")
	client.orderAPI = api

	order, err := client.CreateOrder("inv_123", 10, "INR")
	require.NoError(t, err)
	require.NotNil(t, order)

	require.Equal(t, "order_test_1", order.ID)
	require.Equal(t, int64(1000), order.Amount, "10 INR must become 1000 paise")

	notes, ok := api.got["notes"].(map[string]interface{})
	require.True(t, ok, "order payload must include notes")
	assert.Equal(t, "inv_123", notes["invoice_id"], "notes.invoice_id must equal the invoice id")
	assert.Equal(t, int64(1000), api.got["amount"])
	assert.Equal(t, "INR", api.got["currency"])
}

// TestCreateOrder_NotConfigured proves order creation fails closed when
// credentials are absent.
func TestCreateOrder_NotConfigured(t *testing.T) {
	client := NewRazorpayClient("", "")
	_, err := client.CreateOrder("inv_1", 10, "INR")
	require.ErrorIs(t, err, ErrNotConfigured)
}
