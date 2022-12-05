package alert

import (
	"fmt"

	viperutil "github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// AlertConfig alert configuration such as DingTalk settings etc.
type AlertConfig struct {
	// Custom tags are usually used to differentiate between different networks and enviroments
	// such as mainnet/testnet, prod/test/dev or any custom info for more details.
	CustomTags []string `default:"[dev,test]"`

	// DingTalk settings
	DingTalk DingTalkConfig
}

// DingTalkConfig DingTalk configurations
type DingTalkConfig struct {
	Enabled   bool     // switch to turn on or off DingTalk
	AtMobiles []string // mobiles for @ members
	IsAtAll   bool     // whether to @ all members
	Webhook   string   // webhook url
	Secret    string   // secret token
}

// MustInitFromViper inits alert from viper settings or panic on error.
func MustInitFromViper() {
	var config AlertConfig
	viperutil.MustUnmarshalKey("alert", &config, func(key string) (interface{}, bool) {
		switch key {
		case "alert.customTags", "alert.dingtalk.atMobiles":
			return viper.GetStringSlice(key), true
		}

		return nil, false
	})

	Init(config)
}

// Init inits alert with provided configurations.
func Init(config AlertConfig) {
	if config.DingTalk.Enabled {
		InitDingTalk(&config.DingTalk, config.CustomTags)
		logrus.WithField("config", fmt.Sprintf("%+v", config)).Debug("Alert (dingtalk) initialized")
	}
}
