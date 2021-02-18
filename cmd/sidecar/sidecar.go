// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

			stop := make(chan struct{})
			waitForSignal(stop)
		},
	}

	cmd.PersistentFlags().StringVar(&cfg.LogOutput, "log-output", "stderr", "the output file path of error log")
	cmd.PersistentFlags().StringVar(&cfg.LogLevel, "log-level", "info", "the error log level")
	return cmd
}
