package log

import (
	"os"
	"strings"
	"testing"

	"github.com/Conflux-Chain/go-conflux-util/alert"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func initTest() {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("cfx")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	os.Setenv("CFX_ALERT_CUSTOMTAGS", "log hook test")
	os.Setenv("CFX_ALERT_DINGTALK_ENABLED", "true")
	os.Setenv("CFX_ALERT_DINGTALK_ATMOBILES", "")
	os.Setenv("CFX_ALERT_DINGTALK_ISATALL", "false")
	os.Setenv("CFX_ALERT_DINGTALK_WEBHOOK", "")
	os.Setenv("CFX_ALERT_DINGTALK_SECRET", "")

	alert.InitDingRobotFromViper()
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
