package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/api7/apisix-mesh-agent/pkg/version"
)

// NewCommand creates the version command for apisix-mesh-agent.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Version for apisix-mesh-agent",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.String())
		},
	}
	return cmd
}
