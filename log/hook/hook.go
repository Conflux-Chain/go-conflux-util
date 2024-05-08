package hook

import (
	"context"
	stderr "errors"

	"github.com/Conflux-Chain/go-conflux-util/alert"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// logrus entry field configured for alert channels
	chLogEntryField = "@channel"

	// alert message title
	alertMsgTitle = "logrus alert notification"
)

type Config struct {
	// logrus level hooked for alert notification
	Level string `default:"warn"`
	// default alert channels
	Channels []string
	// async worker options
	Async AsyncOption
}

// AddAlertHook adds logrus hook for alert notification with specified log levels.
func AddAlertHook(conf Config) error {
	if len(conf.Channels) == 0 {
		return nil
	}

	var chs []alert.Channel
	for _, chn := range conf.Channels {
		ch, ok := alert.DefaultManager().Channel(chn)
		if !ok {
			return alert.ErrChannelNotFound(chn)
		}
		chs = append(chs, ch)
	}

	// hook alert logging levels
	lvl, err := logrus.ParseLevel(conf.Level)
	if err != nil {
		return errors.WithMessage(err, "failed to parse log level")
	}

	var hookLvls []logrus.Level
	for l := logrus.PanicLevel; l <= lvl; l++ {
		hookLvls = append(hookLvls, l)
	}

	var alertHook logrus.Hook = NewAlertHook(hookLvls, chs)
	if conf.Async.Enabled { // use async mode
		alertHook = NewAsyncHook(context.Background(), alertHook, conf.Async)
	}

	logrus.AddHook(alertHook)
	return nil
}

// AlertHook logrus hooks to send specified level logs as text message for alerting.
type AlertHook struct {
	levels          []logrus.Level
	defaultChannels []alert.Channel
}

// NewAlertHook constructor to new AlertHook instance.
func NewAlertHook(lvls []logrus.Level, chs []alert.Channel) *AlertHook {
	return &AlertHook{levels: lvls, defaultChannels: chs}
}

// implements `logrus.Hook` interface methods.
func (hook *AlertHook) Levels() []logrus.Level {
	return hook.levels
}

func (hook *AlertHook) Fire(logEntry *logrus.Entry) (err error) {
	notifyChans, err := hook.getAlertChannels(logEntry)
	if err != nil || len(notifyChans) == 0 {
		return err
	}

	note := &alert.Notification{
		Title:   alertMsgTitle,
		Content: logEntry,
	}

	for _, ch := range notifyChans {
		err = stderr.Join(ch.Send(note))
	}

	return errors.WithMessage(err, "failed to notify channel message")
}

func (hook *AlertHook) getAlertChannels(logEntry *logrus.Entry) (chs []alert.Channel, err error) {
	v, ok := logEntry.Data[chLogEntryField]
	if !ok { // notify channel not configured, use default
		return hook.defaultChannels, nil
	}

	var chns []string
	switch chv := v.(type) {
	case string:
		chns = append(chns, chv)
	case []string:
		chms := make(map[string]struct{})
		for _, chn := range chv {
			if _, ok := chms[chn]; !ok { // dedupe
				chms[chn] = struct{}{}
				chns = append(chns, chn)
			}
		}
	case alert.Channel:
		chs = append(chs, chv)
		return
	case []alert.Channel:
		chs = append(chs, chv...)
		return
	default:
		return nil, errors.New("invalid log entry configured for alert channel")
	}

	// parse notify channel from channel name
	for _, chn := range chns {
		ch, ok := alert.DefaultManager().Channel(chn)
		if !ok {
			return nil, alert.ErrChannelNotFound(chn)
		}

		chs = append(chs, ch)
	}

	return chs, nil
}
