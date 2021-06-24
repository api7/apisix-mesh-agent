package config

import (
	"errors"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()
	assert.Equal(t, cfg.LogLevel, "info")
	assert.Equal(t, cfg.LogOutput, "stderr")
	assert.Equal(t, cfg.Provisioner, XDSV3FileProvisioner)
	assert.Equal(t, cfg.GRPCListen, DefaultGRPCListen)
	assert.Equal(t, cfg.EtcdKeyPrefix, DefaultEtcdKeyPrefix)
	assert.Equal(t, cfg.APISIXHomePath, DefaultAPISIXHomePath)
	assert.Equal(t, cfg.APISIXBinPath, DefaultAPISIXBinPath)
	assert.Equal(t, cfg.RunMode, StandaloneMode)
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
	cfg.Provisioner = "nacos"
	assert.Equal(t, cfg.Validate(), ErrEmptyNacosSource)
}

func TestGetRunningContext(t *testing.T) {
	assert.Nil(t, os.Setenv("POD_NAMESPACE", "apisix"))
	rc := getRunningContext()
	assert.Equal(t, rc.PodNamespace, "apisix")
	assert.Nil(t, os.Setenv("POD_NAMESPACE", ""))
	rc = getRunningContext()
	assert.Equal(t, rc.PodNamespace, "default")
	assert.NotNil(t, net.ParseIP(rc.IPAddress))
}
