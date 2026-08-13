package expense

import (
	"errors"
	"strings"
	"time"

	"transport-app/internal/domain/types"
)

type ExpenseType string

const (
	ExpenseTypeFuel    ExpenseType = "fuel"
	ExpenseTypeToll    ExpenseType = "toll"
	ExpenseTypeFood    ExpenseType = "food"
	ExpenseTypeRepair  ExpenseType = "repair"
	ExpenseTypeAdvance ExpenseType = "advance"
)

var (
	ErrInvalidExpenseType = errors.New("invalid expense type")
	ErrNegativeAmount     = errors.New("expense amount must be greater than zero")
)

// DriverExpense represents an itemized driver kharcha claim.
type DriverExpense struct {
	ID          string
	TripID      types.TripID
	DriverID    types.DriverID
	ExpenseType ExpenseType
	Amount      float64
	Description *string
	ReceiptURL  *string
	Approved    bool
	CreatedAt   time.Time
}

// Approve marks the expense as approved by dispatch/accountant.
func (e *DriverExpense) Approve() {
	e.Approved = true
}

// Validate validates expense fields.
func (e DriverExpense) Validate() error {
	if e.Amount <= 0 {
		return ErrNegativeAmount
	}
	switch strings.ToLower(string(e.ExpenseType)) {
	case "fuel", "toll", "food", "repair", "advance":
		return nil
	default:
		return ErrInvalidExpenseType
	}
}
