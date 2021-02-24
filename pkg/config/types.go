package config

import (
	"errors"
)

const (
	// XDSV3FileProvioner means to use the xds v3 file provisioner.
	XDSV3FileProvisioner = "xds-v3-file"
)

var (
	// ErrUnknownProvisioner means user specified an unknown provisioner.
	ErrUnknownProvisioner = errors.New("unknown provisioner")
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
	// The watched xds files, only valid if
	XDSWatchFiles []string `json:"xds_watch_files" yaml:"xds_watch_files"`
}

// NewDefaultConfig returns a Config object with all items filled by
// their default values.
func NewDefaultConfig() *Config {
	return &Config{
		LogLevel:    "info",
		LogOutput:   "stderr",
		Provisioner: XDSV3FileProvisioner,
	}
}

func (cfg *Config) Validate() error {
	if cfg.Provisioner == "" {
		return errors.New("unspecified provisioner")
	}
	if cfg.Provisioner != XDSV3FileProvisioner {
		return ErrUnknownProvisioner
	}
	return nil
}
