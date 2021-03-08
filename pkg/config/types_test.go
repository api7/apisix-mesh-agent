package config

import (
	"errors"
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

	cfg.Provisioner = ""
	assert.Equal(t, cfg.Validate(), errors.New("unspecified provisioner"))

	cfg = NewDefaultConfig()
	cfg.GRPCListen = "127:8080"
	assert.Equal(t, cfg.Validate(), ErrBadGRPCListen)
	cfg.GRPCListen = "127.0.0.1:aa"
	assert.Equal(t, cfg.Validate(), ErrBadGRPCListen)
	cfg.GRPCListen = "hello"
	assert.Equal(t, cfg.Validate(), ErrBadGRPCListen)

	cfg.Provisioner = "xds-v3-grpc"
	assert.Equal(t, cfg.Validate(), ErrEmptyXDSConfigSource)
}
