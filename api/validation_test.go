package api

import (
	"testing"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestRegexHex(t *testing.T) {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	assert.True(t, ok)

	type req struct {
		V string `binding:"hex"`
	}

	assert.NoError(t, v.Struct(req{V: "0x"}))
	assert.NoError(t, v.Struct(req{V: "0x12abCD"}))

	assert.Error(t, v.Struct(req{V: "0X"}))
	assert.Error(t, v.Struct(req{V: ""}))
	assert.Error(t, v.Struct(req{V: "0x0"}))
	assert.Error(t, v.Struct(req{V: "0x123"}))
	assert.Error(t, v.Struct(req{V: "0x123G"}))
}
