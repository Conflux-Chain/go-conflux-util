package crypto

import (
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/sha256"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	gethks "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	hmacKeyStoreWriteBuffer = 2000 // key store write buffer size
)

// hmacKeyStoreWriteOp key store write operation
type hmacKeyStoreWriteOp struct {
	kid PrivKeyIdentifier
	key *gethks.Key
}

// HmacPrivKeyProvider generates private key on the fly with HMAC.
// The basic idea is to concatenate the key and the unique identifier, and hash them together
// to generate the private key. It also stores the generated keys into keystore directory
// on disk and caches them within in-memory LRU cache of specified capacity.
type HmacPrivKeyProvider struct {
	secretKey   []byte                    // secret key used for HMAC to generate private key
	cache       *lru.Cache                // thread safe LRU cache
	store       *KeyStoreManager          // manages a key storage directory on disk
	ksWriteChan chan *hmacKeyStoreWriteOp // key store async write channel
	networkID   uint32                    // default conflux network ID
}

// MustNewHmacPrivKeyProviderFromViperWithSecret new HmacPrivKeyProvider instance from viper
// settings and provided HMAC secret.
func MustNewHmacPrivKeyProviderFromViperWithSecret(
	secret []byte, networkID uint32,
) *HmacPrivKeyProvider {
	hmconf := &cryptoConf.PrivateKeyProvider.HMAC

	p, err := NewHmacPrivKeyProvider(
		secret, hmconf.KeystoreDir, hmconf.KeystorePartitions,
		hmconf.LRUCapacity, networkID,
	)
	if err != nil {
		logrus.WithError(err).Fatal(
			"Failed to create HMAC private key provider from viper with provided secret",
		)
	}

	return p
}

// MustNewHmacPrivKeyProviderFromViper new HmacPrivKeyProvider instance from viper settings.
func MustNewHmacPrivKeyProviderFromViper(networkID uint32) *HmacPrivKeyProvider {
	hmconf := &cryptoConf.PrivateKeyProvider.HMAC

	// Read HMAC secret from file path
	data, err := ioutil.ReadFile(hmconf.SecretFile)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to read HMAC secret from file")
	}

	p, err := NewHmacPrivKeyProvider(
		data, hmconf.KeystoreDir, hmconf.KeystorePartitions,
		hmconf.LRUCapacity, networkID,
	)
	if err != nil {
		logrus.WithError(err).Fatal(
			"Failed to create HMAC private key provider from viper",
		)
	}

	if !hmconf.WipeOutSecret {
		return p
	}

	// wipe out secret file in case secret leak out
	if err := os.Remove(hmconf.SecretFile); err != nil {
		logrus.WithError(err).Fatal("Failed to wipe out HMAC secret file")
	}

	return p
}

// NewHmacPrivKeyProvider new HmacPrivKeyProvider constructor.
func NewHmacPrivKeyProvider(
	secret []byte, keydir string, ksPartitions uint32, capacity int, networkID uint32,
) (*HmacPrivKeyProvider, error) {
	// init secret with sha256
	sh := sha256.New()
	if _, err := sh.Write(secret); err != nil {
		return nil, errors.WithMessage(err, "secret SHA256 bytes written error")
	}

	hashedSecretKey := sh.Sum(nil)

	// init LRU cache
	cache, err := lru.New(capacity)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create LRU cache")
	}

	// init key store
	store, err := NewKeyStoreManager(keydir, ksPartitions)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create cache store")
	}

	provider := &HmacPrivKeyProvider{
		secretKey:   hashedSecretKey,
		cache:       cache,
		store:       store,
		ksWriteChan: make(chan *hmacKeyStoreWriteOp, hmacKeyStoreWriteBuffer),
		networkID:   networkID,
	}

	go provider.writeKeyStore() // start keystore writing

	return provider, nil
}

// PublicKey implements PrivKeyProvider interface.
func (p *HmacPrivKeyProvider) PublicKey(kid PrivKeyIdentifier) (*ecdsa.PublicKey, error) {
	key, err := p.getKey(kid)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to derive key")
	}

	return &key.PrivateKey.PublicKey, err
}

