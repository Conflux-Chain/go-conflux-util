package hook

import (
	"os"
	"testing"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/alert"
	"github.com/sirupsen/logrus"
)

var (
	channels []alert.Channel
)

// Please set the following enviroments before running tests:
// `TEST_DINGTALK_WEBHOOK`: DingTalk webhook;
// `TEST_DINGTALK_SECRET`: DingTalk secret;
// `TEST_TELEGRAM_API_TOKEN`: Telegram API token;
// `TEST_TELEGRAM_CHAT_ID`: Telegram chat ID.
func TestMain(m *testing.M) {
	fmtter := alert.NewSimpleTextFormatter([]string{"log", "hook", "test"})

	dtWebhook := os.Getenv("TEST_DINGTALK_WEBHOOK")
	dtSecret := os.Getenv("TEST_DINGTALK_SECRET")
	if len(dtWebhook) > 0 && len(dtSecret) > 0 {
		dingrobot := alert.NewDingTalkChannel("dingrobot", fmtter, alert.DingTalkConfig{
			Webhook: dtWebhook, Secret: dtSecret,
		})
		channels = append(channels, dingrobot)
	}

	tgApiToken := os.Getenv("TEST_TELEGRAM_API_TOKEN")
	tgChatId := os.Getenv("TEST_TELEGRAM_API_TOKEN")
	if len(tgApiToken) > 0 && len(tgChatId) > 0 {
		tgrobot, err := alert.NewTelegramChannel("tgrobot", fmtter, alert.TelegramConfig{
			ApiToken: tgApiToken, ChatId: tgChatId,
		})
		if err != nil {
			logrus.WithError(err).Fatal("Failed to new telegram channel")
		}

		channels = append(channels, tgrobot)
	}

	os.Exit(m.Run())
}

func TestLogrusAddHooks(t *testing.T) {
	if len(channels) == 0 {
		return
	}

	// Add alert hook for logrus fatal/warn/error level
	hookLevels := []logrus.Level{logrus.FatalLevel, logrus.WarnLevel, logrus.ErrorLevel}
	logrus.AddHook(NewAlertHook(hookLevels, channels, 3*time.Second))

	// Need to manually check if message sent to dingtalk group chat
	logrus.Warn("Test logrus add hooks warns")
	logrus.Error("Test logrus add hooks error")
	logrus.Fatal("Test logrus add hooks fatal")
}
