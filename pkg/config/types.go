package config

// Config contains configurations required for running apisix-mesh-agent.
type Config struct {
	LogLevel  string `json:"log_level" yaml:"log_level"`
	LogOutput string `json:"log_output" yaml:"log_output"`
}

// NewDefaultConfig returns a Config object with all items filled by
// their default values.
func NewDefaultConfig() *Config {
	return &Config{
		LogLevel:  "info",
		LogOutput: "stderr",
	}
}
