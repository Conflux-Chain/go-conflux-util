package alert

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/Conflux-Chain/go-conflux-util/alert/flashduty"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
	ID     string          // the identifier of the channel
	Config FlashDutyConfig // the configuration for the FlashDuty channel
}

// NewFlashDutyChannel creates a new FlashDuty channel with the given ID and configuration
func NewFlashDutyChannel(chID string, conf FlashDutyConfig) *FlashDutyChannel {
	return &FlashDutyChannel{
		ID:     chID,
		Config: conf,
		Robot:  flashduty.NewRobot(conf.Webhook, conf.Secret),
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
	var data map[string]string
	var err error
	if _, ok := note.Content.(*logrus.Entry); ok {
		data, err = formatLogrusEntry(note)
	} else {
		data, err = formatDefault(note)
	}

	if err != nil {
		return errors.WithMessage(err, "failed to format notification content")
	}

	return c.Robot.Send(ctx, note.Title, c.adaptSeverity(note.Severity), "", "", data)
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

func formatLogrusEntry(note *Notification) (map[string]string, error) {
	entry := note.Content.(*logrus.Entry)
	entryError, _ := entry.Data[logrus.ErrorKey].(error)

	ctxFields := make(map[string]string)
	for k, v := range entry.Data {
		if k == logrus.ErrorKey {
			continue
		}

		vvStr, err := formatToString(v)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to format field to string")
		}
		ctxFields[k] = vvStr
	}

	ctxFields["level"] = entry.Level.String()
	ctxFields["time"] = entry.Time.String()
	ctxFields["msg"] = entry.Message
	if entryError != nil {
		ctxFields["error"] = entryError.Error()
	}

	return ctxFields, nil
}

func formatDefault(note *Notification) (map[string]string, error) {
	ctxFields := make(map[string]string)
	contentStr, err := formatToString(note.Content)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to format notification content to string")
	}

	ctxFields["content"] = contentStr

	return ctxFields, nil
}

func formatToString(v interface{}) (string, error) {
	switch vv := v.(type) {
	case string:
		return vv, nil
	case float64:
		if vv == float64(int64(vv)) {
			return strconv.FormatInt(int64(vv), 10), nil
		} else {
			return strconv.FormatFloat(vv, 'f', -1, 64), nil
		}
	case bool:
		return strconv.FormatBool(vv), nil
	default:
		b, err := json.Marshal(vv)
		if err != nil {
			return "", errors.WithMessagef(err, "failed to format field %s", vv)
		}
		return string(b), nil
	}
}
