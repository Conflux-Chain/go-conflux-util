package alert

import (
	"fmt"
	"strings"
	"time"

	"github.com/royeo/dingrobot"
	"github.com/spf13/viper"
)

const (
	// DingTalk alert message template
	dingTalkAlertMsgTpl = "logrus alert notification\ntags:\t%v;\nlevel:\t%v;\nbrief:\t%v;\ndetail:\t%v;\ntime:\t%v\n"
)

var (
	// Custom tags are usually used to differentiate between different networks and enviroments
	// such as mainnet/testnet, prod/test/dev or any custom info for more details.
	dingTalkCustomTags    []string
	dingTalkCustomTagsStr string

	dingTalkAtMobiles []string // mobiles for @ members
	dingTalkIsAtAll   bool     // whether to @ all members

	dingRobot dingrobot.Roboter
)

func init() {
	// only init DingTalk robot from viper if enabled
	if viper.GetBool("alert.dingtalk.enabled") {
		InitDingRobotFromViper()
	}
}

// InitDingRobotFromViper inits DingTalk robot from Viper
func InitDingRobotFromViper() {
	dingTalkCustomTags = viper.GetStringSlice("alert.customTags")
	dingTalkCustomTagsStr = strings.Join(dingTalkCustomTags, "/")

	dingTalkAtMobiles = viper.GetStringSlice("alert.dingtalk.atMobiles")
	dingTalkIsAtAll = viper.GetBool("alert.dingtalk.isAtAll")

	// webhook and secrets
	webHook := viper.GetString("alert.dingtalk.webhook")
	secret := viper.GetString("alert.dingtalk.secret")

	dingRobot = dingrobot.NewRobot(webHook)
	dingRobot.SetSecret(secret)
}

// SendDingTalkTextMessage sends text message to DingTalk group chat.
func SendDingTalkTextMessage(level, brief, detail string) error {
	if dingRobot == nil {
		return nil
	}

	nowStr := time.Now().Format("2006-01-02T15:04:05-0700")
	msg := fmt.Sprintf(dingTalkAlertMsgTpl, dingTalkCustomTagsStr, level, brief, detail, nowStr)

	return dingRobot.SendText(msg, dingTalkAtMobiles, dingTalkIsAtAll)
}
