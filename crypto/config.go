package crypto

import (
	"github.com/Conflux-Chain/go-conflux-util/viper"
)

var (
	cryptoConf CryptoConfig // crypto config
)

// init inits crypto settings.
//
// Note that viper must be initialized to load settings
// correctly from viper.
func init() {
	viper.MustUnmarshalKey("crypto", &cryptoConf)
}

// CryptoConfig crypto configurations for private key provider etc.,
type CryptoConfig struct {
	// private key custodian configurations
	PrivateKeyProvider PrivateKeyProviderConfig
}

// PrivateKeyProviderConfig private key provider configurations for
// kinds of private key providers such as HMAC, MPC etc.,
type PrivateKeyProviderConfig struct {
	// HMAC private key provider configurations
	HMAC HmacPrivateKeyProviderConfig
}

// HmacPrivateKeyProviderConfig configurations to generate private key
// on the fly with HMAC crypto algorithm.
type HmacPrivateKeyProviderConfig struct {
	// file path where HMAC secret stored
	SecretFile string `default:"./config/hmac_secret"`
	// whether to wipe out secret file on initial load in case of secret leak out
	WipeOutSecret bool `default:"true"`
	// keystore directory where private key will be loaded and generated
	KeystoreDir string `default:"./keystore"`
	// keystore partitions for key distributions
	KeystorePartitions uint32 `default:"100"`
	// LRU cache capacity
	LRUCapacity int `default:"5000"`
}
