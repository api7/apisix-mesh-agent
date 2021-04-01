package util

import (
	"net"
	"strings"
)

var (
	_ipAddr = "127.0.0.1"
)

func init() {
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
				_ipAddr = addrs[0].String()
			}
		}
	}
}

// GenNodeId generates an id used for xDS protocol. The format is like:
// sidecar~172.10.0.2~12345asad034~default.svc.cluster.local
func GenNodeId(runId, dnsDomain string) string {
	var buf strings.Builder
	buf.WriteString("sidecar~")
	buf.WriteString(_ipAddr)
	buf.WriteString("~")
	buf.WriteString(runId)
	buf.WriteString("~")
	buf.WriteString(dnsDomain)
	return buf.String()
}
