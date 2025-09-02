package middleware

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var errInvalidPem = errors.New("invalid PEM data")

type PrivateKey interface {
	Public() crypto.PublicKey
	Equal(x crypto.PrivateKey) bool
}

type SigningMethod[T PrivateKey] interface {
	MustGenerateKey() T
	ToPEM(key T) []byte
	FromPEM(data []byte) (T, error)

	JwtSigningMethod() jwt.SigningMethod
}

var SigningMethods struct {
	// RSA
	RS256 SigningMethod[*rsa.PrivateKey]

	// EdDSA
	EdDSA SigningMethod[ed25519.PrivateKey]
}

func init() {
	SigningMethods.RS256 = &signingMethodRSA{}
	SigningMethods.EdDSA = &signingMethodEdDSA{}
}

///////////////////////////////////////////////////////////////////////////
//
// RSA signing method
//
///////////////////////////////////////////////////////////////////////////

type signingMethodRSA struct{}

func (method *signingMethodRSA) MustGenerateKey() *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	return key
}

func (method *signingMethodRSA) ToPEM(key *rsa.PrivateKey) []byte {
	block := pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	return pem.EncodeToMemory(&block)
}

func (method *signingMethodRSA) FromPEM(data []byte) (*rsa.PrivateKey, error) {
	data = ensurePrivateKeyType(
		string(data),
		"-----BEGIN RSA PRIVATE KEY-----",
		"-----END RSA PRIVATE KEY-----",
	)

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errInvalidPem
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func ensurePrivateKeyType(data, prefix, suffix string) []byte {
	if !strings.HasPrefix(data, prefix) {
		data = prefix + "\n" + data
	}

	if !strings.HasSuffix(data, suffix) {
		data = data + "\n" + suffix
	}

	return []byte(data)
}

func (method *signingMethodRSA) JwtSigningMethod() jwt.SigningMethod {
	return jwt.SigningMethodRS256
}

///////////////////////////////////////////////////////////////////////////
//
// EdDSA signing method
//
///////////////////////////////////////////////////////////////////////////

type signingMethodEdDSA struct{}

func (method *signingMethodEdDSA) MustGenerateKey() ed25519.PrivateKey {
	_, prv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	return prv
}

func (method *signingMethodEdDSA) ToPEM(key ed25519.PrivateKey) []byte {
	encoded, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		panic(err)
	}

	block := pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: encoded,
	}

	return pem.EncodeToMemory(&block)
}

func (method *signingMethodEdDSA) FromPEM(data []byte) (ed25519.PrivateKey, error) {
	data = ensurePrivateKeyType(
		string(data),
		"-----BEGIN PRIVATE KEY-----",
		"-----END PRIVATE KEY-----",
	)

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errInvalidPem
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return key.(ed25519.PrivateKey), nil
}

func (method *signingMethodEdDSA) JwtSigningMethod() jwt.SigningMethod {
	return jwt.SigningMethodEdDSA
}
