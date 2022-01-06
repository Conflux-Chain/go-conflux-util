package alert

import (
	"fmt"
	"strings"
	"time"

	"github.com/royeo/dingrobot"
)

const (
	// DingTalk alert message template
	dingTalkAlertMsgTpl = "logrus alert notification\ntags:\t%v;\nlevel:\t%v;\nbrief:\t%v;\ndetail:\t%v;\ntime:\t%v\n"
)

var (
	dingTalkCustomTagsStr string
	dingTalkConfig        *DingTalkConfig
	dingRobot             dingrobot.Roboter
)

// InitDingTalk inits DingTalk with provided configurations.
func InitDingTalk(config *DingTalkConfig, customTags []string) {
	if !config.Enabled {
		return
	}

	dingTalkCustomTagsStr = strings.Join(customTags, "/")
	dingTalkConfig = config

	// init DingTalk robots
	dingRobot = dingrobot.NewRobot(config.Webhook)
	dingRobot.SetSecret(config.Secret)
}

// SendDingTalkTextMessage sends text message to DingTalk group chat.
func SendDingTalkTextMessage(level, brief, detail string) error {
	if dingRobot == nil { // robot not set
		return nil
	}

	nowStr := time.Now().Format("2006-01-02T15:04:05-0700")
	msg := fmt.Sprintf(dingTalkAlertMsgTpl, dingTalkCustomTagsStr, level, brief, detail, nowStr)

	return dingRobot.SendText(msg, dingTalkConfig.AtMobiles, dingTalkConfig.IsAtAll)
}
