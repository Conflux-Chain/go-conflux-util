package alert

import (
	"context"

	viperutil "github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/sirupsen/logrus"
)

// Alert severity level.
type Severity int

const (
	SeverityLow Severity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

func (s Severity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Notification channel type.
type ChannelType string

const (
	ChannelTypeDingTalk  ChannelType = "dingtalk"
	ChannelTypeTelegram  ChannelType = "telegram"
	ChannelTypeSMTP      ChannelType = "smtp"
	ChannelTypePagerDuty ChannelType = "pagerduty"
)

// Notification channel interface.
type Channel interface {
	Name() string
	Type() ChannelType
	Send(context.Context, *Notification) error
}

// Notification represents core information for an alert.
type Notification struct {
	Title    string      // message title
	Content  interface{} // message content
	Severity Severity    // severity level
}

// MustInitFromViper inits alert from viper settings or panic on error.
func MustInitFromViper() {
	var conf struct {
		CustomTags []string `default:"[dev]"`
		Channels   map[string]interface{}
	}

	viperutil.MustUnmarshalKey("alert", &conf)

	for chID, chmap := range conf.Channels {
		ch, err := parseAlertChannel(chID, chmap.(map[string]interface{}), conf.CustomTags)
		if err != nil {
			logrus.WithField("channelId", chID).WithError(err).Fatal("Failed to parse alert channel")
		}

		DefaultManager().Add(ch)
	}
}
