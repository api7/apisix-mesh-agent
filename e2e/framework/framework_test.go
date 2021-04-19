package framework

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultFramework(t *testing.T) {
	f, err := NewDefaultFramework()
	assert.Nil(t, err)
	assert.Equal(t, f.cp.Namespace(), f.cpNamespace)
	assert.Equal(t, f.cp.Type(), "istio")

	err = f.Deploy()
	assert.Nil(t, err)
}
