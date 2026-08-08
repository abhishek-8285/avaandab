package alerts

import (
	"fmt"

	"gopkg.in/telebot.v3"
)

type Formatter interface {
	Format(event AlertEvent) string
}

type TelegramFormatter struct{}

func NewTelegramFormatter() *TelegramFormatter {
	return &TelegramFormatter{}
}

func (f *TelegramFormatter) Format(event AlertEvent) string {
	switch event.Category {
	case CategoryRevenue:
		return formatRevenue(event)
	case CategoryChurnRisk:
		return formatChurnRisk(event)
	case CategoryActivation:
		return formatActivation(event)
	case CategorySystem:
		return formatSystem(event)
	default:
		return formatGeneric(event)
	}
}

func formatRevenue(e AlertEvent) string {
	company, _ := e.Metadata["company"].(string)
	plan, _ := e.Metadata["plan"].(string)
	mrr, _ := e.Metadata["mrr"].(string)

	return fmt.Sprintf("💰 *%s*\n\n*Company:*\n%s\n\n*Plan:*\n%s\n\n*MRR:*\n%s", e.Title, company, plan, mrr)
}

func formatChurnRisk(e AlertEvent) string {
	company, _ := e.Metadata["company"].(string)
	score, _ := e.Metadata["score"].(string)
	reason, _ := e.Metadata["reason"].(string)
	action, _ := e.Metadata["action"].(string)

	return fmt.Sprintf("⚠️ *%s*\n\n*Company:*\n%s\n\n*Health Score:*\n%s%%\n\n*Reason:*\n%s\n\n*Suggested Action:*\n%s", e.Title, company, score, reason, action)
}

func formatActivation(e AlertEvent) string {
	company, _ := e.Metadata["company"].(string)
	timeTaken, _ := e.Metadata["activation_time"].(string)

	return fmt.Sprintf("🎉 *%s*\n\n*Company:*\n%s\n\n✓ Company Profile\n✓ Vehicle\n✓ Driver\n✓ Customer\n✓ Booking\n✓ Trip\n\n*Activation Time:*\n%s", e.Title, company, timeTaken)
}

func formatSystem(e AlertEvent) string {
	return fmt.Sprintf("🚨 *CRITICAL SYSTEM ALERT*\n\n*%s*\n%s", e.Title, e.Summary)
}

func formatGeneric(e AlertEvent) string {
	return fmt.Sprintf("🔔 *%s*\n\n%s", e.Title, e.Summary)
}

// TelegramBotNotifier sends alerts directly via telebot.v3
type TelegramBotNotifier struct {
	bot    *telebot.Bot
	chatID int64
	fmt    Formatter
}

func NewTelegramBotNotifier(bot *telebot.Bot, chatID int64) *TelegramBotNotifier {
	return &TelegramBotNotifier{
		bot:    bot,
		chatID: chatID,
		fmt:    NewTelegramFormatter(),
	}
}

func (n *TelegramBotNotifier) SendAlert(event AlertEvent) error {
	if n.bot == nil || n.chatID == 0 {
		return nil // Graceful fallback if Telegram integration is disabled
	}
	msg := n.fmt.Format(event)
	_, err := n.bot.Send(telebot.ChatID(n.chatID), msg, telebot.ModeMarkdown)
	return err
}
