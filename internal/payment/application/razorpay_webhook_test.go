package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"transport-app/internal/events"
	invoiceDomain "transport-app/internal/invoice/domain"
	invoiceagg "transport-app/internal/invoice/domain/aggregate"
	"transport-app/internal/payment/domain"
	paymentagg "transport-app/internal/payment/domain/aggregate"
	"transport-app/internal/shared"
	"transport-app/internal/shared/ports"
)

const testWebhookSecret = "whsec_test"

// ---- fakes ----

type fakeRepoProvider struct {
	payments *fakePaymentRepo
	invoices *fakeInvoiceRepo
}

func (f *fakeRepoProvider) Bookings() any  { return nil }
func (f *fakeRepoProvider) Trips() any     { return nil }
func (f *fakeRepoProvider) Drivers() any   { return nil }
func (f *fakeRepoProvider) Vehicles() any  { return nil }
func (f *fakeRepoProvider) Invoices() any  { return f.invoices }
func (f *fakeRepoProvider) Payments() any  { return f.payments }
func (f *fakeRepoProvider) AuditLogs() any { return nil }

type fakeTxContext struct {
	context.Context
	repos *fakeRepoProvider
}

func (tx *fakeTxContext) Repositories() ports.RepositoryProvider { return tx.repos }

type fakeUnitOfWork struct {
	repos *fakeRepoProvider
}

func (u *fakeUnitOfWork) Execute(ctx context.Context, fn func(ports.TxContext) error) error {
	return fn(&fakeTxContext{Context: ctx, repos: u.repos})
}

func newFakeUnitOfWork() *fakeUnitOfWork {
	return &fakeUnitOfWork{
		repos: &fakeRepoProvider{
			payments: &fakePaymentRepo{
				byID:  make(map[paymentagg.PaymentID]*paymentagg.PaymentAggregate),
				byRef: make(map[string]paymentagg.PaymentID),
			},
			invoices: &fakeInvoiceRepo{
				byID: make(map[invoiceagg.InvoiceID]*invoiceagg.InvoiceAggregate),
			},
		},
	}
}

type fakePaymentRepo struct {
	mu    sync.Mutex
	byID  map[paymentagg.PaymentID]*paymentagg.PaymentAggregate
	byRef map[string]paymentagg.PaymentID
}

func (r *fakePaymentRepo) Save(_ context.Context, p *paymentagg.PaymentAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[p.ID] = p
	if p.Reference != nil && *p.Reference != "" {
		r.byRef[*p.Reference] = p.ID
	}
	return nil
}

func (r *fakePaymentRepo) Find(_ context.Context, id paymentagg.PaymentID, _ shared.TenantID) (*paymentagg.PaymentAggregate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.byID[id]; ok {
		return p, nil
	}
	return nil, sql.ErrNoRows
}

func (r *fakePaymentRepo) FindByReference(_ context.Context, reference string, _ shared.TenantID) (paymentagg.PaymentID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.byRef[reference]; ok {
		return id, nil
	}
	return "", sql.ErrNoRows
}

func (r *fakePaymentRepo) GetReadModel(_ context.Context, _ paymentagg.PaymentID, _ shared.TenantID) (domain.PaymentReadModel, error) {
	return domain.PaymentReadModel{}, nil
}

func (r *fakePaymentRepo) GetPaymentsByInvoice(_ context.Context, invoiceID string, _ shared.TenantID) ([]domain.PaymentReadModel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []domain.PaymentReadModel
	for _, p := range r.byID {
		if p.InvoiceID == invoiceID {
			ref := ""
			if p.Reference != nil {
				ref = *p.Reference
			}
			out = append(out, domain.PaymentReadModel{
				ID:        string(p.ID),
				InvoiceID: p.InvoiceID,
				Amount:    p.Amount,
				Method:    string(p.Method),
				Reference: &ref,
			})
		}
	}
	return out, nil
}

func (r *fakePaymentRepo) SearchReadModels(_ context.Context, _ shared.TenantID, _ string, _, _ int) ([]domain.PaymentReadModel, int64, error) {
	return nil, 0, nil
}

type fakeInvoiceRepo struct {
	mu   sync.Mutex
	byID map[invoiceagg.InvoiceID]*invoiceagg.InvoiceAggregate
}

func (r *fakeInvoiceRepo) Save(_ context.Context, inv *invoiceagg.InvoiceAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[inv.ID] = inv
	return nil
}

func (r *fakeInvoiceRepo) Find(_ context.Context, id invoiceagg.InvoiceID, _ shared.TenantID) (*invoiceagg.InvoiceAggregate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if inv, ok := r.byID[id]; ok {
		return inv, nil
	}
	return nil, sql.ErrNoRows
}

