package alert

import (
	"context"

	"github.com/Conflux-Chain/go-conflux-util/alert/dingtalk"
	"github.com/pkg/errors"
)

var (
	_ Channel = (*DingTalkChannel)(nil)
)

type DingTalkConfig struct {
	AtMobiles []string // mobiles for @ members
	IsAtAll   bool     // whether to @ all members
	Webhook   string   // webhook url
	Secret    string   // secret token
}

// DingTalkChannel DingTalk notification channel
type DingTalkChannel struct {
	Formatter Formatter      // message formatter
	ID        string         // channel id
	Config    DingTalkConfig // channel config

	bot *dingtalk.Robot
}

func NewDingTalkChannel(chID string, fmt Formatter, conf DingTalkConfig) *DingTalkChannel {
	return &DingTalkChannel{
		ID: chID, Formatter: fmt, Config: conf,
		bot: dingtalk.NewRobot(conf.Webhook, conf.Secret),
	}
}

func (dtc *DingTalkChannel) Name() string {
	return dtc.ID
}

func (dtc *DingTalkChannel) Type() ChannelType {
	return ChannelTypeDingTalk
}

func (dtc *DingTalkChannel) Send(ctx context.Context, note *Notification) error {
	msg, err := dtc.Formatter.Format(note)
	if err != nil {
		return errors.WithMessage(err, "failed to format alert msg")
	}

	return dtc.bot.SendMarkdown(ctx, note.Title, msg, dtc.Config.AtMobiles, dtc.Config.IsAtAll)
}

// SendRaw sends raw message using the DingTalk channel.
func (dtc *DingTalkChannel) SendRaw(ctx context.Context, content interface{}) error {
	switch v := content.(type) {
	case string:
		return dtc.bot.SendText(ctx, v, dtc.Config.AtMobiles, dtc.Config.IsAtAll)
	case *dingtalk.TextMessage, *dingtalk.MarkdownMessage:
		return dtc.bot.Send(ctx, v)
	default:
		return ErrInvalidContentType
	}
}
