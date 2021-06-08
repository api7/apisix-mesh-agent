package constant

import "os"

var (
	Helm              = "helm"
	DefaultKubeconfig = "~/.kube/config"
	E2eHome           = os.Getenv("APISIX_MESH_AGENT_E2E_HOME")
)
