package customer_health

import (
	"fmt"
	"time"
)

type CustomerHealthFactors struct {
	LastLoginDays       int     `json:"last_login_days"`
	BookingsCount30d    int     `json:"bookings_count_30d"`
	TripsCompleted30d   int     `json:"trips_completed_30d"`
	ActiveUsersCount    int     `json:"active_users_count"`
	ErrorsEncounteredd  int     `json:"errors_encountered"`
	DaysUntilTrialExpiry int    `json:"days_until_trial_expiry"`
	IsTrial             bool    `json:"is_trial"`
}

type CustomerHealthResult struct {
	CompanyID   string    `json:"company_id"`
	CompanyName string    `json:"company_name"`
	Score       int       `json:"score"` // 0 to 100
	Status      string    `json:"status"` // Healthy, At Risk, Critical
	Reasons     []string  `json:"reasons"`
	SuggestedAction string `json:"suggested_action"`
	CalculatedAt time.Time `json:"calculated_at"`
}

// CalculateHealthScore produces a Customer Health Score (0-100%) based on platform telemetry
func CalculateHealthScore(companyID, companyName string, factors CustomerHealthFactors) CustomerHealthResult {
	score := 100
	reasons := make([]string, 0)
	action := "Regular follow-up"

	// 1. Recency penalty
	if factors.LastLoginDays > 14 {
		score -= 40
		reasons = append(reasons, fmt.Sprintf("No login for %d days", factors.LastLoginDays))
		action = "Call customer immediately"
	} else if factors.LastLoginDays > 7 {
		score -= 20
		reasons = append(reasons, fmt.Sprintf("No login for %d days", factors.LastLoginDays))
		action = "Send re-engagement email"
	}

	// 2. Core usage penalty
	if factors.BookingsCount30d == 0 {
		score -= 30
		reasons = append(reasons, "No booking created")
		if action != "Call customer immediately" {
			action = "Offer onboarding / demo call"
		}
	}

	if factors.TripsCompleted30d == 0 {
		score -= 15
		reasons = append(reasons, "No trips completed")
	}

	// 3. Trial urgency
	if factors.IsTrial {
		if factors.DaysUntilTrialExpiry <= 2 {
			score -= 15
			reasons = append(reasons, fmt.Sprintf("Trial ends in %d days", factors.DaysUntilTrialExpiry))
		}
	}

	if score < 0 {
		score = 0
	}

	status := "Healthy"
	if score < 40 {
		status = "Critical"
	} else if score < 70 {
		status = "At Risk"
	}

	return CustomerHealthResult{
		CompanyID:       companyID,
		CompanyName:     companyName,
		Score:           score,
		Status:          status,
		Reasons:         reasons,
		SuggestedAction: action,
		CalculatedAt:    time.Now(),
	}
}
