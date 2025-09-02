package middleware

import (
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testJwtConfig = JwtConfig{
	Issuer:              "dummy issuer",
	Subject:             "dummy subject",
	AccessTokenTimeout:  5 * time.Second,
	RefreshTokenTimeout: 10 * time.Minute,
}

func newTestJwt() *Jwt[int, ed25519.PrivateKey] {
	return MustNewRandomJwt[int](testJwtConfig, SigningMethods.EdDSA)
}

func TestJwtValid(t *testing.T) {
	jwt := newTestJwt()

	// generate tokens
	accessToken, refreshToken, err := jwt.Generate(23)
	assert.NoError(t, err)

	// succeed to validate access token
	claims, err := jwt.Validate(accessToken)
	assert.NoError(t, err)
	assert.Equal(t, testJwtConfig.Issuer, claims.Issuer)
	assert.Equal(t, testJwtConfig.Subject, claims.Subject)
	assert.Equal(t, 23, claims.Data)

	// succeed to validate refresh token
	claims, err = jwt.Validate(refreshToken)
	assert.NoError(t, err)
	assert.Equal(t, testJwtConfig.Issuer, claims.Issuer)
	assert.Equal(t, testJwtConfig.Subject, claims.Subject)
	assert.Equal(t, 23, claims.Data)
}

func TestJwtAccessTokenExpired(t *testing.T) {
	jwt := newTestJwt()

	// generate tokens
	expiredAccessTokenTime := time.Now().Add(-testJwtConfig.AccessTokenTimeout - 5*time.Second)
	accessToken, refreshToken, err := jwt.Generate(23, expiredAccessTokenTime)
	assert.NoError(t, err)

	// access token expired
	claims, err := jwt.Validate(accessToken)
	assert.Error(t, err)
	assert.Nil(t, claims)

	// refresh token valid
	claims, err = jwt.Validate(refreshToken)
	assert.NoError(t, err)
	assert.Equal(t, 23, claims.Data)
}

func TestJwtRefreshTokenExpired(t *testing.T) {
	jwt := newTestJwt()

	// generate tokens
	expiredRefreshTokenTime := time.Now().Add(-testJwtConfig.RefreshTokenTimeout - 5*time.Second)
	accessToken, refreshToken, err := jwt.Generate(23, expiredRefreshTokenTime)
	assert.NoError(t, err)

	// access token expired
	_, err = jwt.Validate(accessToken)
	assert.Error(t, err)

	// refresh token expired
	_, err = jwt.Validate(refreshToken)
	assert.Error(t, err)
}

func TestJwtValidateFromHeader(t *testing.T) {
	jwt := newTestJwt()

	// generate tokens
	accessToken, refreshToken, err := jwt.Generate(23)
	assert.NoError(t, err)

	// no Bearer prefix
	claims, err := jwt.validateFromHeader(accessToken)
	assert.Error(t, err)
	assert.Nil(t, claims)

	// succeed to validate access token
	claims, err = jwt.validateFromHeader("Bearer " + accessToken)
	assert.NoError(t, err)
	assert.Equal(t, 23, claims.Data)

	// succeed to validate refresh token
	claims, err = jwt.validateFromHeader("Bearer " + refreshToken)
	assert.NoError(t, err)
	assert.Equal(t, 23, claims.Data)
}
