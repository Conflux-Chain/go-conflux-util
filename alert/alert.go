package alert

import (
	viperutil "github.com/Conflux-Chain/go-conflux-util/viper"
)

// AlertConfig alert configuration such as DingTalk settings.,
type AlertConfig struct {
	// Custom tags are usually used to differentiate between different networks and enviroments
	// such as mainnet/testnet, prod/test/dev or any custom info for more details.
	CustomTags []string `default:"[dev]" mapstructure:"customTags"`
	// DingTalk settings
	DingTalk *DingTalkConfig `mapstructure:"dingtalk"`
}

// DingTalkConfig DingTalk configurations
type DingTalkConfig struct {
	// switch to turn on or off DingTalk
	Enabled bool `default:"false" mapstructure:"enabled"`
	// mobiles for @ members
	AtMobiles []string `mapstructure:"atMobiles"`
	// whether to @ all members
	IsAtAll bool `default:"false" mapstructure:"isAtAll"`
	// webhook url
	Webhook string `mapstructure:"webhook"`
	// secret token
	Secret string `mapstructure:"secret"`
}

// MustInitFromViper inits alert from viper settings or panic on error.
func MustInitFromViper() {
	var config AlertConfig
	viperutil.MustUnmarshalKey("alert", &config)

	Init(&config)
}

// Init inits alert with provided configurations.
func Init(config *AlertConfig) {
	if config.DingTalk.Enabled {
		InitDingTalk(config.DingTalk, config.CustomTags)
	}
}
