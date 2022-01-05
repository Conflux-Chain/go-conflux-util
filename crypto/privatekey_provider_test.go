package crypto

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	commonutil "github.com/Conflux-Chain/go-conflux-util/common"
	"github.com/ethereum/go-ethereum/accounts"
	gethks "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	hmacProvider     *HmacPrivKeyProvider
	hmacSecretKey    = "secret!2#4%6&*9)"
	hmacTmpKeystore  = "./tmp_key"
	hmacLRUCapacity  = 1000
	hmacKSPartitions = uint32(100)
	cfxNetworkID     = cfxaddress.NetworkTypeTestnetID
)

func setup() error {
	var err error

	// set up HMAC private key provider
	hmacProvider, err = NewHmacPrivKeyProvider(
		[]byte(hmacSecretKey), hmacTmpKeystore, hmacKSPartitions,
		hmacLRUCapacity, cfxNetworkID,
	)
	return err
}

func teardown() (err error) {
	return os.RemoveAll(hmacTmpKeystore) // delete HMAC key store directory
}

func TestMain(m *testing.M) {
	if err := setup(); err != nil {
		panic(errors.WithMessage(err, "failed to setup"))
	}

	code := m.Run()

	if err := teardown(); err != nil {
		panic(errors.WithMessage(err, "failed to tear down"))
	}

	os.Exit(code)
}

func TestHmacProviderGetKey(t *testing.T) {
	num := 100
	keys := make([]*gethks.Key, 0, num)

	for i := 0; i < num; i++ {
		k, err := hmacProvider.getKey(PrivKeyIdentifier(fmt.Sprint(i)))
		assert.NoError(t, err)
		keys = append(keys, k)
	}

	for j := 0; j < num; j++ { // test caching
		k, err := hmacProvider.getKey(PrivKeyIdentifier(fmt.Sprint(j)))
		assert.NoError(t, err)
		assert.Equal(t, k, keys[j])
	}
}

func TestHmacProviderPublicKey(t *testing.T) {
	num := 100
	for i := 0; i < num; i++ {
		pubKey, err := hmacProvider.PublicKey(PrivKeyIdentifier(fmt.Sprint(i)))
		assert.NoError(t, err)
		assert.NotNil(t, pubKey)
	}
}

func TestHmacProviderAddress(t *testing.T) {
	num := 100
	for i := 0; i < num; i++ {
		kid := PrivKeyIdentifier(fmt.Sprint(i))
		maddr, err := hmacProvider.Address(kid, cfxaddress.NetowrkTypeMainnetID)
		assert.NoError(t, err)
		assert.NotNil(t, maddr)

		addr, err := hmacProvider.Address(kid)
		assert.NoError(t, err)
		assert.NotNil(t, addr)

		assert.NotEqual(t, maddr, addr)

		taddr, err := hmacProvider.Address(kid, cfxaddress.NetworkTypeTestnetID)
		assert.NoError(t, err)
		assert.NotNil(t, taddr)

		assert.Equal(t, addr, taddr)
	}
}

func TestHmacProviderSign(t *testing.T) {
	num := 100
	for i := 0; i < num; i++ {
		kid := PrivKeyIdentifier(fmt.Sprint(i))

		// signing
		h := sha256.New()
		_, err := h.Write([]byte(fmt.Sprintf("message %v", i)))
		assert.NoError(t, err)

		digestHash := h.Sum(nil)
		sig, err := hmacProvider.Sign(kid, digestHash)
		assert.NoError(t, err)
		assert.Greater(t, len(sig), 64)

		// verification
		pubKey, err := hmacProvider.PublicKey(kid)
		assert.NoError(t, err)

		pubKeyBytes := crypto.CompressPubkey(pubKey)
		ok := crypto.VerifySignature(pubKeyBytes, digestHash, sig[0:64])
		assert.True(t, ok)

		r, s := new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:64])
		ok = ecdsa.Verify(pubKey, digestHash, r, s)
		assert.True(t, ok)
	}
}

func TestHmacProviderGethKeyStoreLoad(t *testing.T) {
	km := hmacProvider.store

	kid := PrivKeyIdentifier(fmt.Sprint(10000))

	publicKey, err := hmacProvider.PublicKey(kid)
	assert.NoError(t, err)

	time.Sleep(time.Second * 2) // sleep for a while to wait for keygen
	commonAddr := crypto.PubkeyToAddress(*publicKey)

	gks := gethks.NewKeyStore(filepath.Dir(km.keyPath(kid)), keyStoreEncryptionScryptN, keyStoreEncryptionScryptP)
	assert.True(t, gks.HasAddress(commonAddr))

	accts := gks.Accounts()
	assert.NotEmpty(t, accts)

	var refAcct accounts.Account
	for _, act := range accts {
		if reflect.DeepEqual(act.Address, commonAddr) {
			refAcct = act
			break
		}
	}
	assert.NotNil(t, refAcct)

	err = gks.Unlock(refAcct, string(hmacProvider.secretKey))
	assert.NoError(t, err)

	// verify signing
	h := sha256.New()
	_, err = h.Write([]byte("test message"))
	assert.NoError(t, err)

	digestHash := h.Sum(nil)
	sig, err := hmacProvider.Sign(kid, digestHash)
	assert.NoError(t, err)

	sig2, err := gks.SignHash(refAcct, digestHash)
	assert.NoError(t, err)

	assert.Equal(t, sig, sig2)
}

func BenchmarkHmacProviderMint(b *testing.B) {
	v := commonutil.RandUint64(math.MaxUint64)

	b.ResetTimer() // reset timer

	_, err := hmacProvider.mint(PrivKeyIdentifier(fmt.Sprint(v)))
	if err != nil {
		b.Fatalf("failed to mint for test case (v=%v)", v)
	}
}

func BenchmarkHmacProviderSign(b *testing.B) {
	v := commonutil.RandUint64(math.MaxUint64)
	kid := PrivKeyIdentifier(fmt.Sprint(v))

	// signing
	h := sha256.New()
	_, err := h.Write([]byte(fmt.Sprintf("message %v", v)))
	if err != nil {
		b.Fatalf("failed to sha256 message for test case (v=%v)", v)
	}

	digestHash := h.Sum(nil)

	b.ResetTimer() // reset timer

	_, err = hmacProvider.Sign(kid, digestHash)
	if err != nil {
		b.Fatalf("failed to sign for test case (v=%v)", v)
	}
}
