package hook

import (
	"os"
	"testing"

	"github.com/Conflux-Chain/go-conflux-util/alert"
	"github.com/sirupsen/logrus"
)

// Please set the following enviroments before running tests:
// `TEST_DINGTALK_WEBHOOK`: DingTalk webhook;
// `TEST_DINGTALK_SECRET`: DingTalk secret.

func TestMain(m *testing.M) {
	webhook := os.Getenv("TEST_DINGTALK_WEBHOOK")
	secret := os.Getenv("TEST_DINGTALK_SECRET")

	if len(webhook) == 0 || len(secret) == 0 {
		return
	}

	alert.InitDingTalk(&alert.DingTalkConfig{
		Enabled: true,
		Webhook: webhook,
		Secret:  secret,
	}, []string{"log", "hook", "test"})

	os.Exit(m.Run())
}

func TestLogrusAddHooks(t *testing.T) {
	// Add alert hook for logrus fatal/warn/error level
	hookLevels := []logrus.Level{logrus.FatalLevel, logrus.WarnLevel, logrus.ErrorLevel}
	dingTalkAlertHook := NewDingTalkAlertHook(hookLevels)
	logrus.AddHook(dingTalkAlertHook)

	// Need to manually check if message sent to dingtalk group chat
	logrus.Warn("Test logrus add hooks warns")
	logrus.Error("Test logrus add hooks error")
	logrus.Fatal("Test logrus add hooks fatal")
}
