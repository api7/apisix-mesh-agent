package id

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenID(t *testing.T) {
	hash := GenID("")
	assert.Len(t, hash, 0)

	assert.Equal(t, GenID("111"), GenID("111"))
	assert.NotEqual(t, GenID("112"), GenID("111"))
}
