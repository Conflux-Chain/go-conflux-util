package middleware

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/api"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestJwtConstructorTimeouts(t *testing.T) {
	method := SigningMethods.EdDSA
	keyPEM := string(method.ToPEM(method.MustGenerateKey()))
	for _, tc := range []struct {
		name            string
		access, refresh time.Duration
		wantErr         string
	}{
		{"positive", time.Minute, time.Hour, ""},
		{"zero access", 0, time.Hour, "Access token timeout must be positive"},
		{"negative access", -time.Minute, time.Hour, "Access token timeout must be positive"},
		{"zero refresh", time.Minute, 0, "Refresh token timeout must be positive"},
		{"negative refresh", time.Minute, -time.Hour, "Refresh token timeout must be positive"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := testJwtConfig
			config.Key = keyPEM
			config.AccessTokenTimeout, config.RefreshTokenTimeout = tc.access, tc.refresh
			auth, err := NewJwt[int](config, method)
			if tc.wantErr != "" {
				assert.EqualError(t, err, tc.wantErr)
				assert.Nil(t, auth)
				assert.PanicsWithValue(t, tc.wantErr, func() {
					MustNewRandomJwt[int](config, method)
				})
				return
			}
			require.NoError(t, err)
			require.NotNil(t, auth)
			assert.NotPanics(t, func() { MustNewRandomJwt[int](config, method) })
		})
	}
}

func TestJwtValid(t *testing.T) {
	auth := newTestJwt()

	// generate tokens
	accessToken, refreshToken, err := auth.Generate(23)
	assert.NoError(t, err)

	// succeed to validate access token
	claims, err := auth.ValidateAccessToken(accessToken)
	assert.NoError(t, err)
	assert.Equal(t, testJwtConfig.Issuer, claims.Issuer)
	assert.Equal(t, testJwtConfig.Subject, claims.Subject)
	assert.Equal(t, 23, claims.Data)

	// succeed to validate refresh token
	claims, err = auth.ValidateRefreshToken(refreshToken)
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
		TokenUse: tokenUseAccess,
		Data:     23,
	}

	token := mustSignToken(t, auth, claims)
	parsed, err := auth.ValidateAccessToken(token)
	assert.ErrorIs(t, err, jwt.ErrTokenInvalidIssuer)
	assert.Nil(t, parsed)

	claims.Issuer = testJwtConfig.Issuer
	claims.Subject = "other subject"
	token = mustSignToken(t, auth, claims)
	parsed, err = auth.ValidateAccessToken(token)
	assert.ErrorIs(t, err, jwt.ErrTokenInvalidSubject)
	assert.Nil(t, parsed)
}

func TestJwtRequiresExpiration(t *testing.T) {
	auth := newTestJwt()
	for _, tc := range []struct {
		purpose  string
		validate func(string) (*JwtClaims[int], error)
	}{
		{tokenUseAccess, auth.ValidateAccessToken},
		{tokenUseRefresh, auth.ValidateRefreshToken},
	} {
		t.Run(tc.purpose, func(t *testing.T) {
			token := mustSignToken(t, auth, JwtClaims[int]{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:  testJwtConfig.Issuer,
					Subject: testJwtConfig.Subject,
				},
				TokenUse: tc.purpose,
			})
			claims, err := tc.validate(token)
			assert.ErrorIs(t, err, jwt.ErrTokenRequiredClaimMissing)
			assert.Nil(t, claims)
		})
	}
}

func TestJwtIssuerAndSubjectOptions(t *testing.T) {
	for _, tc := range []struct {
		name, issuer, subject, tokenIssuer, tokenSubject string
		wantErr                                          error
	}{
		{"missing issuer", "issuer", "subject", "", "subject", jwt.ErrTokenRequiredClaimMissing},
		{"missing subject", "issuer", "subject", "issuer", "", jwt.ErrTokenRequiredClaimMissing},
		{"issuer unrestricted", "", "subject", "other issuer", "subject", nil},
		{"subject unrestricted", "issuer", "", "issuer", "other subject", nil},
		{"both absent", "", "", "", "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := testJwtConfig
			config.Issuer, config.Subject = tc.issuer, tc.subject
			auth := MustNewRandomJwt[int](config, SigningMethods.EdDSA)
			for _, check := range []struct {
				purpose  string
				validate func(string) (*JwtClaims[int], error)
			}{
				{tokenUseAccess, auth.ValidateAccessToken},
				{tokenUseRefresh, auth.ValidateRefreshToken},
			} {
				token := mustSignToken(t, auth, JwtClaims[int]{
					RegisteredClaims: jwt.RegisteredClaims{
						Issuer:    tc.tokenIssuer,
						Subject:   tc.tokenSubject,
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
					},
					TokenUse: check.purpose,
				})
				claims, err := check.validate(token)
				if tc.wantErr != nil {
					assert.ErrorIs(t, err, tc.wantErr)
					assert.Nil(t, claims)
				} else {
					require.NoError(t, err)
					require.NotNil(t, claims)
				}
			}
		})
	}
}

