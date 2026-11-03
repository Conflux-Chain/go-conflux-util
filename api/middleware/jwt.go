package middleware

import (
	"strings"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/api"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

const (
	tokenUseAccess  = "access"
	tokenUseRefresh = "refresh"
)

type JwtClaims[T any] struct {
	jwt.RegisteredClaims

	// TokenUse identifies the token's purpose: "access" or "refresh".
	TokenUse string `json:"tokenUse"`

	// Custom data carried by the token.
	Data T
}

type JwtConfig struct {
	Issuer  string
	Subject string

	AccessTokenTimeout  time.Duration `default:"10m"` // 10 minutes
	RefreshTokenTimeout time.Duration `default:"24h"` // 1 day

	Key string // private key in PEM format, key type could be ignored
}

type Jwt[DATA any, KEY PrivateKey] struct {
	config JwtConfig
	method SigningMethod[KEY]
	key    KEY
}

func NewJwt[T any, KEY PrivateKey](config JwtConfig, method SigningMethod[KEY]) (*Jwt[T, KEY], error) {
	if config.AccessTokenTimeout <= 0 {
		return nil, errors.New("Access token timeout must be positive")
	}

	if config.RefreshTokenTimeout <= 0 {
		return nil, errors.New("Refresh token timeout must be positive")
	}

	if len(config.Key) == 0 {
		return nil, errors.New("JWT key not specified")
	}

	key, err := method.FromPEM([]byte(config.Key))
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to parse private key in PEM format")
	}

	return &Jwt[T, KEY]{config, method, key}, nil
}

func MustNewRandomJwt[T any, KEY PrivateKey](config JwtConfig, method SigningMethod[KEY]) *Jwt[T, KEY] {
	if config.AccessTokenTimeout <= 0 {
		panic("Access token timeout must be positive")
	}

	if config.RefreshTokenTimeout <= 0 {
		panic("Refresh token timeout must be positive")
	}

	return &Jwt[T, KEY]{config, method, method.MustGenerateKey()}
}

func (j *Jwt[T, KEY]) Config() JwtConfig {
	return j.config
}

// Generate generates access token and refresh token.
func (j *Jwt[T, KEY]) Generate(data T, optNow ...time.Time) (string, string, error) {
	now := time.Now()
	if len(optNow) > 0 {
		now = optNow[0]
	}

	// access token
	claims := JwtClaims[T]{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.config.Issuer,
			Subject:   j.config.Subject,
			ExpiresAt: jwt.NewNumericDate(now.Add(j.config.AccessTokenTimeout)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		TokenUse: tokenUseAccess,
		Data:     data,
	}

	accessToken, err := jwt.NewWithClaims(j.method.JwtSigningMethod(), claims).SignedString(j.key)
	if err != nil {
		return "", "", errors.WithMessage(err, "Failed to generate access token")
	}

	// refresh token
	claims.TokenUse = tokenUseRefresh
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(j.config.RefreshTokenTimeout))

	refreshToken, err := jwt.NewWithClaims(j.method.JwtSigningMethod(), claims).SignedString(j.key)
	if err != nil {
		return "", "", errors.WithMessage(err, "Failed to generate refresh token")
	}

	return accessToken, refreshToken, nil
}

// validate checks the token's signature, registered claims, and expected purpose.
func (j *Jwt[T, KEY]) validate(tokenString, expectedTokenUse string) (*JwtClaims[T], error) {
	options := []jwt.ParserOption{
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods([]string{j.method.JwtSigningMethod().Alg()}),
		jwt.WithIssuer(j.config.Issuer),
		jwt.WithSubject(j.config.Subject),
	}

	keyFunc := func(_ *jwt.Token) (any, error) {
		return j.key.Public(), nil
	}

	var claims JwtClaims[T]
	if _, err := jwt.ParseWithClaims(tokenString, &claims, keyFunc, options...); err != nil {
		return nil, errors.WithMessage(err, "Failed to parse token")
	}

	if claims.TokenUse != expectedTokenUse {
		return nil, errors.Errorf("Invalid token use, expected = %v, actual = %v", expectedTokenUse, claims.TokenUse)
	}

	return &claims, nil
}

// ValidateAccessToken validates an access token, rejecting refresh tokens and
// legacy tokens without a tokenUse claim. The exp claim is required.
func (j *Jwt[T, KEY]) ValidateAccessToken(tokenString string) (*JwtClaims[T], error) {
	return j.validate(tokenString, tokenUseAccess)
}

// ValidateRefreshToken validates a refresh token, rejecting access tokens and
// legacy tokens without a tokenUse claim. The exp claim is required.
func (j *Jwt[T, KEY]) ValidateRefreshToken(tokenString string) (*JwtClaims[T], error) {
	return j.validate(tokenString, tokenUseRefresh)
}

// Middleware validates an access token from the HTTP Authorization header in
// the format 'Bearer xxx'. Refresh tokens and tokens without tokenUse are rejected.
func (j *Jwt[T, KEY]) Middleware(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")

	claims, err := j.validateFromHeader(authHeader)
	if err != nil {
		ResponseError(c, err)
		c.Abort()
	} else {
		c.Set("JWTClaims", claims)
		c.Next()
	}
}
func (j *Jwt[T, KEY]) validateFromHeader(authHeader string) (*JwtClaims[T], error) {
	if len(authHeader) == 0 {
		return nil, api.ErrJwt("Authorization header not specified")
	}

	const bearer string = "Bearer "

	if !strings.HasPrefix(authHeader, bearer) {
		return nil, api.ErrJwt("Authorization header should be in format 'Bearer xxx'")
	}

	tokenString := authHeader[len(bearer):]

	claims, err := j.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, api.ErrJwt(err.Error())
	}

	return claims, nil
}

// ClaimsFromContext returns JWT claims that injected via middleware.
func (j *Jwt[T, KEY]) ClaimsFromContext(c *gin.Context) (data T, err error) {
	claims, err := j.RawClaimsFromContext(c)
	if err != nil {
		return data, err
	}

	return claims.Data, nil
}

// ClaimsFromContext returns JWT claims that injected via middleware.
func (j *Jwt[T, KEY]) RawClaimsFromContext(c *gin.Context) (*JwtClaims[T], error) {
	val, ok := c.Get("JWTClaims")
	if !ok {
		return nil, api.ErrJwt("JWT claims not found")
	}

	claims, ok := val.(*JwtClaims[T])
	if !ok {
		return nil, api.ErrJwt("JWT claims type invalid")
	}

	return claims, nil
}
