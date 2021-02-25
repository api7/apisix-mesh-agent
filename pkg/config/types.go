package config

import (
	"errors"
	"net"
	"strconv"
)

const (
	// XDSV3FileProvioner means to use the xds v3 file provisioner.
	XDSV3FileProvisioner = "xds-v3-file"
)

var (
	// ErrUnknownProvisioner means user specified an unknown provisioner.
	ErrUnknownProvisioner = errors.New("unknown provisioner")
	// ErrBadGRPCListen means the grpc listen address is invalid.
	ErrBadGRPCListen = errors.New("bad grpc listen address")

	// DefaultGRPCListen is the default gRPC server listen address.
	DefaultGRPCListen = "127.0.0.1:13133"
	// DefaultEtcdKeyPrefix is the default key prefix in the mimicking
	// etcd v3 server.
	DefaultEtcdKeyPrefix = "/apisix"
)

// Config contains configurations required for running apisix-mesh-agent.
type Config struct {
	// The minimum log level that will be printed.
	LogLevel string `json:"log_level" yaml:"log_level"`
	// The destination of logs.
	LogOutput string `json:"log_output" yaml:"log_output"`
	// The Provisioner to use.
	// Value can be "xds-v3-file".
	Provisioner string `json:"provisioner" yaml:"provisioner"`
	// The watched xds files, only valid if the Provisioner is "xds-v3-file"
	XDSWatchFiles []string `json:"xds_watch_files" yaml:"xds_watch_files"`
	// The grpc listen address
	GRPCListen string `json:"grpc_listen" yaml:"grpc_listen"`
	// The key prefix in the mimicking etcd v3 server.
	EtcdKeyPrefix string `json:"etcd_key_prefix" yaml:"etcd_key_prefix"`
}

// NewDefaultConfig returns a Config object with all items filled by
// their default values.
func NewDefaultConfig() *Config {
	return &Config{
		LogLevel:      "info",
		LogOutput:     "stderr",
		Provisioner:   XDSV3FileProvisioner,
		GRPCListen:    DefaultGRPCListen,
		EtcdKeyPrefix: DefaultEtcdKeyPrefix,
	}
}

func (cfg *Config) Validate() error {
	if cfg.Provisioner == "" {
		return errors.New("unspecified provisioner")
	}
	if cfg.Provisioner != XDSV3FileProvisioner {
		return ErrUnknownProvisioner
	}
	ip, port, err := net.SplitHostPort(cfg.GRPCListen)
	if err != nil {
		return ErrBadGRPCListen
	}

	if net.ParseIP(ip) == nil {
		return ErrBadGRPCListen
	}
	pnum, err := strconv.Atoi(port)
	if err != nil || pnum < 1 || pnum > 65535 {
		return ErrBadGRPCListen
	}

	return nil
}
