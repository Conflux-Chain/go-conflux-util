package alert

import (
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
	ChannelTypeDingTalk ChannelType = "dingtalk"
	ChannelTypeTelegram ChannelType = "telegram"
)

// Notification channel interface.
type Channel interface {
	Name() string
	Type() ChannelType
	Send(note *Notification) error
}

// Notification represents core information for an alert.
type Notification struct {
	Title    string   // message title
	Content  string   // message content
	Severity Severity // severity level
}

// MustInitFromViper inits alert from viper settings or panic on error.
func MustInitFromViper() {
	var conf struct {
		CustomTags []string `default:"[dev,test]"`
		Channels   map[string]interface{}
	}

	viperutil.MustUnmarshalKey("alert", &conf)

	formatter := NewSimpleTextFormatter(conf.CustomTags)
	for chID, chmap := range conf.Channels {
		ch, err := parseAlertChannel(chID, chmap.(map[string]interface{}), formatter)
		if err != nil {
			logrus.WithField("channelId", chID).Fatal("Failed to parse alert channel")
		}

		DefaultManager().Add(ch)
	}
}