func (r *fakeInvoiceRepo) FindByBookingID(_ context.Context, _ string, _ shared.TenantID) (*invoiceagg.InvoiceAggregate, error) {
	return nil, sql.ErrNoRows
}

func (r *fakeInvoiceRepo) GetReadModel(_ context.Context, _ invoiceagg.InvoiceID, _ shared.TenantID) (invoiceDomain.InvoiceReadModel, error) {
	return invoiceDomain.InvoiceReadModel{}, nil
}

func (r *fakeInvoiceRepo) SearchReadModels(_ context.Context, _ shared.TenantID, _, _ string, _, _ int) ([]invoiceDomain.InvoiceReadModel, int64, error) {
	return nil, 0, nil
}

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time { return c.now }

type fakeIDGen struct {
	next int
}

func (g *fakeIDGen) GenerateUUID() string {
	g.next++
	return fmt.Sprintf("uuid-%d", g.next)
}

func (g *fakeIDGen) GenerateDisplayID(prefix string) string {
	g.next++
	return fmt.Sprintf("%s-%d", prefix, g.next)
}

// ---- helpers ----

func signWebhook(t *testing.T, body []byte, secret string) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func capturedWebhookBody(paymentID, invoiceID string, amountPaise int64) []byte {
	payload := map[string]interface{}{
		"event": "payment.captured",
		"payload": map[string]interface{}{
			"payment": map[string]interface{}{
				"entity": map[string]interface{}{
					"id":       paymentID,
					"order_id": "order_123",
					"amount":   amountPaise,
					"currency": "INR",
					"status":   "captured",
					"notes": map[string]interface{}{
						"invoice_id": invoiceID,
					},
				},
			},
		},
	}
	b, _ := json.Marshal(payload)
	return b
}

func refundWebhookBody(refundID, paymentID string, amountPaise int64) []byte {
	payload := map[string]interface{}{
		"event": "refund.processed",
		"payload": map[string]interface{}{
			"refund": map[string]interface{}{
				"entity": map[string]interface{}{
					"id":         refundID,
					"payment_id": paymentID,
					"amount":     amountPaise,
					"currency":   "INR",
					"status":     "processed",
				},
			},
		},
	}
	b, _ := json.Marshal(payload)
	return b
}

func failedWebhookBody(paymentID, invoiceID string, amountPaise int64) []byte {
	payload := map[string]interface{}{
		"event": "payment.failed",
		"payload": map[string]interface{}{
			"payment": map[string]interface{}{
				"entity": map[string]interface{}{
					"id":                paymentID,
					"order_id":          "order_123",
					"amount":            amountPaise,
					"currency":          "INR",
					"status":            "failed",
					"error_code":        "BAD_REQUEST_ERROR",
					"error_description": "Payment failed",
					"notes": map[string]interface{}{
						"invoice_id": invoiceID,
					},
				},
			},
		},
	}
	b, _ := json.Marshal(payload)
	return b
}

func seedInvoice(t *testing.T, repo *fakeInvoiceRepo, clock *fakeClock, total float64) invoiceagg.InvoiceID {
	t.Helper()
	id := invoiceagg.InvoiceID(fmt.Sprintf("inv-%d", time.Now().UnixNano()))
	inv := invoiceagg.NewInvoiceAggregate(
		id,
		"1",
		"INV-001",
		"bk-1",
		"cust-1",
		nil,
		total-180,
		180,
		0,
		total,
		invoiceagg.PaymentStatusPending,
		clock.Now(),
	)
	require.NoError(t, repo.Save(context.Background(), inv))
	return id
}

func setupWebhookUnitTest(t *testing.T) (context.Context, *fakeUnitOfWork, *RecordPaymentUseCase, *ReversePaymentUseCase, *RazorpayWebhookUseCase, *fakeClock, *fakeIDGen) {
	t.Helper()
	clock := &fakeClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	idGen := &fakeIDGen{}
	uow := newFakeUnitOfWork()
	recordUC := NewRecordPaymentUseCase(uow, idGen, clock)
	reverseUC := NewReversePaymentUseCase(uow, idGen, clock)
	webhookUC := NewRazorpayWebhookUseCase(recordUC, uow, testWebhookSecret, clock)
	webhookUC.SetReversePaymentUseCase(reverseUC)
	return context.Background(), uow, recordUC, reverseUC, webhookUC, clock, idGen
}

// ---- tests ----

func TestRazorpayWebhook_VerifySignature(t *testing.T) {
	clock := &fakeClock{now: time.Now()}
	body := capturedWebhookBody("pay_sig", "inv_sig", 10000)

	uc := NewRazorpayWebhookUseCase(nil, nil, testWebhookSecret, clock)
	sig := signWebhook(t, body, testWebhookSecret)

	assert.NoError(t, uc.VerifySignature(body, sig))
	assert.ErrorIs(t, uc.VerifySignature(body, "deadbeef"), ErrWebhookInvalidSignature)
	assert.ErrorIs(t, uc.VerifySignature(body, ""), ErrWebhookInvalidSignature)

	emptyUC := NewRazorpayWebhookUseCase(nil, nil, "", clock)
	assert.ErrorIs(t, emptyUC.VerifySignature(body, sig), ErrWebhookNotConfigured)
}