// Address implements PrivKeyProvider interface.
func (p *HmacPrivKeyProvider) Address(kid PrivKeyIdentifier, networkID ...uint32) (*cfxaddress.Address, error) {
	key, err := p.getKey(kid)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to derive key")
	}

	// use default network ID unless custom network ID provided
	cfxNetworkID := p.networkID
	if len(networkID) > 0 {
		cfxNetworkID = networkID[0]
	}

	cfxAddr, err := cfxaddress.NewFromCommon(convertCfxAccountAddr(key.Address), cfxNetworkID)
	return &cfxAddr, err
}

// SignHex implements PrivKeyProvider interface.
func (p *HmacPrivKeyProvider) SignHex(kid PrivKeyIdentifier, digestHash types.Hash) (sig []byte, err error) {
	return p.Sign(kid, digestHash.ToCommonHash().Bytes())
}

// Sign implements PrivKeyProvider interface.
func (p *HmacPrivKeyProvider) Sign(kid PrivKeyIdentifier, digestHash []byte) (sig []byte, err error) {
	key, err := p.getKey(kid)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to derive key")
	}

	return crypto.Sign(digestHash, key.PrivateKey)
}

// SignTx implements PrivKeyProvider interface.
func (p *HmacPrivKeyProvider) SignTx(kid PrivKeyIdentifier, tx *types.UnsignedTransaction) ([]byte, error) {
	txHash, err := tx.Hash()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to calculate tx hash")
	}

	sig, err := p.Sign(kid, txHash)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to sign tx hash")
	}

	return tx.EncodeWithSignature(sig[64], sig[0:32], sig[32:64])
}

// getKey gets wrapped key info from LRU cache or mint one.
func (p *HmacPrivKeyProvider) getKey(kid PrivKeyIdentifier) (*gethks.Key, error) {
	logger := logrus.WithField("kid", string(kid))

	// 1. Get key from LRU cache first.
	if cv, ok := p.cache.Get(kid); ok {
		if key, yes := cv.(*gethks.Key); yes {
			logger.Debug("HMAC key provider key hit in the LRU cache")

			return key, nil
		} else {
			logger.Debug("HMAC key provider cached value missed")
		}
	}

	// 2. Generate private key on the fly.
	privKey, err := p.mint(kid)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to mint key")
	}

	key := &gethks.Key{
		Id:         uuid.New(),
		Address:    crypto.PubkeyToAddress(privKey.PublicKey),
		PrivateKey: privKey,
	}

	// store minted key to LRU cache
	p.cache.Add(kid, key)
	// save generated key to keystore asynchronically
	p.ksWriteChan <- &hmacKeyStoreWriteOp{kid: kid, key: key}

	return key, nil
}

// mint Mints a private key from key identifier.
func (p *HmacPrivKeyProvider) mint(kid PrivKeyIdentifier) (*ecdsa.PrivateKey, error) {
	hm := hmac.New(sha256.New, p.secretKey)

	_, err := hm.Write([]byte(kid))
	if err != nil {
		return nil, errors.WithMessage(err, "HMAC bytes written error")
	}

	b := hm.Sum(nil)
	k := new(big.Int).SetBytes(b)

	c := secp256k1.S256()
	n := new(big.Int).Sub(c.Params().N, common.Big1)
	k.Mod(k, n)
	k.Add(k, common.Big1)

	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = c
	priv.D = k
	priv.PublicKey.X, priv.PublicKey.Y = secp256k1.S256().ScalarBaseMult(k.Bytes())

	logrus.WithField("kid", kid).Debug("HMacPrivKeyProvider key minted")

	return priv, nil
}

// writeKeyStore writes key to file.
func (p *HmacPrivKeyProvider) writeKeyStore() {
	for ksw := range p.ksWriteChan {
		if saved, _ := p.store.HasKey(ksw.kid); saved { // already saved?
			continue
		}

		// save key to keystore
		if err := p.store.StoreKey(ksw.kid, ksw.key, string(p.secretKey)); err != nil {
			logrus.WithField("kid", ksw.kid).WithError(err).Error("failed to store key to keystore")
		}
	}
}

// TODO: Use SDK method to construct conflux address from geth common.Address
// once relative issue resolved:
// https://github.com/Conflux-Chain/go-conflux-sdk/issues/116
func convertCfxAccountAddr(ethAddr common.Address) common.Address {
	ethAddr[0] = ethAddr[0]&0x1f | 0x10 // account address in conflux must start with '0x1'
	return ethAddr
}
