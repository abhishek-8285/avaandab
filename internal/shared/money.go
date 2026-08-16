package shared

import (
	"errors"
	"math"
)

// Money represents a monetary value with a currency.
type Money struct {
	Amount   int64  // In minor units (e.g. cents)
	Currency string // e.g. "USD", "INR"
}

// NewMoney creates a new Money instance.
func NewMoney(amount int64, currency string) (Money, error) {
	if currency == "" {
		return Money{}, errors.New("currency is required")
	}
	return Money{Amount: amount, Currency: currency}, nil
}

// FloatToMoney converts a float64 to minor units (multiplies by 100).
func FloatToMoney(val float64, currency string) Money {
	return Money{
		Amount:   int64(math.Round(val * 100)),
		Currency: currency,
	}
}

// MoneyToFloat converts Money minor units to float64.
func (m Money) MoneyToFloat() float64 {
	return float64(m.Amount) / 100.0
}

// Add sums two Money instances of the same currency.
func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, errors.New("currencies must match to perform addition")
	}
	return Money{Amount: m.Amount + other.Amount, Currency: m.Currency}, nil
}

// Subtract subtracts one Money instance from another.
func (m Money) Subtract(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, errors.New("currencies must match to perform subtraction")
	}
	return Money{Amount: m.Amount - other.Amount, Currency: m.Currency}, nil
}

// Multiply multiplies Money by a multiplier factor.
func (m Money) Multiply(multiplier int64) Money {
	return Money{Amount: m.Amount * multiplier, Currency: m.Currency}
}

// Equals checks if two Money instances are equal.
func (m Money) Equals(other Money) bool {
	return m.Amount == other.Amount && m.Currency == other.Currency
}

// IsZero checks if the money amount is zero.
func (m Money) IsZero() bool {
	return m.Amount == 0
}
