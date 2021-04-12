package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenNodeId(t *testing.T) {
	id := GenNodeId("12345", "10.0.5.3", "default.svc.cluster.local")
	assert.Equal(t, id, "sidecar~10.0.5.3~12345~default.svc.cluster.local")
}
