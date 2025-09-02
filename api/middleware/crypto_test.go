package middleware

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSigningMethods(t *testing.T) {
	testSigningMethod(t, SigningMethods.RS256)
	testSigningMethod(t, SigningMethods.EdDSA)

	// EdDSA key in PEM format without type info
	_, err := SigningMethods.EdDSA.FromPEM([]byte("MC4CAQAwBQYDK2VwBCIEIJwhN7FdPh7hUGP7rK2m370+XUI0r3VdIGkNsHAFQn3X"))
	assert.NoError(t, err)
}

func testSigningMethod[T PrivateKey](t *testing.T, method SigningMethod[T]) {
	key := method.MustGenerateKey()

	encoded := method.ToPEM(key)

	decoded, err := method.FromPEM(encoded)
	assert.NoError(t, err)
	assert.True(t, decoded.Equal(key))
}
