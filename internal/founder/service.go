package founder

import (
	"context"
	"fmt"
	"time"

	"transport-app/internal/events"
	"transport-app/internal/founder/alerts"
	"transport-app/internal/founder/customer_health"
	"transport-app/internal/founder/digest"
)

type FounderService struct {
	notifier Notifier
}

type Notifier interface {
	SendAlert(event alerts.AlertEvent) error
}

func NewFounderService(notifier Notifier) *FounderService {
	return &FounderService{
		notifier: notifier,
	}
}

// RegisterEventHandlers subscribes the founder alert service to relevant domain events on the event bus
func (s *FounderService) RegisterEventHandlers(bus events.EventBus) {
	if bus == nil {
		return
	}

	// 1. Paid Customer / Revenue
	bus.Subscribe("customer.activated", func(ctx context.Context, e events.Event) error {
		payload, _ := e.Payload.(map[string]interface{})
		companyName, _ := payload["company_name"].(string)
		plan, _ := payload["plan"].(string)
		mrr, _ := payload["mrr"].(string)

		return s.notifier.SendAlert(alerts.AlertEvent{
			ID:       fmt.Sprintf("rev_%d", time.Now().UnixNano()),
			Category: alerts.CategoryRevenue,
			Priority: alerts.PriorityHigh,
			Title:    "New Business Customer",
			Metadata: map[string]interface{}{
				"company": companyName,
				"plan":    plan,
				"mrr":     mrr,
			},
			Timestamp: time.Now(),
		})
	})

	// 2. Critical System Outage
	bus.Subscribe("system.critical_failure", func(ctx context.Context, e events.Event) error {
		payload, _ := e.Payload.(map[string]interface{})
		title, _ := payload["title"].(string)
		summary, _ := payload["summary"].(string)

		return s.notifier.SendAlert(alerts.AlertEvent{
			ID:        fmt.Sprintf("sys_%d", time.Now().UnixNano()),
			Category:  alerts.CategorySystem,
			Priority:  alerts.PriorityCritical,
			Title:     title,
			Summary:   summary,
			Timestamp: time.Now(),
		})
	})

	// 3. Customer Activation
	bus.Subscribe("customer.onboarding_completed", func(ctx context.Context, e events.Event) error {
		payload, _ := e.Payload.(map[string]interface{})
		companyName, _ := payload["company_name"].(string)
		activationTime, _ := payload["activation_time"].(string)

		return s.notifier.SendAlert(alerts.AlertEvent{
			ID:       fmt.Sprintf("act_%d", time.Now().UnixNano()),
			Category: alerts.CategoryActivation,
			Priority: alerts.PriorityMedium,
			Title:    "New Activated Customer",
			Metadata: map[string]interface{}{
				"company":         companyName,
				"activation_time": activationTime,
			},
			Timestamp: time.Now(),
		})
	})
}

// EvaluateCustomerHealth calculates health score and sends a churn risk alert if critical
func (s *FounderService) EvaluateCustomerHealth(companyID, companyName string, factors customer_health.CustomerHealthFactors) customer_health.CustomerHealthResult {
	result := customer_health.CalculateHealthScore(companyID, companyName, factors)

	if result.Score < 40 && s.notifier != nil {
		reasonStr := ""
		if len(result.Reasons) > 0 {
			reasonStr = result.Reasons[0]
		}
		_ = s.notifier.SendAlert(alerts.AlertEvent{
			ID:       fmt.Sprintf("churn_%s_%d", companyID, time.Now().Unix()),
			Category: alerts.CategoryChurnRisk,
			Priority: alerts.PriorityHigh,
			Title:    "Churn Risk",
			Metadata: map[string]interface{}{
				"company": companyName,
				"score":   fmt.Sprintf("%d", result.Score),
				"reason":  reasonStr,
				"action":  result.SuggestedAction,
			},
			Timestamp: time.Now(),
		})
	}

	return result
}

// SendDailyDigest compiles and dispatches executive daily report to Telegram
func (s *FounderService) SendDailyDigest(report digest.DailyDigestReport) error {
	if s.notifier == nil {
		return nil
	}
	msg := digest.FormatDailyDigest(report)
	return s.notifier.SendAlert(alerts.AlertEvent{
		ID:        fmt.Sprintf("digest_%d", time.Now().Unix()),
		Category:  alerts.CategoryProductUsage,
		Priority:  alerts.PriorityLow,
		Title:     "Daily Report",
		Summary:   msg,
		Timestamp: time.Now(),
	})
}
