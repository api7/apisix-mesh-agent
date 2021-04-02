package util

import (
	"strings"
)

// GenNodeId generates an id used for xDS protocol. The format is like:
// sidecar~172.10.0.2~12345asad034~default.svc.cluster.local
func GenNodeId(runId, ipAddr, dnsDomain string) string {
	var buf strings.Builder
	buf.WriteString("sidecar~")
	buf.WriteString(ipAddr)
	buf.WriteString("~")
	buf.WriteString(runId)
	buf.WriteString("~")
	buf.WriteString(dnsDomain)
	return buf.String()
}
