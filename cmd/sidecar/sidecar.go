package sidecar

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/api7/apisix-mesh-agent/pkg/config"
	"github.com/api7/apisix-mesh-agent/pkg/log"
	"github.com/api7/apisix-mesh-agent/pkg/sidecar"
	"github.com/api7/apisix-mesh-agent/pkg/version"
)

func dief(template string, args ...interface{}) {
	if !strings.HasSuffix(template, "\n") {
		template += "\n"
	}
	_, _ = fmt.Fprintf(os.Stderr, template, args...)
	os.Exit(1)
}

func initializeDefaultLogger(cfg *config.Config) {
	logger, err := log.NewLogger(
		log.WithLogLevel(cfg.LogLevel),
		log.WithOutputFile(cfg.LogOutput),
	)
	if err != nil {
		dief("failed to initialize logging: %s", err)
	}
	log.DefaultLogger = logger
}

func waitForSignal(stopCh chan struct{}) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	log.Infof("signal %d (%s) received", sig, sig.String())
	close(stopCh)
}

// NewCommand creates the sidecar command for apisix-mesh-agent.
func NewCommand() *cobra.Command {
	cfg := config.NewDefaultConfig()
	cmd := &cobra.Command{
		Use:   "sidecar [flags]",
		Short: "Launch apisix-mesh-agent as a sidecar process",
		Run: func(cmd *cobra.Command, args []string) {
			initializeDefaultLogger(cfg)
			log.Infow("apisix-mesh-agent started")
			defer log.Info("apisix-mesh-agent exited")
			log.Info("version:\n", version.String())
			data, err := json.MarshalIndent(cfg, "", "    ")
			if err != nil {
				panic(err)
			}
			log.Info("use configuration:\n", string(data))

			sc, err := sidecar.NewSidecar(cfg)
			if err != nil {
				dief("failed to initialize: %s", err)
			}

			stop := make(chan struct{})
			go func() {
				if err := sc.Run(stop); err != nil {
					panic(err)
				}
			}()

			waitForSignal(stop)
		},
	}

	cmd.PersistentFlags().StringVar(&cfg.LogOutput, "log-output", "stderr", "the output file path of error log")
	cmd.PersistentFlags().StringVar(&cfg.LogLevel, "log-level", "info", "the error log level")
	cmd.PersistentFlags().StringVar(&cfg.Provisioner, "provisioner", config.XDSV3FileProvisioner, "the provisioner to use, option can be \"xds-v3-file\"")
	cmd.PersistentFlags().StringSliceVar(&cfg.XDSWatchFiles, "xds-watch-files", nil, "file paths watched by xds-v3-file provisioner")
	cmd.PersistentFlags().StringVar(&cfg.GRPCListen, "grpc-listen", config.DefaultGRPCListen, "grpc server listen address")
	cmd.PersistentFlags().StringVar(&cfg.EtcdKeyPrefix, "etcd-key-prefix", config.DefaultEtcdKeyPrefix, "the key prefix in the mimicking etcd v3 server")
	return cmd
}
