package customer_health

import (
	"testing"
)

func TestCalculateHealthScore(t *testing.T) {
	factors := CustomerHealthFactors{
		LastLoginDays:        12,
		BookingsCount30d:     0,
		TripsCompleted30d:    0,
		ActiveUsersCount:     1,
		ErrorsEncounteredd:   0,
		DaysUntilTrialExpiry: 2,
		IsTrial:              true,
	}

	result := CalculateHealthScore("comp_123", "ABC Logistics", factors)

	if result.Score > 40 {
		t.Errorf("Expected score to be critical (<40), got %d", result.Score)
	}

	if result.Status != "Critical" {
		t.Errorf("Expected status Critical, got %s", result.Status)
	}

	if len(result.Reasons) == 0 {
		t.Errorf("Expected health risk reasons to be populated")
	}
}
