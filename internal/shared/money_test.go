package shared

import "testing"

func TestFloatToMoney_RoundsInsteadOfTruncating(t *testing.T) {
	tests := []struct {
		in   float64
		want int64
	}{
		{19.99, 1999},
		{0.1 + 0.2, 30},
		{19.9955, 2000},
		{0.01, 1},
		{10.0, 1000},
		{0, 0},
	}
	for _, tt := range tests {
		got := FloatToMoney(tt.in, "INR").Amount
		if got != tt.want {
			t.Errorf("FloatToMoney(%v).Amount = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestFloatToMoney_RoundTrip(t *testing.T) {
	m := FloatToMoney(19.99, "INR")
	if m.MoneyToFloat() != 19.99 {
		t.Errorf("MoneyToFloat() = %v, want 19.99", m.MoneyToFloat())
	}
}
