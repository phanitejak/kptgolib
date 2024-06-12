package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_constants(t *testing.T) {
	assert.Equal(t, ColorRed, "\u001b[31m")
	assert.Equal(t, ColorGreen, "\u001b[32m")
	assert.Equal(t, ColorBlue, "\u001b[34m")
	assert.Equal(t, ColorReset, "\u001b[0m")
}
