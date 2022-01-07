package chain

import (
	"time"

	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// CfxClientOptionConfig Conflux SDK client option configurations.
type CfxClientOptionConfig struct {
	// retry counts if request failed
	Retry int `default:"3"`
	// retry interval if request failed
	RetryInterval time.Duration `default:"1s"`
	// request timeout
	RequestTimeout time.Duration `default:"3s"`
}

// MustNewCfxClientFromViper creates an instance of Conflux client
// from viper or panic on error.
func MustNewCfxClientFromViper(isWebSocket ...bool) *sdk.Client {
	endpoint := viper.GetString("cfx.http")
	if len(isWebSocket) > 0 && isWebSocket[0] {
		endpoint = viper.GetString("cfx.ws")
	}

	client, err := NewCfxClientWithOptionFromViper(endpoint)
	if err != nil {
		logrus.WithField("endpoint", endpoint).
			WithError(err).
			Fatal("Failed to create CFX client")
	}

	return client
}

// NewCfxClientWithOptionFromViper creates an instance of Conflux client
// with option provided from viper.
func NewCfxClientWithOptionFromViper(endpoint string) (*sdk.Client, error) {
	var optConfig CfxClientOptionConfig

	if err := viper.UnmarshalKey("cfx", &optConfig); err != nil {
		err = errors.WithMessage(err, "failed to unmarshal config from viper")
		return nil, err
	}

	option := sdk.ClientOption{
		RetryCount:     optConfig.Retry,
		RetryInterval:  optConfig.RetryInterval,
		RequestTimeout: optConfig.RequestTimeout,
	}
	return sdk.NewClient(endpoint, option)
}