func TestJwtRejectsUnexpectedSigningMethod(t *testing.T) {
	auth := MustNewRandomJwt[int](testJwtConfig, SigningMethods.RS256)
	for _, check := range []struct {
		purpose  string
		validate func(string) (*JwtClaims[int], error)
	}{
		{tokenUseAccess, auth.ValidateAccessToken},
		{tokenUseRefresh, auth.ValidateRefreshToken},
	} {
		t.Run(check.purpose, func(t *testing.T) {
			claims := JwtClaims[int]{
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    testJwtConfig.Issuer,
					Subject:   testJwtConfig.Subject,
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
				},
				TokenUse: check.purpose,
			}
			_, err := check.validate(mustSignToken(t, auth, claims))
			require.NoError(t, err)

			// The same RSA key can verify RS512, but this instance only permits RS256.
			token, err := jwt.NewWithClaims(jwt.SigningMethodRS512, claims).SignedString(auth.key)
			require.NoError(t, err)
			_, err = jwt.Parse(token, func(_ *jwt.Token) (any, error) {
				return auth.key.Public(), nil
			})
			require.NoError(t, err)
			parsed, err := check.validate(token)
			assert.ErrorIs(t, err, jwt.ErrTokenSignatureInvalid)
			assert.Nil(t, parsed)
		})
	}
}

func TestJwtAccessTokenExpired(t *testing.T) {
	auth := newTestJwt()

	// generate tokens
	expiredAccessTokenTime := time.Now().Add(-testJwtConfig.AccessTokenTimeout - 5*time.Second)
	accessToken, refreshToken, err := auth.Generate(23, expiredAccessTokenTime)
	assert.NoError(t, err)

	// access token expired
	claims, err := auth.ValidateAccessToken(accessToken)
	assert.Error(t, err)
	assert.Nil(t, claims)

	// refresh token valid
	claims, err = auth.ValidateRefreshToken(refreshToken)
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
	_, err = auth.ValidateAccessToken(accessToken)
	assert.Error(t, err)

	// refresh token expired
	_, err = auth.ValidateRefreshToken(refreshToken)
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

	// refresh tokens must not authenticate business requests
	claims, err = auth.validateFromHeader("Bearer " + refreshToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJwtTokenUse(t *testing.T) {
	config := testJwtConfig
	config.AccessTokenTimeout = time.Minute
	auth := MustNewRandomJwt[int](config, SigningMethods.EdDSA)
	access, refresh, err := auth.Generate(23)
	require.NoError(t, err)
	expiredAccess, expiredRefresh, err := auth.Generate(23, time.Now().Add(-2*config.RefreshTokenTimeout))
	require.NoError(t, err)

	// Inspect the actual signed payload to enforce the camelCase wire format.
	for token, purpose := range map[string]string{access: tokenUseAccess, refresh: tokenUseRefresh} {
		parsed, err := jwt.Parse(token, func(_ *jwt.Token) (any, error) {
			return auth.key.Public(), nil
		})
		require.NoError(t, err)
		payload := parsed.Claims.(jwt.MapClaims)
		assert.Equal(t, purpose, payload["tokenUse"])
		assert.NotContains(t, payload, "token_use")
		assert.NotContains(t, payload, "TokenUse")
	}

	claims, err := auth.ValidateAccessToken(access)
	require.NoError(t, err)
	// Sign a token in the old format, with no tokenUse field at all.
	legacy, err := jwt.NewWithClaims(auth.method.JwtSigningMethod(), struct {
		jwt.RegisteredClaims
		Data int
	}{claims.RegisteredClaims, 23}).SignedString(auth.key)
	require.NoError(t, err)
	claims.TokenUse = "unknown"
	unknown := mustSignToken(t, auth, *claims)
	claims.TokenUse = tokenUseAccess
	wrongKey := mustSignToken(t, newTestJwt(), *claims)

	for _, tc := range []struct {
		name, token         string
		accessOK, refreshOK bool
	}{
		{"access", access, true, false},
		{"refresh", refresh, false, true},
		{"expired access", expiredAccess, false, false},
		{"expired refresh", expiredRefresh, false, false},
		{"legacy", legacy, false, false},
		{"unknown purpose", unknown, false, false},
		{"wrong signing key", wrongKey, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, check := range []struct {
				validate func(string) (*JwtClaims[int], error)
				ok       bool
			}{
				{auth.ValidateAccessToken, tc.accessOK},
				{auth.ValidateRefreshToken, tc.refreshOK},
			} {
				parsed, err := check.validate(tc.token)
				if check.ok {
					require.NoError(t, err)
					assert.Equal(t, 23, parsed.Data)
				} else {
					require.Error(t, err)
					assert.Nil(t, parsed)
				}
			}

			reached := false
			router := gin.New()
			router.GET("/private", auth.Middleware, func(c *gin.Context) {
				reached = true
				data, err := auth.ClaimsFromContext(c)
				require.NoError(t, err)
				assert.Equal(t, 23, data)
				c.Status(http.StatusNoContent)
			})
			req := httptest.NewRequest(http.MethodGet, "/private", nil)
			req.Header.Set("Authorization", "Bearer "+tc.token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			assert.Equal(t, tc.accessOK, reached)
			if tc.accessOK {
				assert.Equal(t, http.StatusNoContent, response.Code)
			} else {
				assert.Equal(t, http.StatusOK, response.Code)
				var result api.BusinessError
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
				assert.Equal(t, api.ErrCodeJwt, result.Code)
			}
		})
	}
}
