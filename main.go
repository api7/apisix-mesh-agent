package main

import (
	"fmt"
	"os"

	"github.com/api7/apisix-mesh-agent/cmd"
)

func main() {
	rootCmd := cmd.NewMeshAgentCommand()
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
