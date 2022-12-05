package log

import (
	"strings"

	"github.com/Conflux-Chain/go-conflux-util/alert"
	"github.com/sirupsen/logrus"
)

// AddDingTalkAlertHook adds logrus hook for DingTalk alert with specified log levels.
func AddDingTalkAlertHook(hookLevels []logrus.Level) {
	dingTalkAlertHook := NewDingTalkAlertHook(hookLevels)
	logrus.AddHook(dingTalkAlertHook)
}

// DingTalkAlertHook logrus hooks to send specified level logs as
// text message to DingTalk group chat.
type DingTalkAlertHook struct {
	levels []logrus.Level
}

// NewDingTalkAlertHook constructor to new DingTalkAlertHook instance.
func NewDingTalkAlertHook(lvls []logrus.Level) *DingTalkAlertHook {
	return &DingTalkAlertHook{levels: lvls}
}

// Levels implements logrus.Hook interface `Levels` method.
func (hook *DingTalkAlertHook) Levels() []logrus.Level {
	return hook.levels
}

// Fire implements logrus.Hook interface `Fire` method.
func (hook *DingTalkAlertHook) Fire(logEntry *logrus.Entry) error {
	level := logEntry.Level.String()
	brief := logEntry.Message

	formatter := &logrus.JSONFormatter{}
	detailBytes, _ := formatter.Format(logEntry)
	// Trim last newline char to uniform message format
	detail := strings.TrimSuffix(string(detailBytes), "\n")

	return alert.SendDingTalkTextMessage(level, brief, detail)
}
