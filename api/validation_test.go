package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegexHex(t *testing.T) {
	assert.True(t, regexHex.MatchString("0x"))
	assert.True(t, regexHex.MatchString("0x12abCD"))

	assert.False(t, regexHex.MatchString("0X"))
	assert.False(t, regexHex.MatchString(""))
	assert.False(t, regexHex.MatchString("0x0"))
	assert.False(t, regexHex.MatchString("0x123"))
	assert.False(t, regexHex.MatchString("0x123G"))
}
