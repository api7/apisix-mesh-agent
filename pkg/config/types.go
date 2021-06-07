package config

import (
	"errors"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

const (
	// XDSV3FileProvisioner means to use the xds v3 file provisioner.
	XDSV3FileProvisioner = "xds-v3-file"
	// XDSV3GRPCProvisioner means to use the xds v3 grpc provisioner.
	XDSV3GRPCProvisioner = "xds-v3-grpc"
	// NacosProvisioner means to use nacos provisioner.
	NacosProvisioner = "nacos"

	// StandaloneMode means run apisix-mesh-agent standalone.
	StandaloneMode = "standalone"
	// BundleMode means run apisix-mesh-agent and apisix.
	BundleMode = "bundle"
	// DefaultAPISIXHomePath is the default home path for Apache APISIX.
	DefaultAPISIXHomePath = "/usr/local/apisix"
	// DefaultAPISIXBinPath is the default binary path for Apache APISIX.
	DefaultAPISIXBinPath = "/usr/local/bin/apisix"
)

var (
	// ErrUnknownProvisioner means user specified an unknown provisioner.
	ErrUnknownProvisioner = errors.New("unknown provisioner")
	// ErrBadGRPCListen means the grpc listen address is invalid.
	ErrBadGRPCListen = errors.New("bad grpc listen address")
	// ErrEmptyXDSConfigSource means the XDS config source is empty.
	ErrEmptyXDSConfigSource = errors.New("empty xds config source, --xds-config-source option is required")
	// ErrEmptyNacosSource means nacos source is empty
	ErrEmptyNacosSource = errors.New("empty nacos source, --nacos-source option is required")

	// DefaultGRPCListen is the default gRPC server listen address.
	DefaultGRPCListen = "127.0.0.1:2379"
	// DefaultEtcdKeyPrefix is the default key prefix in the mimicking
	// etcd v3 server.
	DefaultEtcdKeyPrefix = "/apisix"
)

// RunningContext contains data which can be decided only when running.
type RunningContext struct {
	// PodNamespace is the namesapce of the resident pod.
	PodNamespace string
	// The IP address of the resident pod.
	IPAddress string
}

// Config contains configurations required for running apisix-mesh-agent.
type Config struct {
	// Running Id of this instance, it will be filled by
	// a random string when the instance started.
	RunId string
	// The minimum log level that will be printed.
	LogLevel string `json:"log_level" yaml:"log_level"`
	// The destination of logs.
	LogOutput string `json:"log_output" yaml:"log_output"`
	// The Provisioner to use.
	// Value can be "xds-v3-file", "xds-v3-grpc", "nacos".
	Provisioner string `json:"provisioner" yaml:"provisioner"`
	// The watched xds files, only valid if the Provisioner is "xds-v3-file"
	XDSWatchFiles   []string `json:"xds_watch_files" yaml:"xds_watch_files"`
	// XDSConfigSource only valid if the Provisioner is "xds-v3-grpc"
	XDSConfigSource string   `json:"xds_config_source" yaml:"xds_config_source"`
	// NacosSource should have format: SCHEME://URL:PORT/CONTEXT_PATH, for example: http://localhost:8848/nacos
	NacosSource string `json:"nacos_source" yaml:"nacos_source"`
	// The grpc listen address
	GRPCListen string `json:"grpc_listen" yaml:"grpc_listen"`
	// The key prefix in the mimicking etcd v3 server.
	EtcdKeyPrefix string `json:"etcd_key_prefix" yaml:"etcd_key_prefix"`
	// THe running mode of apisix-mesh-agent, can be:
	// 1. standalone - only launch apisix-mesh-agent
	// 2. bundle - launch apisix-mesh-agent and apisix, in such case,
	// correct apisix home path and bin path should be configured.
	// And when you shutdown apisix-mesh-agent, APISIX will also be closed.
	RunMode string `json:"run_mode" yaml:"run_mode"`
	// The home path of Apache APISIX.
	APISIXHomePath string `json:"apisix_home_path" yaml:"apisix_home_path"`
	// The executable binary path of Apache APISIX.
	APISIXBinPath string `json:"apisix_bin_path" yaml:"apisix_bin_path"`

	// RunningContext is the running context, it's self-contained.
	// TODO: Move it outside here since it doesn't belong to "configuration".
	RunningContext *RunningContext `json:"running_context" yaml:"running_context"`
}

// NewDefaultConfig returns a Config object with all items filled by
// their default values.
func NewDefaultConfig() *Config {
	return &Config{
		RunId:          uuid.NewString(),
		LogLevel:       "info",
		LogOutput:      "stderr",
		Provisioner:    XDSV3FileProvisioner,
		GRPCListen:     DefaultGRPCListen,
		EtcdKeyPrefix:  DefaultEtcdKeyPrefix,
		APISIXHomePath: DefaultAPISIXHomePath,
		APISIXBinPath:  DefaultAPISIXBinPath,
		RunMode:        StandaloneMode,

		RunningContext: getRunningContext(),
	}
}

// Validate validates the config object.
func (cfg *Config) Validate() error {
	if cfg.Provisioner == "" {
		return errors.New("unspecified provisioner")
	}
	if cfg.Provisioner != XDSV3FileProvisioner && cfg.Provisioner != XDSV3GRPCProvisioner && cfg.Provisioner != NacosProvisioner {
		return ErrUnknownProvisioner
	}
	if cfg.Provisioner == XDSV3GRPCProvisioner && cfg.XDSConfigSource == "" {
		return ErrEmptyXDSConfigSource
	}
	if cfg.Provisioner == NacosProvisioner && cfg.NacosSource == ""{
		return ErrEmptyNacosSource
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

func getRunningContext() *RunningContext {
	namespace := "default"
	if value := os.Getenv("POD_NAMESPACE"); value != "" {
		namespace = value
	}

	var (
		ipAddr string
	)
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}
	for _, iface := range ifaces {
		if iface.Name != "lo" {
			addrs, err := iface.Addrs()
			if err != nil {
				panic(err)
			}
			if len(addrs) > 0 {
				ipAddr = strings.Split(addrs[0].String(), "/")[0]
			}
		}
	}
	if ipAddr == "" {
		ipAddr = "127.0.0.1"
	}
	return &RunningContext{
		PodNamespace: namespace,
		IPAddress:    ipAddr,
	}
}
