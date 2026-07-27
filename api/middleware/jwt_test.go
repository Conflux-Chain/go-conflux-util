package middleware

import (
	"crypto/ed25519"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

func mustSignToken[T any, KEY PrivateKey](t *testing.T, auth *Jwt[T, KEY], claims JwtClaims[T]) string {
	t.Helper()

	token, err := jwt.NewWithClaims(auth.method.JwtSigningMethod(), claims).SignedString(auth.key)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	return token
}

func TestJwtValid(t *testing.T) {
	auth := newTestJwt()

	// generate tokens
	accessToken, refreshToken, err := auth.Generate(23)
	assert.NoError(t, err)

	// succeed to validate access token
	claims, err := auth.Validate(accessToken)
	assert.NoError(t, err)
	assert.Equal(t, testJwtConfig.Issuer, claims.Issuer)
	assert.Equal(t, testJwtConfig.Subject, claims.Subject)
	assert.Equal(t, 23, claims.Data)

	// succeed to validate refresh token
	claims, err = auth.Validate(refreshToken)
	assert.NoError(t, err)
	assert.Equal(t, testJwtConfig.Issuer, claims.Issuer)
	assert.Equal(t, testJwtConfig.Subject, claims.Subject)
	assert.Equal(t, 23, claims.Data)
}

func TestJwtValidateRejectsMismatchedIssuerOrSubject(t *testing.T) {
	auth := newTestJwt()
	now := time.Now()

	claims := JwtClaims[int]{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "other issuer",
			Subject:   testJwtConfig.Subject,
			ExpiresAt: jwt.NewNumericDate(now.Add(testJwtConfig.AccessTokenTimeout)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		Data: 23,
	}

	token := mustSignToken(t, auth, claims)
	parsed, err := auth.Validate(token)
	assert.Error(t, err)
	assert.Nil(t, parsed)

	claims.Issuer = testJwtConfig.Issuer
	claims.Subject = "other subject"
	token = mustSignToken(t, auth, claims)
	parsed, err = auth.Validate(token)
	assert.Error(t, err)
	assert.Nil(t, parsed)
}

func TestJwtAccessTokenExpired(t *testing.T) {
	auth := newTestJwt()

	// generate tokens
	expiredAccessTokenTime := time.Now().Add(-testJwtConfig.AccessTokenTimeout - 5*time.Second)
	accessToken, refreshToken, err := auth.Generate(23, expiredAccessTokenTime)
	assert.NoError(t, err)

	// access token expired
	claims, err := auth.Validate(accessToken)
	assert.Error(t, err)
	assert.Nil(t, claims)

	// refresh token valid
	claims, err = auth.Validate(refreshToken)
	assert.NoError(t, err)
	assert.Equal(t, 23, claims.Data)
}

func TestJwtRefreshTokenExpired(t *testing.T) {
	auth := newTestJwt()

	// generate tokens
	expiredRefreshTokenTime := time.Now().Add(-testJwtConfig.RefreshTokenTimeout - 5*time.Second)
	accessToken, refreshToken, err := auth.Generate(23, expiredRefreshTokenTime)
	assert.NoError(t, err)

	// access token expired
	_, err = auth.Validate(accessToken)
	assert.Error(t, err)

	// refresh token expired
	_, err = auth.Validate(refreshToken)
	assert.Error(t, err)
}

func TestJwtValidateFromHeader(t *testing.T) {
	auth := newTestJwt()

	// generate tokens
	accessToken, refreshToken, err := auth.Generate(23)
	assert.NoError(t, err)

	// no Bearer prefix
	claims, err := auth.validateFromHeader(accessToken)
	assert.Error(t, err)
	assert.Nil(t, claims)

	// succeed to validate access token
	claims, err = auth.validateFromHeader("Bearer " + accessToken)
	assert.NoError(t, err)
	assert.Equal(t, 23, claims.Data)

	// succeed to validate refresh token
	claims, err = auth.validateFromHeader("Bearer " + refreshToken)
	assert.NoError(t, err)
	assert.Equal(t, 23, claims.Data)
}
