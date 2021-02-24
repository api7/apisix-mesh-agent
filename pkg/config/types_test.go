package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()
	assert.Equal(t, cfg.LogLevel, "info")
	assert.Equal(t, cfg.LogOutput, "stderr")
}

func TestConfigValidate(t *testing.T) {
	cfg := NewDefaultConfig()
	cfg.Provisioner = "redis"
	assert.Equal(t, cfg.Validate(), ErrUnknownProvisioner)
}
