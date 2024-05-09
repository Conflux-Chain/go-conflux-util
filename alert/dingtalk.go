package alert

import (
	"github.com/pkg/errors"
	"github.com/royeo/dingrobot"
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
}

func NewDingTalkChannel(chID string, fmt Formatter, conf DingTalkConfig) *DingTalkChannel {
	return &DingTalkChannel{ID: chID, Formatter: fmt, Config: conf}
}

func (dtc *DingTalkChannel) Name() string {
	return dtc.ID
}

func (dtc *DingTalkChannel) Type() ChannelType {
	return ChannelTypeDingTalk
}

func (dtc *DingTalkChannel) Send(note *Notification) error {
	msg, err := dtc.Formatter.Format(note)
	if err != nil {
		return errors.WithMessage(err, "failed to format alert msg")
	}

	dingRobot := dingrobot.NewRobot(dtc.Config.Webhook)
	dingRobot.SetSecret(dtc.Config.Secret)
	return dingRobot.SendMarkdown(note.Title, msg, dtc.Config.AtMobiles, dtc.Config.IsAtAll)
}
