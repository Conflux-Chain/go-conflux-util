package log

import (
	"testing"

	"github.com/Conflux-Chain/go-conflux-util/alert"
	"github.com/sirupsen/logrus"
)

func initTest() {
	alert.InitDingTalk(&alert.DingTalkConfig{
		Enabled: true,
		Webhook: "http://test.webhook",
		Secret:  "test.secret",
	}, []string{"log", "hook", "test"})
}

func TestLogrusAddHooks(t *testing.T) {
	initTest()

	// Add alert hook for logrus fatal/warn/error level
	hookLevels := []logrus.Level{logrus.FatalLevel, logrus.WarnLevel, logrus.ErrorLevel}
	dingTalkAlertHook := NewDingTalkAlertHook(hookLevels)
	logrus.AddHook(dingTalkAlertHook)

	// Need to manually check if message sent to dingtalk group chat
	logrus.Warn("Test logrus add hooks warns")
	logrus.Error("Test logrus add hooks error")
	logrus.Fatal("Test logrus add hooks fatal")
}
