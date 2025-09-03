package alert

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/Conflux-Chain/go-conflux-util/alert/flashduty"
	"github.com/pkg/errors"
)

var (
	_ Channel = (*FlashDutyChannel)(nil)
)

type FlashDutyConfig struct {
	Webhook string // webhook url
	Secret  string // secret token
}

// FlashDutyChannel represents a FlashDuty notification channel.
type FlashDutyChannel struct {
	*flashduty.Robot
	ID        string          // the identifier of the channel
	Config    FlashDutyConfig // the configuration for the FlashDuty channel
	Formatter Formatter       // message formatter
}

// NewFlashDutyChannel creates a new FlashDuty channel with the given ID and configuration
func NewFlashDutyChannel(chID string, fmt Formatter, conf FlashDutyConfig) *FlashDutyChannel {
	return &FlashDutyChannel{
		ID:        chID,
		Config:    conf,
		Formatter: fmt,
		Robot:     flashduty.NewRobot(conf.Webhook, conf.Secret),
	}
}

// Name returns the ID of the channel
func (c *FlashDutyChannel) Name() string {
	return c.ID
}

// Type returns the type of the channel, which is FlashDuty
func (c *FlashDutyChannel) Type() ChannelType {
	return ChannelTypeFlashDuty
}

// Send sends notification using the FlashDuty channel.
func (c *FlashDutyChannel) Send(ctx context.Context, note *Notification) error {
	jsonMsgText, err := c.Formatter.Format(note)
	if err != nil {
		return errors.WithMessage(err, "failed to format alert msg from notification")
	}

	var raw map[string]interface{}
	err = json.Unmarshal([]byte(jsonMsgText), &raw)
	if err != nil {
		return errors.WithMessage(err, "failed to format alert msg from json")
	}

	m := make(map[string]string)
	for k, v := range raw {
		switch vv := v.(type) {
		case string:
			m[k] = vv
		case float64:
			if vv == float64(int64(vv)) {
				m[k] = strconv.FormatInt(int64(vv), 10)
			} else {
				m[k] = strconv.FormatFloat(vv, 'f', -1, 64)
			}
		case bool:
			m[k] = strconv.FormatBool(vv)
		default:
			b, _ := json.Marshal(vv)
			m[k] = string(b)
		}
	}

	return c.Robot.Send(ctx, note.Title, c.adaptSeverity(note.Severity), "", "", m)
}

// adaptSeverity adapts notification severity level to FlashDuty severity level.
func (c *FlashDutyChannel) adaptSeverity(severity Severity) string {
	switch severity {
	case SeverityMedium:
		return flashduty.MsgLevelWarning
	case SeverityCritical:
		return flashduty.MsgLevelCritical
	default:
		return flashduty.MsgLevelInfo
	}
}
