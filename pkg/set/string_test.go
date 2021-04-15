package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringSet(t *testing.T) {
	s := StringSet{}
	s.Add("123")
	s.Add("456")
	s2 := StringSet{}
	s2.Add("123")
	s2.Add("456")

	assert.Equal(t, s.Equal(s2), true)
	s2.Add("111")
	assert.Equal(t, s.Equal(s2), false)
}
