package precheck

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/api7/apisix-mesh-agent/pkg/types"

	"github.com/spf13/cobra"
)

var (
	// Use variable so unit test cases can change it, as there is no
	// iptables utility on macos.
	_iptablesCmd = "iptables"
)

// NewCommand creates the precheck subcommand object.
func NewCommand() *cobra.Command {
	var (
		apisixBinPath  string
		apisixHomePath string
	)

	cmd := &cobra.Command{
		Use:   "precheck [flags]",
		Short: "Check the running environment for Apache APISIX as the sidecar",
		Long: `Check the running environment for Apache APISIX as the sidecar.

if you just run apisix-mesh-agent in standalone mode, then don't run this precheck as it reports
false positive errors.`,
		Run: func(cmd *cobra.Command, args []string) {
			var code int
			if !check(apisixBinPath, apisixHomePath) {
				code = 1
			}
			os.Exit(code)
		},
	}

	cmd.PersistentFlags().StringVar(&apisixBinPath, "apisix-bin-path", "/usr/local/bin/apisix", "the executable binary file path of Apache APISIX")
	cmd.PersistentFlags().StringVar(&apisixHomePath, "apisix-home-path", "/usr/local/apisix", "the home path of Apache APISIX")
	return cmd
}

func check(bin, home string) bool {
	var buffer strings.Builder
	defer func() {
		fmt.Fprint(os.Stderr, buffer.String())
	}()

	if !checkBin(&buffer, bin) {
		return false
	}
	if !checkHome(&buffer, home) {
		return false
	}
	if !checkIptables(&buffer) {
		return false
	}
	return true
}

func checkBin(buffer *strings.Builder, path string) bool {
	defer func() {
		buffer.WriteByte('\n')
	}()
	buffer.WriteString("checking apisix binary path ")
	buffer.WriteString(path)
	buffer.WriteString(" ... ")
	_, err := os.Stat(path)
	if err != nil {
		buffer.WriteString(err.Error())
		return false
	}
	buffer.WriteString("found")
	return true
}

func checkHome(buffer *strings.Builder, path string) bool {
	defer func() {
		buffer.WriteByte('\n')
	}()
	buffer.WriteString("checking apisix home path ")
	buffer.WriteString(path)
	buffer.WriteString(" ... ")
	s, err := os.Stat(path)
	if err != nil {
		buffer.WriteString(err.Error())
		return false
	}
	if !s.IsDir() {
		buffer.WriteString("not a directory")
		return false
	}
	buffer.WriteString("found")
	return true
}

func checkIptables(buffer *strings.Builder) bool {
	checkChain := func(table, chain string) error {
		cmd := exec.Command(_iptablesCmd, "-t", table, "-L", chain)
		return cmd.Run()
	}
	for _, chain := range []string{types.InboundChain, types.OutputChain, types.RedirectChain, types.PreRoutingChain} {
		buffer.WriteString("checking iptables, table: nat, chain: ")
		buffer.WriteString(chain)
		buffer.WriteString(" ... ")
		err := checkChain("nat", chain)
		if err != nil {
			buffer.WriteString(err.Error())
			buffer.WriteByte('\n')
			return false
		} else {
			buffer.WriteString("found\n")
		}
	}
	return true
}
