package account

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSafeBigFillBytes(t *testing.T) {
	// buf size: 0
	var buf0 [0]byte
	SafeBigFillBytes(big.NewInt(1), buf0[:])
	assert.Equal(t, [0]byte{}, buf0)

	// buf size: 1
	var buf1 [1]byte
	SafeBigFillBytes(big.NewInt(0xFF), buf1[:])
	assert.Equal(t, [1]byte{0xFF}, buf1)

	// buf size: 4
	var buf4 [4]byte
	SafeBigFillBytes(big.NewInt(0xFFFFFFFF), buf4[:])
	assert.Equal(t, [4]byte{0xFF, 0xFF, 0xFF, 0xFF}, buf4)

	// buf size: 4 - overwrite
	SafeBigFillBytes(big.NewInt(0x00000001), buf4[:])
	assert.Equal(t, [4]byte{0x00, 0x00, 0x00, 0x01}, buf4)

	// buf size: 4 - truncated
	SafeBigFillBytes(big.NewInt(0xAABBCCDDEEFF), buf4[:])
	assert.Equal(t, [4]byte{0xCC, 0xDD, 0xEE, 0xFF}, buf4)

	// num is nil - buf unchanged
	SafeBigFillBytes(nil, buf4[:])
	assert.Equal(t, [4]byte{0xCC, 0xDD, 0xEE, 0xFF}, buf4)
}
