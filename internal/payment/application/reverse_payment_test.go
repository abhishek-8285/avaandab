package application

import (
	"testing"

	paymentagg "transport-app/internal/payment/domain/aggregate"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReversePayment_RetryIsIdempotent(t *testing.T) {
	ctx, uow, _, reverseUC, _, clock, _ := setupWebhookUnitTest(t)

	invID := seedInvoice(t, uow.repos.invoices, clock, 1000)

	payID := paymentagg.PaymentID("pay-orig-1")
	original := paymentagg.NewPaymentAggregate(
		payID,
		"1",
		string(invID),
		clock.Now(),
		400,
		paymentagg.PaymentMethodCash,
		nil,
		nil,
		clock.Now(),
	)
	require.NoError(t, uow.repos.payments.Save(ctx, original))

	inv, err := uow.repos.invoices.Find(ctx, invID, "1")
	require.NoError(t, err)
	require.NoError(t, inv.ApplyPayment(400, clock.Now()))
	require.NoError(t, uow.repos.invoices.Save(ctx, inv))

	cmd := ReversePaymentCommand{TenantID: "1", OriginalPayID: payID, Reason: "customer refund"}

	id1, err := reverseUC.Execute(ctx, cmd)
	require.NoError(t, err)
	require.NotEmpty(t, id1)

	invAfterFirst, err := uow.repos.invoices.Find(ctx, invID, "1")
	require.NoError(t, err)
	assert.Equal(t, 0.0, invAfterFirst.PaidAmount, "first reversal decrements paid amount to zero")

	id2, err := reverseUC.Execute(ctx, cmd)
	require.NoError(t, err)
	assert.Equal(t, id1, id2, "retried reversal must return the existing reversal id")

	invAfterRetry, err := uow.repos.invoices.Find(ctx, invID, "1")
	require.NoError(t, err)
	assert.Equal(t, 0.0, invAfterRetry.PaidAmount, "retry must not double-decrement paid amount")
}

func TestReversePayment_ValidationErrors(t *testing.T) {
	ctx, _, _, reverseUC, _, _, _ := setupWebhookUnitTest(t)

	_, err := reverseUC.Execute(ctx, ReversePaymentCommand{TenantID: "1", Reason: "r"})
	assert.ErrorIs(t, err, ErrMissingOriginalPayment)

	_, err = reverseUC.Execute(ctx, ReversePaymentCommand{TenantID: "1", OriginalPayID: "p1"})
	assert.ErrorIs(t, err, ErrMissingReversalReason)
}
