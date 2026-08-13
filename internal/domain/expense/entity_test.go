package expense_test

import (
	"testing"
	"time"

	"transport-app/internal/domain/expense"
	"transport-app/internal/domain/types"
)

func TestDriverExpense_ApproveAndValidate(t *testing.T) {
	desc := "Diesel refuel at Highway Pump"
	url := "https://avandab.com/receipts/fuel-1.jpg"

	exp := expense.DriverExpense{
		ID:          "exp-1",
		TripID:      types.TripID("trp-1"),
		DriverID:    types.DriverID("drv-1"),
		ExpenseType: expense.ExpenseTypeFuel,
		Amount:      500.0,
		Description: &desc,
		ReceiptURL:  &url,
		Approved:    false,
		CreatedAt:   time.Now(),
	}

	if err := exp.Validate(); err != nil {
		t.Fatalf("expected valid expense, got %v", err)
	}

	exp.Approve()
	if !exp.Approved {
		t.Fatalf("expected expense to be approved")
	}

	// Negative amount
	expNeg := exp
	expNeg.Amount = -50.0
	if err := expNeg.Validate(); err != expense.ErrNegativeAmount {
		t.Fatalf("expected ErrNegativeAmount, got %v", err)
	}

	// Invalid type
	expBadType := exp
	expBadType.ExpenseType = "entertainment"
	if err := expBadType.Validate(); err != expense.ErrInvalidExpenseType {
		t.Fatalf("expected ErrInvalidExpenseType, got %v", err)
	}
}
