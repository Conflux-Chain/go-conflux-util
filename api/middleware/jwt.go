package middleware

import (
	"strings"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/api"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
)

type JwtClaims[T any] struct {
	jwt.RegisteredClaims

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
	if config.AccessTokenTimeout == 0 {
		return nil, errors.New("Access token timeout not specified")
	}

	if config.RefreshTokenTimeout == 0 {
		return nil, errors.New("Refresh token timeout not specified")
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
	if config.AccessTokenTimeout == 0 {
		panic("Access token timeout not specified")
	}

	if config.RefreshTokenTimeout == 0 {
		panic("Refresh token timeout not specified")
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
		Data: data,
	}

	accessToken, err := jwt.NewWithClaims(j.method.JwtSigningMethod(), claims).SignedString(j.key)
	if err != nil {
		return "", "", errors.WithMessage(err, "Failed to generate access token")
	}

	// refresh token
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(j.config.RefreshTokenTimeout))

	refreshToken, err := jwt.NewWithClaims(j.method.JwtSigningMethod(), claims).SignedString(j.key)
	if err != nil {
		return "", "", errors.WithMessage(err, "Failed to generate refresh token")
	}

	return accessToken, refreshToken, nil
}

// Validate validates the specified JWT token and returns parsed claims.
func (j *Jwt[T, KEY]) Validate(tokenString string) (*JwtClaims[T], error) {
	var claims JwtClaims[T]

	if _, err := jwt.ParseWithClaims(tokenString, &claims, j.getKey); err != nil {
		return nil, errors.WithMessage(err, "Failed to parse token")
	}

	return &claims, nil
}

func (j *Jwt[T, KEY]) getKey(token *jwt.Token) (any, error) {
	// verify signing method
	algExp := j.method.JwtSigningMethod().Alg()
	algActual := token.Method.Alg()

	if algExp != algActual {
		return nil, errors.Errorf("Invalid token method, expected = %v, actual = %v", algExp, algActual)
	}

	return j.key.Public(), nil
}

// Middleware validates JWT token from HTTP Authorization header that in format of 'Bearer xxx'.
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

	claims, err := j.Validate(tokenString)
	if err != nil {
		return nil, api.ErrJwt(err.Error())
	}

	return claims, nil
}

// ClaimsFromContext returns JWT claims that injected via middleware.
func (j *Jwt[T, KEY]) ClaimsFromContext(c *gin.Context) (*JwtClaims[T], error) {
	val, ok := c.Get("JWTClaims")
	if !ok {
		return nil, api.ErrJwt("JWT claims not found")
	}

	claims, ok := val.(*JwtClaims[T])
	if !ok {
		return nil, api.ErrJwt("JWT claims type ivnalid")
	}

	return claims, nil
}
