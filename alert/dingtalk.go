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
	MsgType   string   `default:"markdown"` // message type: `text` or `markdown`
}

// DingTalkChannel DingTalk notification channel
type DingTalkChannel struct {
	*dingtalk.Robot
	ID        string         // channel id
	Config    DingTalkConfig // channel config
	Formatter Formatter      // message formatter
}

func NewDingTalkChannel(chID string, fmt Formatter, conf DingTalkConfig) *DingTalkChannel {
	return &DingTalkChannel{
		ID: chID, Formatter: fmt, Config: conf,
		Robot: dingtalk.NewRobot(conf.Webhook, conf.Secret),
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

	return dtc.Robot.Send(ctx, dtc.Config.MsgType, note.Title, msg, dtc.Config.AtMobiles, dtc.Config.IsAtAll)
}
