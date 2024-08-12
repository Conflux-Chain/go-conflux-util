package alert

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/pkg/errors"
)

var (
	_ Channel = (*TelegramChannel)(nil)
)

type TelegramConfig struct {
	ApiToken string   // Api token
	ChatId   string   // Chat ID
	AtUsers  []string // Mention users
}

// TelegramChannel Telegram notification channel
type TelegramChannel struct {
	*bot.Bot
	Formatter Formatter      // message formatter
	ID        string         // channel id
	Config    TelegramConfig // channel config
}

func NewTelegramChannel(chID string, fmt Formatter, conf TelegramConfig) (*TelegramChannel, error) {
	bot, err := bot.New(conf.ApiToken, bot.WithSkipGetMe())
	if err != nil {
		return nil, err
	}

	tc := &TelegramChannel{ID: chID, Formatter: fmt, Config: conf, Bot: bot}
	return tc, nil
}

func (tc *TelegramChannel) Name() string {
	return tc.ID
}

func (tc *TelegramChannel) Type() ChannelType {
	return ChannelTypeTelegram
}

func (tc *TelegramChannel) Send(ctx context.Context, note *Notification) error {
	msg, err := tc.Formatter.Format(note)
	if err != nil {
		return errors.WithMessage(err, "failed to format alert msg")
	}

	_, err = tc.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    tc.Config.ChatId,
		Text:      msg,
		ParseMode: models.ParseModeMarkdown,
	})

	return err
}