func TestRazorpayWebhook_ExecuteEvent_Idempotency(t *testing.T) {
	ctx, uow, _, _, webhookUC, clock, _ := setupWebhookUnitTest(t)
	invID := seedInvoice(t, uow.repos.invoices, clock, 1000)

	body := capturedWebhookBody("pay_idem_1", string(invID), 100000)
	sig := signWebhook(t, body, testWebhookSecret)

	id1, err := webhookUC.ExecuteEvent(ctx, body, sig, "evt_idem_1", RazorpayWebhookEvent{})
	require.NoError(t, err)
	require.NotEmpty(t, id1)

	id2, err := webhookUC.ExecuteEvent(ctx, body, sig, "evt_idem_1", RazorpayWebhookEvent{})
	require.NoError(t, err)
	assert.Equal(t, id1, id2, "duplicate event id must return the same payment id")

	status := webhookUC.Status()
	assert.Equal(t, int64(1), status.Counts["payment.captured"])
}

func TestRazorpayWebhook_Execute_PaymentCaptured(t *testing.T) {
	ctx, uow, _, _, webhookUC, clock, _ := setupWebhookUnitTest(t)
	invID := seedInvoice(t, uow.repos.invoices, clock, 1000)

	body := capturedWebhookBody("pay_capture_1", string(invID), 100000)
	sig := signWebhook(t, body, testWebhookSecret)

	id, err := webhookUC.ExecuteEvent(ctx, body, sig, "evt_capture_1", RazorpayWebhookEvent{})
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	payments, err := uow.repos.payments.GetPaymentsByInvoice(ctx, string(invID), "1")
	require.NoError(t, err)
	require.Len(t, payments, 1)
	assert.Equal(t, 1000.0, payments[0].Amount)
	assert.Equal(t, "razorpay", payments[0].Method)
	assert.Equal(t, "pay_capture_1", *payments[0].Reference)
}

func TestRazorpayWebhook_Execute_RefundProcessed(t *testing.T) {
	ctx, uow, recordUC, _, webhookUC, clock, _ := setupWebhookUnitTest(t)
	invID := seedInvoice(t, uow.repos.invoices, clock, 1000)

	originalRef := "pay_original_1"
	_, err := recordUC.Execute(ctx, RecordPaymentCommand{
		TenantID:    "1",
		InvoiceID:   string(invID),
		PaymentDate: clock.Now(),
		Amount:      1000,
		Method:      paymentagg.PaymentMethodRazorpay,
		Reference:   &originalRef,
	})
	require.NoError(t, err)

	body := refundWebhookBody("rfnd_1", originalRef, 100000)
	sig := signWebhook(t, body, testWebhookSecret)

	reversalID, err := webhookUC.ExecuteEvent(ctx, body, sig, "evt_refund_1", RazorpayWebhookEvent{})
	require.NoError(t, err)
	assert.NotEmpty(t, reversalID)

	payments, err := uow.repos.payments.GetPaymentsByInvoice(ctx, string(invID), "1")
	require.NoError(t, err)
	require.Len(t, payments, 2)

	var reversal domain.PaymentReadModel
	for _, p := range payments {
		if p.Amount < 0 {
			reversal = p
		}
	}
	assert.Equal(t, -1000.0, reversal.Amount)
}

func TestRazorpayWebhook_Execute_PaymentFailedEmitsEvent(t *testing.T) {
	ctx, uow, _, _, webhookUC, clock, _ := setupWebhookUnitTest(t)
	invID := seedInvoice(t, uow.repos.invoices, clock, 1000)

	bus := events.NewInMemoryBus()
	var captured events.Event
	bus.Subscribe("RazorpayPaymentFailed", func(_ context.Context, e events.Event) error {
		captured = e
		return nil
	})
	webhookUC.SetEventBus(bus)

	body := failedWebhookBody("pay_fail_1", string(invID), 100000)
	sig := signWebhook(t, body, testWebhookSecret)

	_, err := webhookUC.ExecuteEvent(ctx, body, sig, "evt_fail_1", RazorpayWebhookEvent{})
	require.NoError(t, err)

	require.Equal(t, "RazorpayPaymentFailed", captured.Type)
	payload, ok := captured.Payload.(RazorpayPaymentFailedEvent)
	require.True(t, ok)
	assert.Equal(t, "pay_fail_1", payload.RazorpayPaymentID)
	assert.Equal(t, "evt_fail_1", payload.RazorpayEventID)
}
