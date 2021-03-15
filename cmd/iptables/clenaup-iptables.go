package iptables

import (
	"github.com/api7/apisix-mesh-agent/pkg/types"
	"github.com/spf13/cobra"
	"istio.io/istio/tools/istio-iptables/pkg/dependencies"
)

// NewCleanupIptablesCommand creates the cleanup-iptables sub-command object.
func NewCleanupIptablesCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "cleanup-iptables [flags]",
		Short: "Cleanup iptables rules for the port forwarding",
		Run: func(cmd *cobra.Command, args []string) {
			cleanup(dryRun)
		},
	}
	cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "dry run mode")
	return cmd
}

func cleanup(dryRun bool) {
	var ext dependencies.Dependencies
	if dryRun {
		ext = &dependencies.StdoutStubDependencies{}
	} else {
		ext = &dependencies.RealDependencies{}
	}
	removeOldChains(ext, "iptables")
}

func removeOldChains(ext dependencies.Dependencies, cmd string) {
	ext.RunQuietlyAndIgnore(cmd, "-t", "nat", "-D", types.PreRoutingChain, "-p", "tcp", "-j", types.InboundChain)
	ext.RunQuietlyAndIgnore(cmd, "-t", "nat", "-D", types.OutputChain, "-p", "tcp", "-j", types.OutputChain)
	flushAndDeleteChains(ext, cmd, "nat", []string{types.InboundChain, types.OutputChain, types.RedirectChain, types.InboundRedirectChain})
}

func flushAndDeleteChains(ext dependencies.Dependencies, cmd string, table string, chains []string) {
	for _, chain := range chains {
		ext.RunQuietlyAndIgnore(cmd, "-t", table, "-F", chain)
		ext.RunQuietlyAndIgnore(cmd, "-t", table, "-X", chain)
	}
}
