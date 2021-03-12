package cmd

import (
	"github.com/spf13/cobra"

	"github.com/api7/apisix-mesh-agent/cmd/iptables"
	"github.com/api7/apisix-mesh-agent/cmd/precheck"
	"github.com/api7/apisix-mesh-agent/cmd/sidecar"
	"github.com/api7/apisix-mesh-agent/cmd/version"
)

// NewMeshAgentCommand creates the root command for apisix-mesh-agent.
func NewMeshAgentCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apisix-mesh-agent [command] [flags]",
		Short: "Agent of Apache APISIX to extend it as a Service Mesh Sidecar.",
	}
	cmd.AddCommand(
		sidecar.NewCommand(),
		version.NewCommand(),
		precheck.NewCommand(),
		iptables.NewSetupCommand(),
		iptables.NewCleanupIptablesCommand(),
	)
	return cmd
}
