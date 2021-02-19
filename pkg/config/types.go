package config

// Config contains configurations required for running apisix-mesh-agent.
type Config struct {
	// The minimum log level that will be printed.
	LogLevel string `json:"log_level" yaml:"log_level"`
	// The destination of logs.
	LogOutput string `json:"log_output" yaml:"log_output"`
	// Whether to use the xds file provisioner.
	UseXDSFileProvisioner bool `json:"use_xds_file_provisioner" yaml:"use_xds_file_provisioner"`
	// The watched xds files, only valid if
	XDSWatchFiles []string `json:"xds_watch_files" yaml:"xds_watch_files"`
}

// NewDefaultConfig returns a Config object with all items filled by
// their default values.
func NewDefaultConfig() *Config {
	return &Config{
		LogLevel:              "info",
		LogOutput:             "stderr",
		UseXDSFileProvisioner: false,
	}
}
