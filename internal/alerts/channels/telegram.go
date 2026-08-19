package channels

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"gopkg.in/telebot.v3"
	"transport-app/internal/config"
)

// TelegramProvider delivers critical/blocker operational alerts to Telegram.
type TelegramProvider struct {
	bot    *telebot.Bot
	chatID int64
	logger *slog.Logger
}

// NewTelegramProvider initializes a Telegram bot client from config.
// Returns a safe no-op instance if token or chat_id is missing or invalid.
func NewTelegramProvider(cfg *config.Config, logger *slog.Logger) *TelegramProvider {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg == nil {
		return &TelegramProvider{logger: logger}
	}

	token := cfg.Alerts.TelegramBotToken
	chatIDStr := cfg.Alerts.TelegramChatID

	if token == "" || chatIDStr == "" {
		logger.Debug("telegram alerts provider disabled: token or chat_id empty")
		return &TelegramProvider{logger: logger}
	}

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		logger.Warn("telegram alerts provider disabled: invalid chat_id", "chat_id", chatIDStr, "error", err)
		return &TelegramProvider{logger: logger}
	}

	bot, err := telebot.NewBot(telebot.Settings{Token: token})
	if err != nil {
		logger.Warn("telegram alerts provider disabled: failed to init bot", "error", err)
		return &TelegramProvider{logger: logger}
	}

	logger.Info("telegram alerts provider enabled", "chat_id", chatID)
	return &TelegramProvider{
		bot:    bot,
		chatID: chatID,
		logger: logger,
	}
}

func (p *TelegramProvider) Name() string { return "telegram" }

func (p *TelegramProvider) Send(ctx context.Context, msg Message) error {
	if p.bot == nil || p.chatID == 0 {
		p.logger.Debug("telegram alert skipped (bot not configured)", "title", msg.Title, "severity", msg.Severity)
		return nil
	}

	entityType := ""
	entityID := ""
	if msg.Meta != nil {
		if et, ok := msg.Meta["entity_type"].(string); ok {
			entityType = et
		}
		if eid, ok := msg.Meta["entity_id"].(string); ok {
			entityID = eid
		}
	}

	text := fmt.Sprintf("🚨 *%s Alert: %s*\n%s", msg.Severity, msg.Title, msg.Body)
	if entityType != "" || entityID != "" {
		text += fmt.Sprintf("\n_Entity: %s %s_", entityType, entityID)
	}

	recipient := &telebot.Chat{ID: p.chatID}
	_, err := p.bot.Send(recipient, text, telebot.ModeMarkdown)
	if err != nil {
		p.logger.Error("failed to send telegram alert", "error", err, "alert_id", msg.AlertID)
		return err
	}

	return nil
}
