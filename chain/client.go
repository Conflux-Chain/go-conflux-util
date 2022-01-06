package chain

import (
	"time"

	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	viperutil "github.com/Conflux-Chain/go-conflux-util/viper"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// CfxClientConfig Conflux sdk client configurations
type CfxClientConfig struct {
	// websocket endpoint
	WebsocketUrl string `default:"" mapstructure:"ws"`
	// http endpoint
	HttpUrl string `default:"" mapstructure:"http"`
	// retry counts if request failed
	Retry int `default:"3" mapstructure:"retry"`
	// retry interval if request failed
	RetryInterval time.Duration `default:"1s" mapstructure:"retryInterval"`
	// request timeout
	RequestTimeout time.Duration `default:"3s" mapstructure:"requestTimeout"`
}

// MustNewCfxClientFromViper creates an instance of Conflux client
// from viper or panic on error.
func MustNewCfxClientFromViper(isWebSocket ...bool) *sdk.Client {
	var option CfxClientConfig
	viperutil.MustUnmarshalKey("cfx", &option)

	endpoint := option.HttpUrl
	if len(isWebSocket) > 0 && isWebSocket[0] {
		endpoint = option.WebsocketUrl
	}

	client, err := sdk.NewClient(
		endpoint, sdkClientOptionFromConfig(&option),
	)

	if err != nil {
		logrus.WithField("option", option).
			WithError(err).
			Fatal("Failed to create CFX client")
	}

	return client
}

// GetCfxClientOptionFromViper gets Conflux client option from viper.
//
// Note that viper must be initialized before calling this function,
// otherwise settings might not be loaded correctly.
func GetCfxClientOptionFromViper() (sdk.ClientOption, error) {
	var option CfxClientConfig

	err := viperutil.UnmarshalKey("cfx", &option)
	if err != nil {
		return sdk.ClientOption{}, errors.WithMessage(err, "failed to unmarshal Conflux client config")
	}

	return sdkClientOptionFromConfig(&option), nil
}

// sdkClientOptionFromConfig generates sdk client option from config.
func sdkClientOptionFromConfig(config *CfxClientConfig) sdk.ClientOption {
	return sdk.ClientOption{
		RetryCount:     config.Retry,
		RetryInterval:  config.RetryInterval,
		RequestTimeout: config.RequestTimeout,
	}
}
